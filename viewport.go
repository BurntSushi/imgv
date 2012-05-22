package main

import (
	"image"

	"github.com/BurntSushi/xgbutil/xgraphics"
)

// vpCenter inspects the canvas and image geometry, and determines where the
// origin of the image should be painted into the canvas.
// If the image is bigger than the canvas, this is always (0, 0).
// If the image is the same size, then it is also (0, 0).
// If a dimension of the image is smaller than the canvas, then:
// x = (canvas_width - image_width) / 2 and
// y = (canvas_height - image_height) / 2
func vpCenter(ximg *xgraphics.Image, canWidth, canHeight int) image.Point {
	return image.Point{vpXMargin(ximg, canWidth), vpYMargin(ximg, canHeight)}
}

func vpXMargin(ximg *xgraphics.Image, canWidth int) int {
	if ximg.Bounds().Dx() < canWidth {
		return (canWidth - ximg.Bounds().Dx()) / 2
	}
	return 0
}

func vpYMargin(ximg *xgraphics.Image, canHeight int) int {
	if ximg.Bounds().Dy() < canHeight {
		return (canHeight - ximg.Bounds().Dy()) / 2
	}
	return 0
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
