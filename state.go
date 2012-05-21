package main

import (
	"fmt"
	"image"

	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/xgraphics"
)

type State struct {
	win       *window
	imgs      []*Image
	img       *Image
	ximg      *xgraphics.Image
	size      int
	imgOrigin image.Point
	panStart  image.Point
	panOrigin image.Point
}

func newState(X *xgbutil.XUtil, imgs []*Image) *State {
	return &State{
		win:       newWindow(X),
		imgs:      imgs,
		size:      100,
		imgOrigin: image.Point{0, 0},
	}
}

func (s *State) image() *xgraphics.Image {
	sx, sy := s.imgOrigin.X, s.imgOrigin.Y
	sub := image.Rect(sx, sy,
		sx+s.win.Geom.Width(), sy+s.win.Geom.Height())
	return state.ximg.SubImage(sub)
}

func (s *State) imageSet(img *Image, size int) {
	s.img = img
	s.size = size
	s.ximg = state.img.sizes[s.size]
	if s.ximg == nil {
		img.initializeSize(s.size)
		s.ximg = state.img.sizes[s.size]
	}
	s.win.nameSet(fmt.Sprintf("%s (%dx%d)",
		s.img.name, s.img.Bounds().Dx(), s.img.Bounds().Dy()))
}

func (s *State) originSet(pt image.Point) {
	dw := s.ximg.Bounds().Dx() - s.win.Geom.Width()
	dh := s.ximg.Bounds().Dy() - s.win.Geom.Height()

	pt.X = min(s.ximg.Bounds().Min.X+dw, max(pt.X, 0))
	pt.Y = min(s.ximg.Bounds().Min.Y+dh, max(pt.Y, 0))

	// Valid origin point. If the width/height of an image is smaller than
	// the canvas width/height, then the image origin cannot change in x/y
	// direction.
	if s.ximg.Bounds().Dx() < s.win.Geom.Width() {
		pt.X = 0
	}
	if s.ximg.Bounds().Dy() < s.win.Geom.Height() {
		pt.Y = 0
	}
	s.imgOrigin = pt
	s.win.drawImage()
}

func (s *State) nextSize() int {
	// Find the current size in the available sizes.
	curi := -1
	for i, size := range sizes {
		if size == s.size {
			curi = i
			break
		}
	}
	if curi == -1 {
		errLg.Fatal("Could not find current size '%d' in list of available "+
			"sizes. Something has gone seriously wrong.", s.size)
	}
	if curi == len(sizes)-1 {
		return s.size
	}
	return sizes[curi+1]
}

func (s *State) prevSize() int {
	// Find the current size in the available sizes.
	curi := -1
	for i, size := range sizes {
		if size == s.size {
			curi = i
			break
		}
	}
	if curi == -1 {
		errLg.Fatal("Could not find current size '%d' in list of available "+
			"sizes. Something has gone seriously wrong.", s.size)
	}
	if curi == 0 {
		return s.size
	}
	return sizes[curi-1]
}
