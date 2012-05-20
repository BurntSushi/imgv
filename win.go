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
	"github.com/BurntSushi/xgbutil/xwindow"
)

type window struct {
	*xwindow.Window
}

func newWindow(X *xgbutil.XUtil) *window {
	xwin, err := xwindow.Generate(X)
	if err != nil {
		errLg.Fatalf("Could not create window: %s", err)
	}

	w := &window{xwin}
	w.create()

	return w
}

func (w *window) create() {
	err := w.CreateChecked(w.X.RootWin(), 0, 0, flagWidth, flagHeight,
		xproto.CwBackPixel|xproto.CwEventMask,
		0xffffff,
		xproto.EventMaskStructureNotify|xproto.EventMaskExposure|
			xproto.EventMaskButtonPress|xproto.EventMaskButtonRelease)
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

	// Set up some callback handlers.
	xevent.ConfigureNotifyFun(
		func(X *xgbutil.XUtil, ev xevent.ConfigureNotifyEvent) {
			w.Geom.WidthSet(int(ev.Width))
			w.Geom.HeightSet(int(ev.Height))
		}).Connect(w.X, w.Id)
	xevent.ExposeFun(
		func(X *xgbutil.XUtil, ev xevent.ExposeEvent) {
			// w.drawImage() 
			state.originSet(state.imgOrigin)
		}).Connect(w.X, w.Id)
	mousebind.Drag(w.X, w.Id, w.Id, "1", false,
		func(X *xgbutil.XUtil, rx, ry, ex, ey int) (bool, xproto.Cursor) {
			w.panStart(ex, ey)
			return true, 0
		},
		func(X *xgbutil.XUtil, rx, ry, ex, ey int) {
			w.panStep(ex, ey)
		},
		func(X *xgbutil.XUtil, rx, ry, ex, ey int) {
			w.panEnd(ex, ey)
		})

	w.Map()
}

func (w *window) drawImage() {
	ximg := state.image()
	dst := vpCenter(ximg)
	w.ClearAll()
	ximg.XExpPaint(w.Id, dst.X, dst.Y)
}

func (w *window) panStart(x, y int) {
	state.panStart = image.Point{x, y}
	state.panOrigin = state.imgOrigin
}

func (w *window) panStep(x, y int) {
	xd, yd := state.panStart.X-x, state.panStart.Y-y
	ox, oy := state.panOrigin.X, state.panOrigin.Y
	state.originSet(image.Point{xd + ox, yd + oy})
}

func (w *window) panEnd(x, y int) {
	state.panStart = image.Point{}
	state.panOrigin = image.Point{}
}

func (w *window) nameSet(name string) {
	// Set _NET_WM_NAME so it looks nice.
	err := ewmh.WmNameSet(w.X, w.Id, fmt.Sprintf("imgv :: %s", name))
	if err != nil { // not a fatal error
		lg("Could not set _NET_WM_NAME: %s", err)
	}
}
