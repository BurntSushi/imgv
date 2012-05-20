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

func newImage(X *xgbutil.XUtil, fileName string, img image.Image) Image {
	reg := xgraphics.NewConvert(X, img)
	sizes := map[int]*xgraphics.Image{
		100: reg,
	}

	// Create pixmaps for each size, and fill them in.
	for _, ximg := range sizes {
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

	return Image{
		Image: img,
		name:  name,
		sizes: sizes,
	}
}
