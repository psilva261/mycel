package logger

import (
	"fmt"
	"io"
	goLog "log"
	"os"
	"sync"
)

// Sink for Go's log pkg
var Sink io.Writer
var Quiet bool
var Log *Logger
var gl *goLog.Logger

func Init() {
	gl = goLog.New(os.Stderr, "", goLog.LstdFlags)
	Log = &Logger{}
	if Quiet {
		Sink = &NullWriter{}
		goLog.SetOutput(Sink)
	}
}

type NullWriter struct{}

func (w *NullWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	return
}

type Logger struct {
	Debug    bool

	mu       sync.Mutex
	last     string
	lastSev  int
	repeated int
}

func (l *Logger) Printf(format string, v ...interface{}) {
	l.emit(debug, format, v...)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.emit(info, format, v...)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.emit(er, format, v...)
}

func (l *Logger) Fatal(v ...interface{}) {
	gl.Fatal(v...)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
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

func (l *Logger) Flush() {
	l.emit(flush, "")
}

func (l *Logger) emit(severity int, format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if severity == debug && !l.Debug {
		return
	}
	if (severity != fatal && severity != flush) && Quiet {
		return
	}

	msg := fmt.Sprintf(format, v...)
	switch {
	case l.last == msg && l.lastSev == severity:
		l.repeated++
	case l.repeated > 0:
		goLog.Printf("...and %v more", l.repeated)
		l.repeated = 0
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
	l.last = msg
	l.lastSev = severity
}
