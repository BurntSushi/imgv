package main

import (
	"fmt"
	"image"

	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/xgraphics"
)

type State struct {
	win       *window
	imgs      []Image
	img       Image
	ximg      *xgraphics.Image
	imgOrigin image.Point
	panStart  image.Point
	panOrigin image.Point
}

func newState(X *xgbutil.XUtil, imgs []Image) *State {
	return &State{
		win:       newWindow(X),
		imgs:      imgs,
		imgOrigin: image.Point{0, 0},
	}
}

func (s *State) image() *xgraphics.Image {
	sx, sy := s.imgOrigin.X, s.imgOrigin.Y
	sub := image.Rect(sx, sy,
		sx+s.win.Geom.Width(), sy+s.win.Geom.Height())
	return state.ximg.SubImage(sub)
}

func (s *State) imageSet(img Image, size int) {
	s.img = s.imgs[0]
	s.ximg = state.img.sizes[100]
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
