package fs

import (
	"bufio"
	"encoding/json"
	"fmt"
	go9pfs "github.com/knusbaum/go9p/fs"
	"github.com/knusbaum/go9p/proto"
	"github.com/psilva261/mycel"
	"github.com/psilva261/mycel/logger"
	"github.com/psilva261/mycel/nodes"
	"net"
	"net/http"
	"os/user"
	"strings"
	"sync"
)

type FS struct {
	mu *sync.RWMutex
	c  *sync.Cond

	root     *go9pfs.FS
	oFS     *go9pfs.FS
	un      string
	gn      string
	url     string
	htm     string
	cssDir  *go9pfs.StaticDir
	jsDir   *go9pfs.StaticDir
	rt      *Node
	Client  *http.Client
	Fetcher mycel.Fetcher
}

func New() *FS {
	fs := &FS{}
	fs.mu = &sync.RWMutex{}
	fs.c = sync.NewCond(fs.mu)
	fs.SetDOM(nil)

	return fs
}

func (fs *FS) SetDOM(d *nodes.Node) {
	if fs.rt == nil {
		fs.rt = &Node{
			fs:   fs,
			name: "0",
		}
	}
	fs.rt.nt = d
}

func userGroup() (un, gn string, err error) {
	u, err := user.Current()
	if err != nil {
		return "", "", fmt.Errorf("current user: %w", err)
	}
	un = u.Username
	gn, err = mycel.Group(u)
	if err != nil {
		return "", "", fmt.Errorf("group: %v", err)
	}
	return
}

func (fs *FS) Srv9p() {
	fs.c.L.Lock()
	var root *go9pfs.StaticDir
	var err error

	fs.un, fs.gn, err = userGroup()
	if err != nil {
		log.Errorf("get user: %v", err)
		fs.c.L.Unlock()
		return
	}
	fs.oFS, root = go9pfs.NewFS(fs.un, fs.gn, 0500)
	u := go9pfs.NewDynamicFile(
		fs.oFS.NewStat("url", fs.un, fs.gn, 0400),
		func() []byte {
			fs.mu.RLock()
			defer fs.mu.RUnlock()

			return []byte(fs.url)
		},
	)
	root.AddChild(u)
	h := go9pfs.NewDynamicFile(
		fs.oFS.NewStat("html", fs.un, fs.gn, 0400),
		func() []byte {
			fs.mu.RLock()
			defer fs.mu.RUnlock()

			return []byte(fs.htm)
		},
	)
	root.AddChild(h)
	d, err := go9pfs.CreateStaticDir(fs.oFS, root, fs.un, "css", 0500|proto.DMDIR, 0)
	if err != nil {
		log.Errorf("create static dir: %v", err)
		fs.c.L.Unlock()
		return
	}
	fs.cssDir = d.(*go9pfs.StaticDir)
	root.AddChild(fs.cssDir)
	d, err = go9pfs.CreateStaticDir(fs.oFS, root, fs.un, "js", 0500|proto.DMDIR, 0)
	if err != nil {
		log.Errorf("create static dir: %v", err)
		fs.c.L.Unlock()
		return
	}
	fs.jsDir = d.(*go9pfs.StaticDir)
	root.AddChild(fs.jsDir)
	q := go9pfs.NewListenFile(fs.oFS.NewStat("query", fs.un, fs.gn, 0600))
	root.AddChild(q)
	lq := (*go9pfs.ListenFileListener)(q)
	root.AddChild(fs.rt)
	go fs.Query(lq)
	if fs.Client != nil {
		xhr := go9pfs.NewListenFile(fs.oFS.NewStat("xhr", fs.un, fs.gn, 0600))
		root.AddChild(xhr)
		lxhr := (*go9pfs.ListenFileListener)(xhr)
		go fs.Xhr(lxhr)
	}
	fs.c.Broadcast()
	fs.c.L.Unlock()

	if err := post(fs.oFS.Server()); err != nil {
		log.Errorf("srv9p: %v", err)
	}
}

func (fs *FS) Query(lq *go9pfs.ListenFileListener) {
	for {
		conn, err := lq.Accept()
		if err != nil {
			log.Errorf("query: accept: %v", err)
			continue
		}
		go fs.query(conn)
	}
}

func (fs *FS) query(conn net.Conn) {
	r := bufio.NewReader(conn)
	enc := json.NewEncoder(conn)
	defer conn.Close()

	l, err := r.ReadString('\n')
	if err != nil {
		log.Errorf("read string: %v", err)
		return
	}
	l = strings.TrimSpace(l)

	if fs.rt.nt == nil {
		log.Infof("DOM is nil")
		return
	}
	nodes, err := fs.rt.nt.Query(l)
	if err != nil {
		log.Errorf("query nodes: %v", err)
		return
	}
	if err := enc.Encode(nodes); err != nil {
		return
	}
}

func (fs *FS) Xhr(lxhr *go9pfs.ListenFileListener) {
	for {
		conn, err := lxhr.Accept()
		if err != nil {
			log.Errorf("xhr: accept: %v", err)
			continue
		}
		go fs.xhr(conn)
	}
}

func allowed(h http.Header, reqHost, origHost string) bool {
	if reqHost == origHost {
		return true
	}
	alOrig := h.Get("access-control-allow-origin")
	return alOrig == "*"
}

func (fs *FS) xhr(conn net.Conn) {
	r := bufio.NewReader(conn)
	defer conn.Close()

	req, err := http.ReadRequest(r)
	if err != nil {
		log.Errorf("read request: %v", err)
		return
	}
	log.Infof("xhr: req: %v", req)
	url := req.URL
	url.Host = req.Host
	if h := url.Host; h == "" {
		url.Host = fs.Fetcher.Origin().Host
	}
	url.Scheme = "https"
	proxyReq, err := http.NewRequest(req.Method, url.String(), req.Body)
	if err != nil {
		log.Errorf("new request: %v", err)
		return
	}
	proxyReq.Header.Set("Host", req.Host)
	for header, values := range req.Header {
		for _, value := range values {
			proxyReq.Header.Add(header, value)
		}
	}
	resp, err := fs.Client.Do(proxyReq)
	if err != nil {
		log.Errorf("do request: %v", err)
		return
	}
	if h := url.Host; !allowed(resp.Header, h, fs.Fetcher.Origin().Host) {
		log.Errorf("no cross-origin request: %v", h)
		return
	}
	if err := resp.Write(conn); err != nil {
		log.Errorf("write response: %v", err)
		return
	}
}

func (fs *FS) Update(uri, html string, css []string, js []string) {
	fs.c.L.Lock()
	defer fs.c.L.Unlock()

	if fs.cssDir == nil && fs.jsDir == nil {
		fs.c.Wait()
	}

	fs.url = uri
	fs.htm = html
	if fs.cssDir != nil {
		for name := range fs.cssDir.Children() {
			fs.cssDir.DeleteChild(name)
		}
		for i, s := range css {
			fn := fmt.Sprintf("%d.css", i)
			f := go9pfs.NewStaticFile(
				fs.oFS.NewStat(fn, fs.un, fs.gn, 0400),
				[]byte(s),
			)
			fs.cssDir.AddChild(f)
		}
	}
	if fs.jsDir != nil {
		for name := range fs.jsDir.Children() {
			fs.jsDir.DeleteChild(name)
		}
		for i, s := range js {
			fn := fmt.Sprintf("%d.js", i)
			f := go9pfs.NewStaticFile(
				fs.oFS.NewStat(fn, fs.un, fs.gn, 0400),
				[]byte(s),
			)
			fs.jsDir.AddChild(f)
		}
	}
}
