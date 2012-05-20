package main

import (
	"github.com/BurntSushi/xgbutil/xgraphics"
)

type State struct {
	img Image
	ximg *xgraphics.Image
}

func getCurrentImage() *xgraphics.Image {
	return state.ximg
}

