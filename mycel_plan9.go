package mycel

import (
	"os/user"
)

const PathPrefix = "/mnt/mycel"

func Group(u *user.User) (string, error) {
	return u.Gid, nil
}
