//go:build !plan9

package js

import (
	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
	"fmt"
	"github.com/psilva261/mycel/logger"
	"io"
	"os/user"
)

var (
	conn *client.Conn
	fsys *client.Fsys
)

func (js *JS) dial() (err error) {
	log.Infof("Init...")
	conn, err := client.DialService(js.service)
	if err != nil {
		return fmt.Errorf("dial: %v", err)
	}
	u, err := user.Current()
	if err != nil {
		return
	}
	un := u.Username
	fsys, err = conn.Attach(nil, un, "")
	if err != nil {
		return fmt.Errorf("attach: %v", err)
	}
	return
}

func (js *JS) hangup() {
	if fsys != nil {
		fsys = nil
	}
	if conn != nil {
		conn.Close()
		conn = nil
	}
}

func (js *JS) callSparkleCtl() (rwc io.ReadWriteCloser, err error) {
	if fsys == nil {
		if err := js.dial(); err != nil {
			return nil, fmt.Errorf("dial: %v", err)
		}
	}
	return fsys.Open("ctl", plan9.ORDWR)
}
