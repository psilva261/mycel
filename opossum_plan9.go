package opossum

import (
	"os/user"
)

func Group(u *user.User) (string, error) {
	return u.Gid, nil
}
