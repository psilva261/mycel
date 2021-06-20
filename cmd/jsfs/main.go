// js package as separate program (very wip)
package main

import (
	"bufio"
	"fmt"
	"github.com/knusbaum/go9p"
	"github.com/knusbaum/go9p/fs"
	"github.com/psilva261/opossum"
	"github.com/psilva261/opossum/domino"
	"github.com/psilva261/opossum/logger"
	"io"
	"net"
	"os"
	"os/user"
	"strings"
)

var (
	d *domino.Domino
	log *logger.Logger
	htm string
	js []string
)

func init() {
	f := false
	t := true
	domino.DebugDumpJS = &t
	logger.Quiet = &f
	logger.Init()
	log = &logger.Logger{Debug: true}
	domino.SetLogger(log)
}

func usage() {
	log.Printf("usage: jsfs -h htmlfile jsfile1 [jsfile2 [..]]")
	os.Exit(1)
}

func Main(r io.Reader, w io.Writer) (err error) {
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("get user: %v", err)
	}
	un := u.Username
	gn, err := opossum.Group(u)
	if err != nil {
		return fmt.Errorf("get group: %v", err)
	}

	jsFS, root := fs.NewFS(un, gn, 0500)
	c := fs.NewListenFile(jsFS.NewStat("ctl", un, gn, 0600))
	root.AddChild(c)
	lctl := (*fs.ListenFileListener)(c)
	go Ctl(lctl)
	go func() {
		err := go9p.ServeReadWriter(r, w, jsFS.Server())
		if err != nil {
			log.Errorf("jsfs: serve rw: %v", err)
		}
	}()

	return
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
		log.Errorf("jsfs: read string: %v", err)
		return
	}
	l = strings.TrimSpace(l)

	switch l {
	case "start":
		d = domino.NewDomino(htm, nil, nil)
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
		log.Printf("jsfs: processJS: changed = %v", changed)
		if changed {
			w.WriteString(resHtm)
			w.Flush()
		}
	case "stop":
		if d != nil {
			d.Stop()
			d = nil
		}
	default:
		log.Errorf("unknown cmd")
	}
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

	if err := Main(os.Stdin, os.Stdout); err != nil {
		log.Fatalf("Main: %+v", err)
	}
}
