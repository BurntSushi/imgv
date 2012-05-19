package main

import (
	"image"

	"github.com/BurntSushi/xgbutil/xgraphics"
)

type Image struct {
	orig  image.Image
	sizes map[int]*xgraphics.Image
}

func newImage(img image.Image) Image {
	reg := xgraphics.NewConvert(X, img)
	sizes := map[int]*xgraphics.Image{
		100: reg,
	}

	return Image{
		orig:  img,
		sizes: sizes,
	}
}
