package main

import (
	"fmt"
	"github.com/knusbaum/go9p"
	"github.com/psilva261/opossum/logger"
	"io"
	"os"
	"syscall"
)

func Init() (err error) {
	mtpt = "/mnt/opossum"
	if htm != "" || len(js) > 0 {
		log.Printf("not loading htm/js from mtpt")
		return
	}
	bs, err := os.ReadFile(mtpt + "/html")
	if err != nil {
		return
	}
	htm = string(bs)
	ds, err := os.ReadDir(mtpt + "/js")
	if err != nil {
		return
	}
	for i := 0; i < len(ds); i++ {
		fn := fmt.Sprintf(mtpt+"/js/%v.js", i)
		log.Infof("fn=%v", fn)
		bs, err := os.ReadFile(fn)
		if err != nil {
			return fmt.Errorf("read all: %w", err)
		}
		js = append(js, string(bs))
	}
	return
}

func openQuery() (rwc io.ReadWriteCloser, err error) {
	return os.OpenFile(mtpt+"/query", os.O_RDWR, 0600)
}

func openXhr() (rwc io.ReadWriteCloser, err error) {
	return os.OpenFile(mtpt+"/xhr", os.O_RDWR, 0600)
}

func post(srv go9p.Srv) (err error) {
	f1, f2, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("pipe: %w", err)
	}

	go func() {
		err = go9p.ServeReadWriter(f1, f1, srv)
		if err != nil {
			log.Errorf("serve rw: %v", err)
		}
	}()

	if err = syscall.Mount(int(f2.Fd()), -1, "/mnt/goja", syscall.MCREATE, ""); err != nil {
		return fmt.Errorf("mount: %w", err)
	}
	return
}

func callGojaCtl() (rwc io.ReadWriteCloser, err error) {
	return os.OpenFile("/mnt/goja/ctl", os.O_RDWR, 0600)
}
