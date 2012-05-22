package main

import (
	"flag"
	"log"

	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/xevent"
)

var (
	// When flagVerbose is true, logging output will be written to stderr.
	// Errors will always be written to stderr.
	flagVerbose bool

	// The initial width and height of the window.
	flagWidth, flagHeight int

	// The amount to increment panning when using h,j,k,l
	flagStepIncrement int
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
	flag.IntVar(&flagStepIncrement, "increment", 20,
		"The increment used to pan the image when using keyboard shortcuts.")
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

	// Get another connection to send images over.
	Ximg, err := xgbutil.NewConn()
	if err != nil {
		errLg.Fatal(err)
	}

	chans := canvas(X, flag.NArg())

	for i, fName := range flag.Args() {
		go newImage(Ximg, fName, i, chans.imgChan)
	}

	xevent.Main(X)
}
