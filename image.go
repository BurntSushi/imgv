package main

import (
	"image"
	"time"

	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/xgraphics"
)

type Image struct {
	*xgraphics.Image
	name string
}

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
	imgChan <- loaded
}
