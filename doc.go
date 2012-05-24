/*
imgv is a simple image viewer that only works with X and is written in Go. It 
supports image formats that can be decoded by the Go standard library 
(currently jpeg, gif and png). It supports panning the image but does not (yet) 
support zooming.

Usage:
	imgv [flags] image-file [image-file ...]

The flags are:
	--height pixels, --width pixels
		The 'height' and 'width' flags allow one to specify the initial size
		of the image window. The image window can still change size afterwards.
	--auto-resize
		If set, the image window will be automatically resized to the first 
		image displayed. This overrides the 'height' and 'width' options.
	--increment pixels
		The amount of pixels to pan an image at each step when using the 
		keyboard shortcuts.
	--keybindings
		If set, a list of all key bindings (and mouse bindings) set by imgv is
		printed. A small description of what each key binding does is included.
	-v
		If set, more output will be printed to stderr. Useful for debugging.
	--profile prof-file-name
		If set, a CPU profile will be saved to prof-file-name. This is for
		development purposes only.

Details

imgv is about as simple as it gets for an image viewer. It only supports
displaying the image and panning around the image when parts of it are not
viewable. It does not support zooming or any kind of image manipulation.

My two primary future goals are to support zooming and to increase 
performance.  (I'll rely on the Go standard library to write new image format 
decoders).

I didn't include zooming in the initial release because it adds a surprising 
amount of complexity and has broad-sweeping performance implications depending 
upon its implementation.

High-level overview

imgv starts up by attempting to decode all images specified on the command 
line. After all images are decoded, the first image is converted to an 
xgbutil/xgraphics.Image type and drawn on to an X pixmap. At this point, the 
first image is then painted to the window.

When the next image is requested to be displayed, it is then converted to an
xgbutil/xgraphics.Image type and drawn to an X pixmap on demand. Then it is 
painted to the window.

Performance

It has a somewhat concurrent design, and will benefit from parallelism 
(particularly at startup). Also, the underlying library used (XGB) benefits 
from parallelism.

The high-level overview given above may sound a bit weird (i.e., why decode all 
images before showing the first?). My reason is that this was the quickest and 
simplest way to get something working, since decoding an image has a reasonable 
chance of failure. (There is additional complexity involved in handling failure 
at the concurrent level.) Decoding will take advantage of parallelism and is 
typically fairly quick (unless a lot of images are specified).

Perhaps the biggest performance implication is what is done on-demand when a 
new image must be loaded. If it has already been converted and painted to an X 
pixmap, this process is nearly instant. If its the first loading, then it must 
be converted to an xgbutil/xgraphics.Image type and drawn to an X pixmap before 
it can be painted to a window.

Conversion to the xgbutil/xgraphics.Image type is, by far, the bottleneck. The 
process includes transforming every pixel in the decoded image to the correct 
image byte order (currently BGRA), which is the format expected by X (in common 
configurations). While this is fairly quick for small images, it can be quite 
slow for larger images.

The ideal solution, assuming image conversion itself cannot be sped up, seems 
to be to process image conversions in the background with the hope that they 
will be ready (or close to ready) when they are requested. The big problem with 
this approach is when a lot of images are specified. What if the image 
requested by the user won't even start loading for a long time because other 
image conversions are hogging the CPU? I'm not sure how to solve that, other 
than perhaps an ugly hack using runtime.Gosched.

Another direction that could be taken is to only convert the pieces of the 
image that are being displayed. This relies on the fact that most setups cannot 
view the entirety of a large image (> 2,500,000 pixels) at one time. This comes 
at the cost of increased complexity but is probably the most performant 
solution. (The complexity lay in splitting conversion up into pieces, and 
triggering the appropriate conversion when the image is panned.)

As for drawing the image to an X pixmap, I was surprised to see that this was 
fairly quick by comparison. It uses Go's built in copy function, which I 
suspect is the source of its speediness.

Zooming

Zooming into an image can be more precisely described as increasing the scale 
of an image (zoom in) and decreasing the scale of an image (zoom out).

Zooming adds some complexity to the design of imgv, as it requires representing 
each image as a set of images, and keeping state to determine which image in 
the set is currently viewable. (Where each image in the set corresponds to a 
different scaling level.)

While complexity is a reasonable barrier, the bigger barrier is the performance 
implications. Namely, zooming in exacerbates the performance problems described 
in the section above. It turns a sort-of extreme case (large images) into a 
common occurrence.

It would appear that the only viable means of implementing scaling is to only 
scale the part of the image that can be viewed. (Scaling seems to be bounded by 
the scale rather than the size of original image. On my machine, an Intel Core 
2 Duo, a 1000x1000 scale takes on the order of a second or two. Which is slow.) 
This is probably the only choice not just in the interest in keeping things 
speedy, but also for memory usage. (Keeping several different scaled and 
complete versions of a large image can use a ton of memory. With only a few 
images like this, memory usage adds up quickly.)

Perhaps another option is write a scaling routine that optimizes the use of 
interfaces out of the performance critical sections. Doing this for image 
conversion achieved 50-80% speed ups. (I don't think graphics-go does this 
currently.)

Portability

Obviously, the image viewer will only work with an X server. There are no plans 
to change this.

More interesting is portability among X servers. While imgv is itself portable 
across any X server, the underlying library (xgbutil/xgraphics) is not quite 
there yet. Namely, xgbutil/xgraphics assumes a BGRx format (24 bit depth with 
32 bytes per pixel and a least significant image byte order). This is wrong and 
needs to be more flexible to fit any X server configuration.

If your X server doesn't have the configuration expected by xgbutil/xgraphics, 
you should see some messages emitted to stderr. If you do see this, I'd greatly 
appreciate a bug report filed at the xgbutil project page with the messages
that you see:
https://github.com/BurntSushi/xgbutil.

Author

I have never developed an image viewer before, and I'm pretty sure I've never 
looked at the source code of another image viewer. Therefore, it's quite likely 
that I'm stumbling over solved problems. (Are they solved in image viewers or 
GUI toolkits?)

*/
package main
