package fs

import (
	"fmt"
	"github.com/knusbaum/go9p"
	"os"
	"os/user"
	"syscall"
)

func group(u *user.User) (string, error) {
	return u.Gid, nil
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

	if err = syscall.Mount(int(f2.Fd()), -1, "/mnt/opossum", syscall.MCREATE, ""); err != nil {
		return fmt.Errorf("mount: %w", err)
	}
	return
}
