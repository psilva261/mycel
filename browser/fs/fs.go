package fs

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/knusbaum/go9p/fs"
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

var (
	mu *sync.RWMutex
	c  *sync.Cond

	oFS     *fs.FS
	un      string
	gn      string
	url     string
	htm     string
	cssDir  *fs.StaticDir
	jsDir   *fs.StaticDir
	rt      *Node
	Client  *http.Client
	Fetcher mycel.Fetcher
)

func init() {
	mu = &sync.RWMutex{}
	c = sync.NewCond(mu)
	SetDOM(nil)
}

func SetDOM(d *nodes.Node) {
	if rt == nil {
		rt = &Node{
			name: "0",
		}
	}
	rt.nt = d
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

func Srv9p() {
	c.L.Lock()
	var root *fs.StaticDir
	var err error

	un, gn, err = userGroup()
	if err != nil {
		log.Errorf("get user: %v", err)
		c.L.Unlock()
		return
	}
	oFS, root = fs.NewFS(un, gn, 0500)
	u := fs.NewDynamicFile(
		oFS.NewStat("url", un, gn, 0400),
		func() []byte {
			mu.RLock()
			defer mu.RUnlock()

			return []byte(url)
		},
	)
	root.AddChild(u)
	h := fs.NewDynamicFile(
		oFS.NewStat("html", un, gn, 0400),
		func() []byte {
			mu.RLock()
			defer mu.RUnlock()

			return []byte(htm)
		},
	)
	root.AddChild(h)
	d, err := fs.CreateStaticDir(oFS, root, un, "css", 0500|proto.DMDIR, 0)
	if err != nil {
		log.Errorf("create static dir: %v", err)
		c.L.Unlock()
		return
	}
	cssDir = d.(*fs.StaticDir)
	root.AddChild(cssDir)
	d, err = fs.CreateStaticDir(oFS, root, un, "js", 0500|proto.DMDIR, 0)
	if err != nil {
		log.Errorf("create static dir: %v", err)
		c.L.Unlock()
		return
	}
	jsDir = d.(*fs.StaticDir)
	root.AddChild(jsDir)
	q := fs.NewListenFile(oFS.NewStat("query", un, gn, 0600))
	root.AddChild(q)
	lq := (*fs.ListenFileListener)(q)
	root.AddChild(rt)
	go Query(lq)
	if Client != nil {
		xhr := fs.NewListenFile(oFS.NewStat("xhr", un, gn, 0600))
		root.AddChild(xhr)
		lxhr := (*fs.ListenFileListener)(xhr)
		go Xhr(lxhr)
	}
	c.Broadcast()
	c.L.Unlock()

	if err := post(oFS.Server()); err != nil {
		log.Errorf("srv9p: %v", err)
	}
}

func Query(lq *fs.ListenFileListener) {
	for {
		conn, err := lq.Accept()
		if err != nil {
			log.Errorf("query: accept: %v", err)
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

	if rt.nt == nil {
		log.Infof("DOM is nil")
		return
	}
	nodes, err := rt.nt.Query(l)
	if err != nil {
		log.Errorf("query nodes: %v", err)
		return
	}
	if err := enc.Encode(nodes); err != nil {
		return
	}
}

func Xhr(lxhr *fs.ListenFileListener) {
	for {
		conn, err := lxhr.Accept()
		if err != nil {
			log.Errorf("xhr: accept: %v", err)
			continue
		}
		go xhr(conn)
	}
}

func allowed(h http.Header, reqHost, origHost string) bool {
	if reqHost == origHost {
		return true
	}
	alOrig := h.Get("access-control-allow-origin")
	return alOrig == "*"
}

func xhr(conn net.Conn) {
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
		url.Host = Fetcher.Origin().Host
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
	resp, err := Client.Do(proxyReq)
	if err != nil {
		log.Errorf("do request: %v", err)
		return
	}
	if h := url.Host; !allowed(resp.Header, h, Fetcher.Origin().Host) {
		log.Errorf("no cross-origin request: %v", h)
		return
	}
	if err := resp.Write(conn); err != nil {
		log.Errorf("write response: %v", err)
		return
	}
}

func Update(uri, html string, css []string, js []string) {
	c.L.Lock()
	defer c.L.Unlock()

	if cssDir == nil && jsDir == nil {
		c.Wait()
	}

	url = uri
	htm = html
	if cssDir != nil {
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
	}
	if jsDir != nil {
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
}
