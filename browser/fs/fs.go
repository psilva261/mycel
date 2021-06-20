package fs

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/knusbaum/go9p/fs"
	"github.com/knusbaum/go9p/proto"
	"github.com/psilva261/opossum"
	"github.com/psilva261/opossum/logger"
	"github.com/psilva261/opossum/nodes"
	"net"
	"os/user"
	"strings"
	"sync"
)

var (
	log *logger.Logger
	mu sync.RWMutex
	oFS *fs.FS
	un string
	gn string
	cssDir *fs.StaticDir
	jsDir *fs.StaticDir
	html string
	DOM Queryable
)

type Queryable interface {
	Query(q string) ([]*nodes.Node, error)
}

func SetLogger(l *logger.Logger) {
	log = l
}

func init() {
	var root *fs.StaticDir

	u, err := user.Current()
	if err != nil {
		log.Errorf("get user: %v", err)
		return
	}
	un = u.Username
	gn, err = opossum.Group(u)
	if err != nil {
		log.Errorf("get group: %v", err)
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
	d, err := fs.CreateStaticDir(oFS, root, un, "css", 0500|proto.DMDIR, 0)
	if err != nil {
		log.Errorf("create static dir: %v", err)
		return
	}
	cssDir = d.(*fs.StaticDir)
	root.AddChild(cssDir)
	d, err = fs.CreateStaticDir(oFS, root, un, "js", 0500|proto.DMDIR, 0)
	if err != nil {
		log.Errorf("create static dir: %v", err)
		return
	}
	jsDir = d.(*fs.StaticDir)
	root.AddChild(jsDir)
	q := fs.NewListenFile(oFS.NewStat("query", un, gn, 0600))
	root.AddChild(q)
	lq := (*fs.ListenFileListener)(q)
	go Query(lq)
}

func Query(lq *fs.ListenFileListener) {
	for {
		conn, err := lq.Accept()
		if err != nil {
			log.Errorf("accept: %v", err)
			continue
		}
		go query(conn)
	}
}

func query(conn net.Conn) {
	r := bufio.NewReader(conn)
	enc := json.NewEncoder(conn)
	defer conn.Close()

	l, err := r.ReadString('\n')
	if err != nil {
		log.Errorf("read string: %v", err)
		return
	}
	l = strings.TrimSpace(l)

	if DOM == nil {
		return
	}
	nodes, err := DOM.Query(l)
	if err != nil {
		log.Errorf("query nodes: %v", err)
		return
	}
	if err := enc.Encode(nodes); err != nil {
		log.Errorf("encode: %v", err)
		return
	}
}

func Update(htm string, css []string, js []string) {
	mu.Lock()
	defer mu.Unlock()

	html = htm
	for name := range cssDir.Children() {
		cssDir.DeleteChild(name)
	}
	for i, s := range css {
		fn := fmt.Sprintf("%d.css", i)
		f := fs.NewStaticFile(
			oFS.NewStat(fn, un, gn, 0400),
			[]byte(s),
		)
		cssDir.AddChild(f)
	}
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
