//go:build !plan9

package mycel

import (
	"fmt"
	"os/user"
)

const PathPrefix = "mycel"

func Group(u *user.User) (string, error) {
	g, err := user.LookupGroupId(u.Gid)
	if err != nil {
		return "", fmt.Errorf("get group: %w", err)
	}
	return g.Name, nil
}
