package opossum

import (
	"os/user"
)

const PathPrefix = "/mnt/opossum"

func Group(u *user.User) (string, error) {
	return u.Gid, nil
}
