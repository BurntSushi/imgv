package main

import (
	"flag"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

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

	// Whether to run a CPU profile.
	flagProfile string
)

func init() {
	// Set GOMAXPROCS, since imgv can benefit greatly from parallelism.
	runtime.GOMAXPROCS(runtime.NumCPU())

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
	flag.StringVar(&flagProfile, "profile", "",
		"If set, a CPU profile will be saved to the file name provided.")
	flag.Parse()

	// Do some error checking on the flag values... naughty!
	if flagWidth == 0 || flagHeight == 0 {
		errLg.Fatal("The width and height must be non-zero values.")
	}
}

func main() {
	// Run the CPU profile if we're instructed to.
	if len(flagProfile) > 0 {
		f, err := os.Create(flagProfile)
		if err != nil {
			errLg.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// Connect to X and quit if we fail.
	X, err := xgbutil.NewConn()
	if err != nil {
		errLg.Fatal(err)
	}

	// Create the X window before starting anything so that the user knows
	// something is going on.
	window := newWindow(X)

	// Decode all images (in parallel).
	names, imgs := decodeImages(flag.Args())

	// Create the canvas and start the image goroutines.
	chans := canvas(X, window, names, len(imgs))
	for i, img := range imgs {
		go newImage(X, names[i], img, i, chans.imgLoadChans[i], chans.imgChan)
	}

	// Start the main X event loop.
	xevent.Main(X)
}

// decodeImages takes a list of image files and decodes them into image.Image
// types. Note that the number of images returned may not be the number of
// image files passed in. Namely, an image file is skipped if it cannot be
// read or deocoded into an image type that Go understands.
func decodeImages(imageFiles []string) ([]string, []image.Image) {
	// A temporary type used to transport decoded images over channels.
	type tmpImage struct {
		img  image.Image
		name string
	}

	// Decoded all images specified in parallel.
	imgChans := make([]chan tmpImage, flag.NArg())
	for i, fName := range imageFiles {
		imgChans[i] = make(chan tmpImage, 0)
		go func(i int, fName string) {
			file, err := os.Open(fName)
			if err != nil {
				errLg.Println(err)
				close(imgChans[i])
				return
			}

			img, kind, err := image.Decode(file)
			if err != nil {
				errLg.Printf("Could not decode '%s' into a supported image "+
					"format: %s", fName, err)
				close(imgChans[i])
				return
			}
			lg("Decoded '%s' into image type '%s'.",
				fName, kind)

			imgChans[i] <- tmpImage{
				img:  img,
				name: basename(fName),
			}
		}(i, fName)
	}

	// Now collect all the decoded images into a slice of names and a slice
	// of images.
	names := make([]string, 0, flag.NArg())
	imgs := make([]image.Image, 0, flag.NArg())
	for _, imgChan := range imgChans {
		if tmpImg, ok := <-imgChan; ok {
			names = append(names, tmpImg.name)
			imgs = append(imgs, tmpImg.img)
		}
	}

	return names, imgs
}
