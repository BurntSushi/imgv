package main

import (
	"fmt"
	"image"

	"github.com/BurntSushi/xgbutil"
)

type chans struct {
	imgChan           chan imageLoaded
	drawChan          chan func(pt image.Point) image.Point
	resizeToImageChan chan struct{}
	prevImg           chan struct{}
	nextImg           chan struct{}

	imgLoadChans []chan struct{}

	panStartChan chan image.Point
	panStepChan  chan image.Point
	panEndChan   chan image.Point
}

type imageLoaded struct {
	img   *Image
	index int
}

func canvas(X *xgbutil.XUtil, window *window, names []string, nimgs int) chans {
	imgChan := make(chan imageLoaded, 0)
	drawChan := make(chan func(pt image.Point) image.Point, 0)
	resizeToImageChan := make(chan struct{}, 0)
	prevImg := make(chan struct{}, 0)
	nextImg := make(chan struct{}, 0)

	imgLoadChans := make([]chan struct{}, nimgs)
	for i := range imgLoadChans {
		imgLoadChans[i] = make(chan struct{}, 0)
	}

	panStartChan := make(chan image.Point, 0)
	panStepChan := make(chan image.Point, 0)
	panEndChan := make(chan image.Point, 0)
	panStart, panOrigin := image.Point{}, image.Point{}

	chans := chans{
		imgChan:           imgChan,
		drawChan:          drawChan,
		resizeToImageChan: resizeToImageChan,
		prevImg:           prevImg,
		nextImg:           nextImg,

		imgLoadChans: imgLoadChans,

		panStartChan: panStartChan,
		panStepChan:  panStepChan,
		panEndChan:   panEndChan,
	}

	imgs := make([]*Image, nimgs)
	window.setupEventHandlers(chans)
	current := 0
	origin := image.Point{0, 0}

	setOrigin := func(org image.Point) {
		origin = originTrans(org, window, imgs[current])
	}
	setImage := func(i int, pt image.Point) {
		if i >= len(imgs) {
			i = 0
		}
		if i < 0 {
			i = len(imgs) - 1
		}

		current = i
		if imgs[i] == nil {
			window.nameSet(fmt.Sprintf("%s - Loading...", names[i]))
			window.ClearAll()

			if imgLoadChans[i] != nil {
				imgLoadChans[i] <- struct{}{}
				imgLoadChans[i] = nil
			}
			return
		}

		setOrigin(pt)
		show(window, imgs[i], origin)
	}

	go func() {
		for {
			select {
			case img := <-imgChan:
				imgs[img.index] = img.img

				// If this is the current image, show it!
				if current == img.index {
					show(window, imgs[current], origin)
				}
			case funpt := <-drawChan:
				setImage(current, funpt(origin))
			case <-resizeToImageChan:
				window.Resize(imgs[current].Bounds().Dx(),
					imgs[current].Bounds().Dy())
			case <-prevImg:
				setImage(current-1, image.Point{0, 0})
			case <-nextImg:
				setImage(current+1, image.Point{0, 0})
			case pt := <-panStartChan:
				panStart = pt
				panOrigin = origin
			case pt := <-panStepChan:
				xd, yd := panStart.X-pt.X, panStart.Y-pt.Y
				setImage(current,
					image.Point{xd + panOrigin.X, yd + panOrigin.Y})
			case <-panEndChan:
				panStart, panOrigin = image.Point{}, image.Point{}
			}
		}
	}()

	return chans
}

// originTrans translates the origin with respect to the current image and the
// current canvas size. This makes sure we never incorrect position the image.
// (i.e., panning never goes too far, and whenever the canvas is bigger than
// the image, the origin is *always* (0, 0).
func originTrans(pt image.Point, win *window, img *Image) image.Point {
	// If there's no valid image, then always return (0, 0).
	if img == nil {
		return image.Point{0, 0}
	}

	// Quick aliases.
	ww, wh := win.Geom.Width(), win.Geom.Height()
	dw := img.Bounds().Dx() - ww
	dh := img.Bounds().Dy() - wh

	// Set the allowable range of the origin point of the image.
	// i.e., never less than (0, 0) and never greater than the width/height
	// of the image that isn't viewable at any given point (which is determined
	// by the canvas size).
	pt.X = min(img.Bounds().Min.X+dw, max(pt.X, 0))
	pt.Y = min(img.Bounds().Min.Y+dh, max(pt.Y, 0))

	// Validate origin point. If the width/height of an image is smaller than
	// the canvas width/height, then the image origin cannot change in x/y
	// direction.
	if img.Bounds().Dx() < ww {
		pt.X = 0
	}
	if img.Bounds().Dy() < wh {
		pt.Y = 0
	}

	return pt
}

// show translates the given origin point, paints the appropriate part of the
// current image to the canvas, and sets the name of the window.
// (Painting only paints the sub-image that is viewable.)
func show(win *window, img *Image, pt image.Point) {
	// If there's no valid image, don't bother trying to show it.
	// (We're hopefully loading the image now.)
	if img == nil {
		return
	}

	// Translate the origin to reflect the size of the image and canvas.
	pt = originTrans(pt, win, img)

	// Now paint the sub-image to the window.
	win.paint(img.SubImage(image.Rect(pt.X, pt.Y,
		pt.X+win.Geom.Width(), pt.Y+win.Geom.Height())))

	// Always set the name of the window when we update it with a new image.
	win.nameSet(fmt.Sprintf("%s (%dx%d)",
		img.name, img.Bounds().Dx(), img.Bounds().Dy()))
}
