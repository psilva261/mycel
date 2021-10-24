package js

import (
	"io"
	"os"
)

func hangup() {}

func callGojaCtl() (rwc io.ReadWriteCloser, err error) {
	return os.OpenFile("/mnt/goja/ctl", os.O_RDWR, 0600)
}
