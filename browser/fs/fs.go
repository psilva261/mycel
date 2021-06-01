package fs

import (
	"fmt"
	"github.com/knusbaum/go9p/fs"
	"github.com/knusbaum/go9p/proto"
	"github.com/psilva261/opossum/logger"
	"os/user"
	"sync"
)

var (
	log *logger.Logger
	mu sync.Mutex
	oFS *fs.FS
	un string
	gn string
	jsDir *fs.StaticDir
	html string
)

func SetLogger(l *logger.Logger) {
	log = l
}

func Update(htm string, js []string) {
	mu.Lock()
	defer mu.Unlock()

	html = htm

	for name := range jsDir.Children() {
		jsDir.DeleteChild(name)
	}
	for i, s := range js {
		fn := fmt.Sprintf("%d.js", i)
		f := fs.NewStaticFile(
			oFS.NewStat(fn, un, gn, 0400),
			[]byte(s),
		)
		jsDir.AddChild(f)
	}
}

func Srv9p() {
	if err := srv9p(); err != nil {
		log.Errorf("srv9p: %v", err)
	}
}

func srv9p() (err error) {
	var root *fs.StaticDir

	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	un = u.Username
	gn, err = group(u)
	if err != nil {
		return fmt.Errorf("get group: %w", err)
	}
	oFS, root = fs.NewFS(un, gn, 0500)
	h := fs.NewDynamicFile(
		oFS.NewStat("html", un, gn, 0400),
		func() []byte {
			mu.Lock()
			defer mu.Unlock()

			return []byte(html)
		},
	)
	root.AddChild(h)
	d, err := fs.CreateStaticDir(oFS, root, un, "js", 0500|proto.DMDIR, 0)
	if err != nil {
		return
	}
	jsDir = d.(*fs.StaticDir)
	root.AddChild(jsDir)

	return post(oFS.Server())
}
