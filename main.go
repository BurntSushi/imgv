package main

import (
	"flag"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"

	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/xevent"
)

var (
	// Global state. Contains X connection, images, window and current image.
	state *State

	// When flagVerbose is true, logging output will be written to stderr.
	// Errors will always be written to stderr.
	flagVerbose bool

	// The initial width and height of the window.
	flagWidth, flagHeight int
)

func init() {
	// Set the prefix for verbose output.
	log.SetPrefix("[imgv] ")

	// Set all of the flags.
	flag.BoolVar(&flagVerbose, "v", false,
		"When set, logging output will be printed to stderr.")
	flag.IntVar(&flagWidth, "width", 600,
		"The initial width of the window.")
	flag.IntVar(&flagHeight, "height", 600,
		"The initial height of the window.")
	flag.Parse()

	// Do some error checking on the flag values... naughty!
	if flagWidth == 0 || flagHeight == 0 {
		errLg.Fatal("The width and height must be non-zero values.")
	}
}

func main() {
	// Connect to X.
	X, err := xgbutil.NewConn()
	if err != nil {
		errLg.Fatal(err)
	}

	imgs := make([]Image, 0, flag.NArg())
	for _, fileName := range flag.Args() {
		file, err := os.Open(fileName)
		if err != nil {
			errLg.Println(err)
			continue
		}

		img, kind, err := image.Decode(file)
		if err != nil {
			errLg.Printf("Could not decode '%s' into a supported image "+
				"format: %s", err)
			continue
		}

		lg("Decoded '%s' into image type '%s'.", fileName, kind)
		imgs = append(imgs, newImage(X, fileName, img))
	}

	if len(imgs) == 0 {
		errLg.Println("No image files found.")
		os.Exit(1)
	}

	state = newState(X, imgs)

	state.imageSet(imgs[0], 100)
	xevent.Main(X)
}
