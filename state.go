package main

import (
	"fmt"
	"image"

	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/xgraphics"
)

type State struct {
	win      *window
	imgChans []chan *Image
	imgs     []*Image
	imgi     int

	imgOrigin image.Point
	panStart  image.Point
	panOrigin image.Point
}

func newState(X *xgbutil.XUtil, win *window, imgChans []chan *Image,
	imgs []*Image) *State {

	return &State{
		win:       win,
		imgChans:  imgChans,
		imgs:      imgs,
		imgOrigin: image.Point{0, 0},
	}
}

func (s *State) ximage() *xgraphics.Image {
	sx, sy := s.imgOrigin.X, s.imgOrigin.Y
	sub := image.Rect(sx, sy,
		sx+s.win.Geom.Width(), sy+s.win.Geom.Height())
	return state.image().SubImage(sub)
}

func (s *State) image() *Image {
	return s.imgs[s.imgi]
}

func (s *State) imageSet(i int) {
	// Allow looping around the image list.
	if i < 0 {
		i = len(s.imgs) - 1
	}
	if i >= len(s.imgs) {
		i = 0
	}

	// Wait for the image to finish loading first.
	if s.imgs[i] == nil {
		s.win.nameSet(fmt.Sprintf("Loading..."))
		s.imgs[i] = <-s.imgChans[i]

		// If it's still nil, that means there was an error processing
		// this image. So delete it from s.imgs and s.imgChans and retry.
		if s.imgs[i] == nil {
			s.imgs = append(s.imgs[:i], s.imgs[i+1:]...)
			s.imgChans = append(s.imgChans[:i], s.imgChans[i+1:]...)
			s.imageSet(i)
			return
		}
	}
	s.imgi = i
	s.win.nameSet(fmt.Sprintf("%s (%dx%d)",
		s.image().name, s.image().Bounds().Dx(), s.image().Bounds().Dy()))
}

func (s *State) prevImage() {
	s.imageSet(s.imgi - 1)
	s.originSet(image.Point{0, 0})
}

func (s *State) nextImage() {
	s.imageSet(s.imgi + 1)
	s.originSet(image.Point{0, 0})
}

func (s *State) originSet(pt image.Point) {
	if s.image() == nil {
		return
	}
	dw := s.image().Bounds().Dx() - s.win.Geom.Width()
	dh := s.image().Bounds().Dy() - s.win.Geom.Height()

	pt.X = min(s.image().Bounds().Min.X+dw, max(pt.X, 0))
	pt.Y = min(s.image().Bounds().Min.Y+dh, max(pt.Y, 0))

	// Valid origin point. If the width/height of an image is smaller than
	// the canvas width/height, then the image origin cannot change in x/y
	// direction.
	if s.image().Bounds().Dx() < s.win.Geom.Width() {
		pt.X = 0
	}
	if s.image().Bounds().Dy() < s.win.Geom.Height() {
		pt.Y = 0
	}
	s.imgOrigin = pt
	s.win.drawImage()
}
