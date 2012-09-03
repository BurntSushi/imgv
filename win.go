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

// keyb represents a value in the keybinding list. Namely, it contains the
// function to run when a particular key sequence has been pressed, the
// key sequence to bind to, and a quick description of what the keybinding
// actually does.
type keyb struct {
	key    string
	desc   string
	action func(w *window)
}

// window embeds an xwindow.Window value and all available channels used to
// communicate with the canvas.
// While the canvas and the window are essentialy the same, the canvas
// focuses on the abstraction of drawing some image into a viewport while the
// window focuses on the more X related aspects of setting up the canvas.
type window struct {
	*xwindow.Window
	chans chans
}

// newWndow creates a new window and dies on failure.
// This includes mapping the window but not setting up the event handlers.
// (The event handlers require the channels, and we don't create the channels
// until all images have been decoded. But we want to show the window to the
// user before that task is complete.)
func newWindow(X *xgbutil.XUtil) *window {
	xwin, err := xwindow.Generate(X)
	if err != nil {
		errLg.Fatalf("Could not create window: %s", err)
	}

	w := &window{
		Window: xwin,
	}
	w.create()

	return w
}

// create creates the window, initializes the keybind and mousebind packages
// and sets up the window to act like a real top-level client.
func (w *window) create() {
	keybind.Initialize(w.X)
	mousebind.Initialize(w.X)

	err := w.CreateChecked(w.X.RootWin(), 0, 0, flagWidth, flagHeight,
		xproto.CwBackPixel, 0xffffff)
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
	w.nameSet("Decoding all images...")

	w.Map()
}

// stepLeft moves the origin of the image to the left.
func (w *window) stepLeft() {
	w.chans.drawChan <- func(origin image.Point) image.Point {
		return image.Point{origin.X - flagStepIncrement, origin.Y}
	}
}

// stepRight moves the origin of the image to the right.
func (w *window) stepRight() {
	w.chans.drawChan <- func(origin image.Point) image.Point {
		return image.Point{origin.X + flagStepIncrement, origin.Y}
	}
}

// stepUp moves the origin of the image down (this would be up, but X origins
// are in the top-left corner).
func (w *window) stepUp() {
	w.chans.drawChan <- func(origin image.Point) image.Point {
		return image.Point{origin.X, origin.Y - flagStepIncrement}
	}
}

// stepDown moves the origin of the image up (this would be down, but X origins
// are in the top-left corner).
func (w *window) stepDown() {
	w.chans.drawChan <- func(origin image.Point) image.Point {
		return image.Point{origin.X, origin.Y + flagStepIncrement}
	}
}

// paint uses the xgbutil/xgraphics package to copy the area corresponding
// to ximg in its pixmap to the window. It will also issue a clear request
// before hand to try and avoid artifacts.
func (w *window) paint(ximg *xgraphics.Image) {
	dst := vpCenter(ximg, w.Geom.Width(), w.Geom.Height())
	// UUU Commenting this out avoids flickering, and I see no artifacts!
	// w.ClearAll() 
	ximg.XExpPaint(w.Id, dst.X, dst.Y)
}

// nameSet will set the name of the window and emit a benign message to
// verbose output if it fails.
func (w *window) nameSet(name string) {
	// Set _NET_WM_NAME so it looks nice.
	err := ewmh.WmNameSet(w.X, w.Id, fmt.Sprintf("imgv :: %s", name))
	if err != nil { // not a fatal error
		lg("Could not set _NET_WM_NAME: %s", err)
	}
}

// setupEventHandlers attaches the canvas' channels to the window and
// sets the appropriate callbacks to some events:
// ConfigureNotify events will cause the window to update its state of geometry.
// Expose events will cause the window to repaint the current image.
// Button events to allow panning.
// Key events to perform various tasks when certain keys are pressed. Should
// these be configurable? Meh.
func (w *window) setupEventHandlers(chans chans) {
	w.chans = chans
	w.Listen(xproto.EventMaskStructureNotify | xproto.EventMaskExposure |
		xproto.EventMaskButtonPress | xproto.EventMaskButtonRelease |
		xproto.EventMaskKeyPress)

	// Get the current geometry in case we don't get a ConfigureNotify event
	// (or have already missed it).
	_, err := w.Geometry()
	if err != nil {
		errLg.Fatal(err)
	}

	// And ask the canvas to draw the first image when it gets around to it.
	go func() {
		w.chans.drawChan <- func(origin image.Point) image.Point {
			return image.Point{}
		}
	}()

	// Keep a state of window geometry.
	xevent.ConfigureNotifyFun(
		func(X *xgbutil.XUtil, ev xevent.ConfigureNotifyEvent) {
			w.Geom.WidthSet(int(ev.Width))
			w.Geom.HeightSet(int(ev.Height))
		}).Connect(w.X, w.Id)

	// Repaint the window on expose events.
	xevent.ExposeFun(
		func(X *xgbutil.XUtil, ev xevent.ExposeEvent) {
			w.chans.drawChan <- func(origin image.Point) image.Point {
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
	// for keystring, fun := range kbs { 
	for _, keyb := range keybinds {
		keyb := keyb
		err := keybind.KeyPressFun(
			func(X *xgbutil.XUtil, ev xevent.KeyPressEvent) {
				keyb.action(w)
			}).Connect(w.X, w.Id, keyb.key, false)
		if err != nil {
			errLg.Println(err)
		}
	}
}
