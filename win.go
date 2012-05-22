package main

import (
	"fmt"
	"image"

	"github.com/BurntSushi/xgb/xproto"

	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/ewmh"
	"github.com/BurntSushi/xgbutil/icccm"
	"github.com/BurntSushi/xgbutil/keybind"
	"github.com/BurntSushi/xgbutil/mousebind"
	"github.com/BurntSushi/xgbutil/xevent"
	"github.com/BurntSushi/xgbutil/xgraphics"
	"github.com/BurntSushi/xgbutil/xwindow"
)

type window struct {
	*xwindow.Window
	chans chans
}

func newWindow(X *xgbutil.XUtil, chans chans) *window {
	xwin, err := xwindow.Generate(X)
	if err != nil {
		errLg.Fatalf("Could not create window: %s", err)
	}

	w := &window{
		Window: xwin,
		chans:  chans,
	}
	w.create()

	return w
}

func (w *window) create() {
	keybind.Initialize(w.X)
	mousebind.Initialize(w.X)

	err := w.CreateChecked(w.X.RootWin(), 0, 0, flagWidth, flagHeight,
		xproto.CwBackPixel|xproto.CwEventMask,
		0xffffff,
		xproto.EventMaskStructureNotify|xproto.EventMaskExposure|
			xproto.EventMaskButtonPress|xproto.EventMaskButtonRelease|
			xproto.EventMaskKeyPress)
	if err != nil {
		errLg.Fatalf("Could not create window: %s", err)
	}

	// Make the window close gracefully using the WM_DELETE_WINDOW protocol.
	w.WMGracefulClose(func(w *xwindow.Window) {
		xevent.Detach(w.X, w.Id)
		keybind.Detach(w.X, w.Id)
		mousebind.Detach(w.X, w.Id)
		w.Destroy()
		xevent.Quit(w.X)
	})

	// Set WM_STATE so it is interpreted as top-level and is mapped.
	err = icccm.WmStateSet(w.X, w.Id, &icccm.WmState{
		State: icccm.StateNormal,
	})
	if err != nil { // not a fatal error
		lg("Could not set WM_STATE: %s", err)
	}

	// _NET_WM_STATE = _NET_WM_STATE_NORMAL
	ewmh.WmStateSet(w.X, w.Id, []string{"_NET_WM_STATE_NORMAL"})

	// Set the name to something.
	w.nameSet("Loading...")

	w.setupEventHandlers()

	w.Map()
}

func (w *window) resizeToImage() {
	w.chans.resizeToImageChan <- struct{}{}
}

func (w *window) stepLeft() {
	w.chans.funDrawChan <- func(origin image.Point) image.Point {
		return image.Point{origin.X - flagStepIncrement, origin.Y}
	}
}

func (w *window) stepRight() {
	w.chans.funDrawChan <- func(origin image.Point) image.Point {
		return image.Point{origin.X + flagStepIncrement, origin.Y}
	}
}

func (w *window) stepUp() {
	w.chans.funDrawChan <- func(origin image.Point) image.Point {
		return image.Point{origin.X, origin.Y - flagStepIncrement}
	}
}

func (w *window) stepDown() {
	w.chans.funDrawChan <- func(origin image.Point) image.Point {
		return image.Point{origin.X, origin.Y + flagStepIncrement}
	}
}

func (w *window) paint(ximg *xgraphics.Image) {
	dst := vpCenter(ximg, w.Geom.Width(), w.Geom.Height())
	w.ClearAll()
	ximg.XExpPaint(w.Id, dst.X, dst.Y)
}

func (w *window) nameSet(name string) {
	// Set _NET_WM_NAME so it looks nice.
	err := ewmh.WmNameSet(w.X, w.Id, fmt.Sprintf("imgv :: %s", name))
	if err != nil { // not a fatal error
		lg("Could not set _NET_WM_NAME: %s", err)
	}
}

func (w *window) setupEventHandlers() {
	// Keep a state of window geometry.
	xevent.ConfigureNotifyFun(
		func(X *xgbutil.XUtil, ev xevent.ConfigureNotifyEvent) {
			w.Geom.WidthSet(int(ev.Width))
			w.Geom.HeightSet(int(ev.Height))
		}).Connect(w.X, w.Id)

	// Repaint the window on expose events.
	xevent.ExposeFun(
		func(X *xgbutil.XUtil, ev xevent.ExposeEvent) {
			w.chans.funDrawChan <- func(origin image.Point) image.Point {
				return origin
			}
		}).Connect(w.X, w.Id)

	// Setup a drag handler to allow panning.
	mousebind.Drag(w.X, w.Id, w.Id, "1", false,
		func(X *xgbutil.XUtil, rx, ry, ex, ey int) (bool, xproto.Cursor) {
			w.chans.panStartChan <- image.Point{ex, ey}
			return true, 0
		},
		func(X *xgbutil.XUtil, rx, ry, ex, ey int) {
			w.chans.panStepChan <- image.Point{ex, ey}
		},
		func(X *xgbutil.XUtil, rx, ry, ex, ey int) {
			w.chans.panEndChan <- image.Point{ex, ey}
		})

	// Set up a map of keybindings to avoid a lot of boiler plate.
	kbs := map[string]func(){
		"left":    func() { w.chans.prevImg <- struct{}{} },
		"right":   func() { w.chans.nextImg <- struct{}{} },
		"shift-h": func() { w.chans.prevImg <- struct{}{} },
		"shift-l": func() { w.chans.nextImg <- struct{}{} },

		"r": func() { w.resizeToImage() },

		"h": func() { w.stepLeft() },
		"j": func() { w.stepDown() },
		"k": func() { w.stepUp() },
		"l": func() { w.stepRight() },

		"q": func() { xevent.Quit(w.X) },
	}
	for keystring, fun := range kbs {
		fun := fun
		err := keybind.KeyPressFun(
			func(X *xgbutil.XUtil, ev xevent.KeyPressEvent) {
				fun()
			}).Connect(w.X, w.Id, keystring, false)
		if err != nil {
			errLg.Println(err)
		}
	}
}
