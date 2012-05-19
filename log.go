package main

import (
	"log"
	"os"
)

var errLg = log.New(os.Stderr, "[imgv error] ", log.Lshortfile)

func lg(format string, v ...interface{}) {
	if !flagVerbose {
		return
	}
	log.Printf(format, v...)
}
