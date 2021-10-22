//go:build !plan9

package js

import (
	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
	"fmt"
	"github.com/psilva261/opossum/logger"
	"io"
	"os/user"
)

var fsys *client.Fsys

func dial() (err error) {
	log.Infof("Init...")
	conn, err := client.DialService("goja")
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	u, err := user.Current()
	if err != nil {
		return
	}
	un := u.Username
	fsys, err = conn.Attach(nil, un, "")
	if err != nil {
		log.Fatalf("attach: %v", err)
	}
	return
}

func callGojaCtl() (rwc io.ReadWriteCloser, err error) {
	if fsys == nil {
		if err := dial(); err != nil {
			return nil, fmt.Errorf("dial: %v", err)
		}
	}
	return fsys.Open("ctl", plan9.ORDWR)
}
