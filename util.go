package main

import (
	"image"
	"strings"

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
	xmargin, ymargin := 0, 0
	if ximg.Bounds().Dx() < canWidth {
		xmargin = (canWidth - ximg.Bounds().Dx()) / 2
	}
	if ximg.Bounds().Dy() < canHeight {
		ymargin = (canHeight - ximg.Bounds().Dy()) / 2
	}
	return image.Point{xmargin, ymargin}
}

// basename retrieves the basename of a file path.
func basename(fName string) string {
	if lslash := strings.LastIndex(fName, "/"); lslash != -1 {
		fName = fName[lslash+1:]
	}
	return fName
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
