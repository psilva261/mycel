//go:build !plan9

package opossum

import (
	"fmt"
	"os/user"
)

const PathPrefix = "opossum"

func Group(u *user.User) (string, error) {
	g, err := user.LookupGroupId(u.Gid)
	if err != nil {
		return "", fmt.Errorf("get group: %w", err)
	}
	return g.Name, nil
}
