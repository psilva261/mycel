package js

import (
	"io"
	"os"
)

func hangup() {}

func callSparkleCtl() (rwc io.ReadWriteCloser, err error) {
	return os.OpenFile("/mnt/sparkle/ctl", os.O_RDWR, 0600)
}
