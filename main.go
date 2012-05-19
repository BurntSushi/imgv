package main

import (
	"flag"
	"log"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/xevent"
)

var (
	// The X connection.
	X *xgbutil.XUtil

	// Global slice of all images parsed at startup.
	imgs []Image

	// When flagVerbose is true, logging output will be written to stderr.
	// Errors will always be written to stderr.
	flagVerbose bool
)

func init() {
	var err error

	// Connect to X.
	X, err = xgbutil.NewConn()
	if err != nil {
		errLg.Fatal(err)
	}

	// Set the prefix for verbose output.
	log.SetPrefix("[imgv] ")

	// Set all of the flags.
	flag.BoolVar(&flagVerbose, "v", false,
		"When set, logging output will be printed to stderr.")
	flag.Parse()
}

func main() {
	for _, fileName := range flag.Args() {
		file, err := os.Open(fileName)
		if err != nil {
			errLg.Println(err)
			continue
		}

		img, kind, err := image.Decode(file)
		if err != nil {
			errLg.Printf("Could not decode '%s' into a supported image " +
				"format: %s", err)
			continue
		}

		lg("Decoded '%s' into image type '%s'.", fileName, kind)
		imgs = append(imgs, newImage(img))
	}

	if len(imgs) == 0 {
		errLg.Println("No image files found.")
		os.Exit(1)
	}

	imgs[0].sizes[100].XShow()

	xevent.Main(X)
}

