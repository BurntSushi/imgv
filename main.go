package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/xevent"
)

var (
	// When flagVerbose is true, logging output will be written to stderr.
	// Errors will always be written to stderr.
	flagVerbose bool

	// The initial width and height of the window.
	flagWidth, flagHeight int

	// If set, the image window will automatically resize to the first image
	// that it displays.
	flagAutoResize bool

	// The amount to increment panning when using h,j,k,l
	flagStepIncrement int

	// Whether to run a CPU profile.
	flagProfile string

	// When set, imgv will print all keybindings and exit.
	flagKeybindings bool

	// A list of keybindings. Each value corresponds to a triple of the key
	// sequence to bind to, the action to run when that key sequence is
	// pressed and a quick description of what the keybinding does.
	keybinds = []keyb{
		{
			"left", "Cycle to the previous image.",
			func(w *window) { w.chans.prevImg <- struct{}{} },
		},
		{
			"right", "Cycle to the next image.",
			func(w *window) { w.chans.nextImg <- struct{}{} },
		},
		{
			"shift-h", "Cycle to the previous image.",
			func(w *window) { w.chans.prevImg <- struct{}{} },
		},
		{
			"shift-l", "Cycle to the next image.",
			func(w *window) { w.chans.nextImg <- struct{}{} },
		},
		{
			"r", "Resize the window to fit the current image.",
			func(w *window) { w.chans.resizeToImageChan <- struct{}{} },
		},
		{
			"h", "Pan left.", func(w *window) { w.stepLeft() },
		},
		{
			"j", "Pan down.", func(w *window) { w.stepDown() },
		},
		{
			"k", "Pan up.", func(w *window) { w.stepUp() },
		},
		{
			"l", "Pan right.", func(w *window) { w.stepRight() },
		},
		{
			"q", "Quit.", func(w *window) { xevent.Quit(w.X) },
		},
	}
)

func init() {
	// Set GOMAXPROCS, since imgv can benefit greatly from parallelism.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Set the prefix for verbose output.
	log.SetPrefix("[imgv] ")

	// Set all of the flags.
	flag.BoolVar(&flagVerbose, "v", false,
		"If set, logging output will be printed to stderr.")
	flag.IntVar(&flagWidth, "width", 600,
		"The initial width of the window.")
	flag.IntVar(&flagHeight, "height", 600,
		"The initial height of the window.")
	flag.BoolVar(&flagAutoResize, "auto-resize", false,
		"If set, window will resize to size of first image.")
	flag.IntVar(&flagStepIncrement, "increment", 20,
		"The increment (in pixels) used to pan the image.")
	flag.StringVar(&flagProfile, "profile", "",
		"If set, a CPU profile will be saved to the file name provided.")
	flag.BoolVar(&flagKeybindings, "keybindings", false,
		"If set, imgv will output a list all keybindings.")
	flag.Usage = usage
	flag.Parse()

	// Do some error checking on the flag values... naughty!
	if flagWidth == 0 || flagHeight == 0 {
		errLg.Fatal("The width and height must be non-zero values.")
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [flags] image-file [image-file ...]\n",
		basename(os.Args[0]))
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	// If we just need the keybindings, print them and be done.
	if flagKeybindings {
		for _, keyb := range keybinds {
			fmt.Printf("%-10s %s\n", keyb.key, keyb.desc)
		}
		fmt.Printf("%-10s %s\n", "mouse",
			"Left mouse button will pan the image.")
		os.Exit(0)
	}

	// Run the CPU profile if we're instructed to.
	if len(flagProfile) > 0 {
		f, err := os.Create(flagProfile)
		if err != nil {
			errLg.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// Whoops!
	if flag.NArg() == 0 {
		fmt.Fprint(os.Stderr, "\n")
		errLg.Print("No images specified.\n\n")
		usage()
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
	names, imgs := decodeImages(findFiles(flag.Args()))

	// Die now if we don't have any images!
	if len(imgs) == 0 {
		errLg.Fatal("No images specified could be shown. Quitting...")
	}

	// Auto-size the window if appropriate.
	if flagAutoResize {
		window.Resize(imgs[0].Bounds().Dx(), imgs[0].Bounds().Dy())
	}

	// Create the canvas and start the image goroutines.
	chans := canvas(X, window, names, len(imgs))
	for i, img := range imgs {
		go newImage(X, names[i], img, i, chans.imgLoadChans[i], chans.imgChan)
	}

	// Start the main X event loop.
	xevent.Main(X)
}

func findFiles(args []string) []string {
	files := []string{}
	for _, f := range args {
		fi, err := os.Stat(f)
		if err != nil {
			errLg.Print("Can't access", f, err)
		} else if fi.IsDir() {
			files = append(files, dirImages(f)...)
		} else {
			files = append(files, f)
		}
	}
	return files 
}

func dirImages(dir string) []string {

	fd, _ := os.Open(dir)	
	fs, _ := fd.Readdirnames(0)
	files := []string{}
	for _, f := range fs {
		// TODO filter by regexp
		if filepath.Ext(f) != "" {
			files = append(files, filepath.Join(dir, f))
		}
	}
	return files
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
	imgChans := make([]chan tmpImage, len(imageFiles))
	for i, fName := range imageFiles {
		imgChans[i] = make(chan tmpImage, 0)
		go func(i int, fName string) {
			file, err := os.Open(fName)
			if err != nil {
				errLg.Println(err)
				close(imgChans[i])
				return
			}

			start := time.Now()
			img, kind, err := image.Decode(file)
			if err != nil {
				errLg.Printf("Could not decode '%s' into a supported image "+
					"format: %s", fName, err)
				close(imgChans[i])
				return
			}
			lg("Decoded '%s' into image type '%s' (%s).",
				fName, kind, time.Since(start))

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
