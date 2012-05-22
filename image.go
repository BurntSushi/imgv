package main

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/xgraphics"
)

type Image struct {
	*xgraphics.Image
	name string
}

func newImage(X *xgbutil.XUtil, fName string, index int,
	imgChan chan imageLoaded) {

	// We send this when we're done processing this image, whether its
	// an error or not.
	loaded := imageLoaded{index: index}

	// Use the base name for this image as its name.
	name := fName
	if lslash := strings.LastIndex(name, "/"); lslash != -1 {
		name = name[lslash+1:]
	}

	file, err := os.Open(fName)
	if err != nil {
		errLg.Println(err)
		imgChan <- loaded
		return
	}

	img, kind, err := image.Decode(file)
	if err != nil {
		errLg.Printf("Could not decode '%s' into a supported image "+
			"format: %s", fName, err)
		imgChan <- loaded
		return
	}
	lg("Decoded '%s' into image type '%s'.", name, kind)

	reg := xgraphics.NewConvert(X, img)
	lg("Converted '%s' to xgraphics.Image type.", name)
	if err := reg.CreatePixmap(); err != nil {
		errLg.Fatal(err)
	} else {
		reg.XDraw()
		lg("Drawn '%s' to an X pixmap.", name)
	}

	loaded.img = &Image{
		Image: reg,
		name:  name,
	}
	imgChan <- loaded
}
