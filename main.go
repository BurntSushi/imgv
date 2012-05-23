package main

import (
	"flag"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"strings"

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

type tmpImage struct {
	img  image.Image
	name string
}

func main() {
	// Connect to X.
	X, err := xgbutil.NewConn()
	if err != nil {
		errLg.Fatal(err)
	}

	imgChans := make([]chan tmpImage, flag.NArg())
	for i, fName := range flag.Args() {
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
			lg("Decoded '%s' into image type '%s'.", fName, kind)

			imgChans[i] <- tmpImage{
				img:  img,
				name: basename(fName),
			}
		}(i, fName)
	}

	imgs := make([]tmpImage, 0, flag.NArg())
	names := make([]string, 0, flag.NArg())
	for _, imgChan := range imgChans {
		if tmpImg, ok := <-imgChan; ok {
			imgs = append(imgs, tmpImg)
			names = append(names, tmpImg.name)
		}
	}

	chans := canvas(X, names, len(imgs))
	for i, tmpImage := range imgs {
		go newImage(X, tmpImage.name, tmpImage.img, i,
			chans.imgLoadChans[i], chans.imgChan)
	}

	xevent.Main(X)
}

func basename(fName string) string {
	if lslash := strings.LastIndex(fName, "/"); lslash != -1 {
		fName = fName[lslash+1:]
	}
	return fName
}
