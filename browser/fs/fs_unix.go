//go:build !plan9
// +build !plan9

package fs

import (
	"github.com/knusbaum/go9p"
)

func post(srv go9p.Srv) (err error) {
	return go9p.PostSrv("opossum", srv)
}
