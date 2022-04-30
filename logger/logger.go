package log

import (
	"log"
	"sync"
)

var (
	mu    sync.Mutex
	quiet bool
	Debug bool
)

func SetQuiet() {
	mu.Lock()
	defer mu.Unlock()

	quiet = true
}

func Printf(format string, v ...interface{}) {
	if Debug && !quiet {
		log.Printf(format, v...)
	}
}

func Infof(format string, v ...interface{}) {
	if !quiet {
		log.Printf(format, v...)
	}
}

func Errorf(format string, v ...interface{}) {
	if !quiet {
		log.Printf(format, v...)
	}
}

func Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}
