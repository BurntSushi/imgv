package main

import (
	"image"
	"time"

	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/xgraphics"
)

// Image acts as an xgraphics.Image type with a name.
// (The name is the basename of the image's corresponding file name.)
type Image struct {
	*xgraphics.Image
	name string
}

// newImage is meant to be run as a goroutine and loads a decoded image into
// an xgraphics.Image value and draws it to an X pixmap.
// The loading doesn't start until this image's corresponding imgLoadChan
// has been pinged.
// This implies that all images are decoded on start-up and are converted
// and drawn to an X pixmap on-demand. I am still deliberating on whether this
// is a smart decision.
// Note that this process, particularly image conversion, can be quite
// costly for large images.
func newImage(X *xgbutil.XUtil, name string, img image.Image, index int,
	imgLoadChan chan struct{}, imgChan chan imageLoaded) {

	// Don't start loading until we're told to do so.
	<-imgLoadChan

	// We send this when we're done processing this image, whether its
	// an error or not.
	loaded := imageLoaded{index: index}

	start := time.Now()
	reg := xgraphics.NewConvert(X, img)
	lg("Converted '%s' to xgraphics.Image type (%s).", name, time.Since(start))

	if err := reg.CreatePixmap(); err != nil {
		// TODO: We should display a "Could not load image" image instead
		// of dying. However, creating a pixmap rarely fails, unless we have
		// a *ton* of images. (In all likelihood, we'll run out of memory
		// before a new pixmap cannot be created.)
		errLg.Fatal(err)
	} else {
		start = time.Now()
		reg.XDraw()
		lg("Drawn '%s' to an X pixmap (%s).", name, time.Since(start))
	}

	loaded.img = &Image{
		Image: reg,
		name:  name,
	}

	// Tell the canvas that this image has been loaded.
	imgChan <- loaded
}
