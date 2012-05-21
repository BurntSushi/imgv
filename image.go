package main

import (
	"image"
	"strings"

	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/xgraphics"
)

type Image struct {
	image.Image
	name  string
	sizes map[int]*xgraphics.Image
}

func newImage(X *xgbutil.XUtil, fileName string, img image.Image) *Image {
	reg := xgraphics.NewConvert(X, img)

	imageSizes := make(map[int]*xgraphics.Image, len(sizes))
	has100 := false
	for _, size := range sizes {
		if size == 100 {
			imageSizes[size] = reg
			has100 = true
		} else {
			imageSizes[size] = nil
		}
	}
	if !has100 {
		errLg.Fatal("Could not find 100 in the list of sizes. This is " +
			"required for program function.")
	}

	// Create pixmaps for each size, and fill them in.
	for _, ximg := range imageSizes {
		if ximg == nil {
			continue
		}
		if err := ximg.CreatePixmap(); err != nil {
			errLg.Fatal(err)
		} else {
			ximg.XDraw()
		}
	}

	// Use the base name for this image as its name.
	name := fileName
	if lslash := strings.LastIndex(fileName, "/"); lslash != -1 {
		name = name[lslash+1:]
	}

	return &Image{
		Image: img,
		name:  name,
		sizes: imageSizes,
	}
}

func (im *Image) initializeSize(size int) {
	im.sizes[size] = xgraphics.NewConvert(state.win.X, im).Scale(
		(size*im.Bounds().Dx())/100,
		(size*im.Bounds().Dy())/100)
	if err := im.sizes[size].CreatePixmap(); err != nil {
		errLg.Fatal(err)
	} else {
		im.sizes[size].XDraw()
	}
}
