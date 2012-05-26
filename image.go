package main

import (
	"image"
	"time"

	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/xgraphics"
)

// vimage acts as an xgraphics.Image type with a name.
// (The name is the basename of the image's corresponding file name.)
type vimage struct {
	*xgraphics.Image
	name string
}

// newImage is meant to be run as a goroutine and loads a decoded image into
// an xgraphics.Image value and draws it to an X pixmap.
// The loading doesn't start until this image's corresponding imgLoadChan
// has been pinged.
// This implies that all images are decoded on start-up and are converted
// and drawn to an X pixmap on-demand. I am still deliberating on whether this
// is a smart decision.
// Note that this process, particularly image conversion, can be quite
// costly for large images.
func newImage(X *xgbutil.XUtil, name string, img image.Image, index int,
	imgLoadChan chan struct{}, imgChan chan imageLoaded) {

	// Don't start loading until we're told to do so.
	<-imgLoadChan

	// We send this when we're done processing this image, whether its
	// an error or not.
	loaded := imageLoaded{index: index}

	start := time.Now()
	reg := xgraphics.NewConvert(X, img)
	lg("Converted '%s' to an xgraphics.Image type (%s).",
		name, time.Since(start))

	// Only blend a checkered background if the image *may* have an alpha 
	// channel. If we want to be a bit more efficient, we could type switch
	// on all image types use Opaque, but this may add undesirable overhead.
	// (Where the overhead is scanning the image for opaqueness.)
	switch img.(type) {
	case *image.Gray:
	case *image.Gray16:
	case *image.YCbCr:
	default:
		start = time.Now()
		blendCheckered(reg)
		lg("Blended '%s' into a checkered background (%s).",
			name, time.Since(start))
	}

	if err := reg.CreatePixmap(); err != nil {
		// TODO: We should display a "Could not load image" image instead
		// of dying. However, creating a pixmap rarely fails, unless we have
		// a *ton* of images. (In all likelihood, we'll run out of memory
		// before a new pixmap cannot be created.)
		errLg.Fatal(err)
	} else {
		start = time.Now()
		reg.XDraw()
		lg("Drawn '%s' to an X pixmap (%s).", name, time.Since(start))
	}

	loaded.img = &vimage{
		Image: reg,
		name:  name,
	}

	// Tell the canvas that this image has been loaded.
	imgChan <- loaded
}

// blendCheckered is basically a copy of xgraphics.Blend with no interfaces.
// (It's faster.) Also, it is hardcoded to blend into a checkered background.
func blendCheckered(dest *xgraphics.Image) {
	dsrc := dest.Bounds()
	dmnx, dmxx, dmny, dmxy := dsrc.Min.X, dsrc.Max.X, dsrc.Min.Y, dsrc.Max.Y

	clr1 := xgraphics.BGRA{B: 0xff, G: 0xff, R: 0xff, A: 0xff}
	clr2 := xgraphics.BGRA{B: 0xde, G: 0xdc, R: 0xdf, A: 0xff}

	var dx, dy int
	var bgra, clr xgraphics.BGRA
	for dx = dmnx; dx < dmxx; dx++ {
		for dy = dmny; dy < dmxy; dy++ {
			if dx%30 >= 15 {
				if dy%30 >= 15 {
					clr = clr1
				} else {
					clr = clr2
				}
			} else {
				if dy%30 >= 15 {
					clr = clr2
				} else {
					clr = clr1
				}
			}

			bgra = dest.At(dx, dy).(xgraphics.BGRA)
			dest.SetBGRA(dx, dy, xgraphics.BlendBGRA(clr, bgra))
		}
	}
}
