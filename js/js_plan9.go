package js

import (
	"io"
	"os"
)

func (js *JS) hangup() {}

func (js *JS) callSparkleCtl() (rwc io.ReadWriteCloser, err error) {
	return os.OpenFile("/mnt/sparkle/ctl", os.O_RDWR, 0600)
}
