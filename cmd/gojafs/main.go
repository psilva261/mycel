// js package as separate program (very wip)
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/knusbaum/go9p"
	"github.com/knusbaum/go9p/fs"
	"github.com/psilva261/opossum"
	"github.com/psilva261/opossum/cmd/gojafs/domino"
	"github.com/psilva261/opossum/logger"
	"github.com/psilva261/opossum/nodes"
	"io"
	"net"
	"net/http"
	"os"
	"os/user"
	"strings"
	"sync"
)

var (
	d *domino.Domino
	log *logger.Logger
	service string
	mtpt    string
	htm     string
	js      []string
	mu sync.Mutex
)

func init() {
	f := false
	t := true
	domino.DebugDumpJS = &f
	logger.Quiet = &t
	logger.Init()
	log = &logger.Logger{Debug: true}
	domino.SetLogger(log)
}

func usage() {
	log.Printf("usage: gojafs [-s service] [-m mtpt] [-h htmlfile jsfile1 [jsfile2] [..]]")
	os.Exit(1)
}

func Main(r io.Reader, w io.Writer) (err error) {
	log.Infof("Main...\n")
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("get user: %v", err)
	}
	un := u.Username
	gn, err := opossum.Group(u)
	if err != nil {
		return fmt.Errorf("get group: %v", err)
	}

	gojafs, root := fs.NewFS(un, gn, 0500)
	c := fs.NewListenFile(gojafs.NewStat("ctl", un, gn, 0600))
	root.AddChild(c)
	lctl := (*fs.ListenFileListener)(c)
	go Ctl(lctl)
	log.Infof("go9p.ServeReadWriter...\n")
	return go9p.ServeReadWriter(r, w, gojafs.Server())
}

func Ctl(lctl *fs.ListenFileListener) {
	for {
		conn, err := lctl.Accept()
		if err != nil {
			log.Errorf("accept: %v", err)
			continue
		}
		go ctl(conn)
	}
}

func ctl(conn net.Conn) {
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	defer conn.Close()

	l, err := r.ReadString('\n')
	if err != nil {
		log.Errorf("gojafs: read string: %v", err)
		return
	}
	l = strings.TrimSpace(l)

	mu.Lock()
	defer mu.Unlock()

	switch l {
	case "start":
		d = domino.NewDomino(htm, xhr, query)
		d.Start()
		initialized := false
		for _, s := range js {
			if _, err := d.Exec(s, !initialized); err != nil {
				if strings.Contains(err.Error(), "halt at") {
					log.Errorf("execution halted: %v", err)
					return
				}
				log.Printf("exec <script>: %v", err)
			}
			initialized = true
		}
		if err := d.CloseDoc(); err != nil {
			log.Errorf("close doc: %v", err)
			return
		}
		resHtm, changed, err := d.TrackChanges()
		if err != nil {
			log.Errorf("track changes: %v", err)
			return
		}
		log.Printf("gojafs: processJS: changed = %v", changed)
		if changed {
			w.WriteString(resHtm)
			w.Flush()
		}
	case "stop":
		if d != nil {
			d.Stop()
			d = nil
		}
	case "click":
		sel, err := r.ReadString('\n')
		if err != nil {
			log.Errorf("gojafs: click: read string: %v", err)
			return
		}
		sel = strings.TrimSpace(sel)
		resHtm, changed, err := d.TriggerClick(sel)
		if err != nil {
			log.Errorf("track changes: %v", err)
			return
		}
		log.Printf("gojafs: processJS: changed = %v", changed)
		if changed {
			w.WriteString(resHtm)
			w.Flush()
		}
	default:
		log.Errorf("unknown cmd")
	}
}

func query(sel, prop string) (val string, err error) {
	log.Infof("query: sel=%+v, prop=%+v\n", sel, prop)
	rwc, err := openQuery()
	if err != nil {
		return "", fmt.Errorf("open query: %w", err)
	}
	defer rwc.Close()
	r := json.NewDecoder(rwc)
	_, err = io.WriteString(rwc, sel + "\n")
	if err != nil {
		return "", fmt.Errorf("write: %w", err)
	}
	log.Printf("devjs: query: sent")
	var res []*nodes.Node
	if err := r.Decode(&res); err != nil {
		return "", fmt.Errorf("decodeeee: >%w<", err)
	}
	if n := len(res); n != 1 {
		return "", fmt.Errorf("query returned %v results", n)
	}
	log.Printf("devjs: query: recvd: %v", res[0])
	return res[0].Css(prop), nil
}

func xhr(req *http.Request) (resp *http.Response, err error) {
	rwc, err := openXhr()
	if err != nil {
		return nil, fmt.Errorf("open xhr: %w", err)
	}
	// defer rwc.Close()
	if err := req.Write(rwc); err != nil {
		return nil, fmt.Errorf("write: %v", err)
	}
	r := bufio.NewReader(rwc)
	if resp, err = http.ReadResponse(r, req); err != nil {
		return nil, fmt.Errorf("read: %v", err)
	}
	return
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
	}

	htmlfile := ""
	jsfiles := make([]string, 0, len(args))

	for len(args) > 0 {
		switch args[0] {
		case "-s":
			service, args = args[1], args[2:]
		case "-h":
			htmlfile, args = args[1], args[2:]
		default:
			var jsfile string
			jsfile, args = args[0], args[1:]
			jsfiles = append(jsfiles, jsfile)
		}
	}

	js = make([]string, 0, len(jsfiles))
	if htmlfile != "" {
		b, err := os.ReadFile(htmlfile)
		if err != nil {
			log.Fatalf(err.Error())
		}
		htm = string(b)
	}
	for _, jsfile := range jsfiles {
		b, err := os.ReadFile(jsfile)
		if err != nil {
			log.Fatalf(err.Error())
		}
		js = append(js, string(b))
	}

	if err := Init(); err != nil {
		log.Fatalf("Init: %+v", err)
	}

	if err := Main(os.Stdin, os.Stdout); err != nil {
		log.Fatalf("Main: %+v", err)
	}
}
