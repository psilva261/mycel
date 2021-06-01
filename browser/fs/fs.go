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
	mu sync.RWMutex
	oFS *fs.FS
	un string
	gn string
	jsDir *fs.StaticDir
	html string
)

func SetLogger(l *logger.Logger) {
	log = l
}

func init() {
	var root *fs.StaticDir

	u, err := user.Current()
	if err != nil {
		log.Errorf("get user: %w", err)
		return
	}
	un = u.Username
	gn, err = group(u)
	if err != nil {
		log.Errorf("get group: %w", err)
		return
	}
	oFS, root = fs.NewFS(un, gn, 0500)
	h := fs.NewDynamicFile(
		oFS.NewStat("html", un, gn, 0400),
		func() []byte {
			mu.RLock()
			defer mu.RUnlock()

			return []byte(html)
		},
	)
	root.AddChild(h)
	d, err := fs.CreateStaticDir(oFS, root, un, "js", 0500|proto.DMDIR, 0)
	if err != nil {
		log.Errorf("create static dir: %w", err)
		return
	}
	jsDir = d.(*fs.StaticDir)
	root.AddChild(jsDir)
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
	if err := post(oFS.Server()); err != nil {
		log.Errorf("srv9p: %v", err)
	}
}
