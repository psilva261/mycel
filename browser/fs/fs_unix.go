// +build !plan9

package fs

import (
	"fmt"
	"github.com/knusbaum/go9p"
	"os/user"
)

func group(u *user.User) (string, error) {
	g, err := user.LookupGroupId(u.Gid)
	if err != nil {
		return "", fmt.Errorf("get group: %w", err)
	}
	return g.Name, nil
}

func post(srv go9p.Srv) (err error) {
	return go9p.PostSrv("opossum", srv)
}

