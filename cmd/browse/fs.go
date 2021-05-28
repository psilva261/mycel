package main

import (
	"fmt"
	"github.com/knusbaum/go9p/fs"
	"github.com/psilva261/opossum/browser"
	"os/user"
	"os"
	"sync"
)

func Srv9p(b *browser.Browser) {
	if err := srv9p(b); err != nil {
		log.Errorf("srv9p: %v", err)
	}
}

type Root struct{
	html os.FileInfo
	mu sync.Mutex
	of int64
}

func srv9p(b *browser.Browser) (err error) {
	log.Infof("srv9p")
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	un := u.Username
	gn, err := group(u)
	if err != nil {
		return fmt.Errorf("get group: %w", err)
	}
	oFS, root := fs.NewFS(un, gn, 0500)
	h := fs.NewDynamicFile(
		oFS.NewStat("html", un, gn, 0400),
		func() []byte {
			return []byte(b.Website.Html())
		},
	)
	root.AddChild(h)

	log.Infof("post fs")
	return post(oFS.Server())
}
