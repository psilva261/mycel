package log

import (
	"fmt"
	"io"
	goLog "log"
	"os"
	"sync"
)

// Sink for Go's log pkg
var (
	Sink io.Writer
	quiet bool
	gl *goLog.Logger

	Debug    bool

	mu       sync.Mutex
	last     string
	lastSev  int
	repeated int
)

func init() {
	gl = goLog.New(os.Stderr, "", goLog.LstdFlags)
}

func SetQuiet() {
	mu.Lock()
	defer mu.Unlock()

	quiet = true
	Sink = &NullWriter{}
	goLog.SetOutput(Sink)
}

type NullWriter struct{}

func (w *NullWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	return
}

func Printf(format string, v ...interface{}) {
	emit(debug, format, v...)
}

func Infof(format string, v ...interface{}) {
	emit(info, format, v...)
}

func Errorf(format string, v ...interface{}) {
	emit(er, format, v...)
}

func Fatal(v ...interface{}) {
	gl.Fatal(v...)
}

func Fatalf(format string, v ...interface{}) {
	gl.Fatalf(format, v...)
}

const (
	debug = iota
	print
	info
	er
	fatal

	flush
)

func Flush() {
	emit(flush, "")
}

func emit(severity int, format string, v ...interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if severity == debug && !Debug {
		return
	}
	if (severity != fatal && severity != flush) && quiet {
		return
	}

	msg := fmt.Sprintf(format, v...)
	switch {
	case last == msg && lastSev == severity:
		repeated++
	case repeated > 0:
		goLog.Printf("...and %v more", repeated)
		repeated = 0
		fallthrough
	default:
		switch severity {
		case debug:
			gl.Printf(format, v...)
		case info:
			gl.Printf(format, v...)
		case er:
			gl.Printf(format, v...)
		case fatal:
			gl.Fatalf(format, v...)
		case flush:
		}
	}
	last = msg
	lastSev = severity
}
