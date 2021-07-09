package js

import (
	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"github.com/psilva261/opossum"
	"github.com/psilva261/opossum/logger"
	"github.com/psilva261/opossum/nodes"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"time"
)

var DebugDumpJS *bool
var log *logger.Logger
var timeout = 60*time.Second

func SetLogger(l *logger.Logger) {
	log = l
}

type ReadWriteCloser struct {
	io.Reader
	io.Writer
	io.Closer
}

var (
	fetcher   opossum.Fetcher
	nt        *nodes.Node

	fsys   *client.Fsys
	cancel context.CancelFunc
)

func NewJS(html string, fetcher opossum.Fetcher, nn *nodes.Node) {
	nt = nn
	return
}

func call(fn, cmd string, args... string) (resp string, err error) {
	fid, err := fsys.Open(fn, plan9.ORDWR)
	if err != nil {
		return
	}
	defer fid.Close()
	fid.Write([]byte(cmd+"\n"))
	for _, arg := range args {
		fid.Write([]byte(arg+"\n"))
	}
	r := bufio.NewReader(fid)
	b := bytes.NewBuffer([]byte{})
	_, err = io.Copy(b, r)
	if err != nil && !strings.Contains(err.Error(), io.ErrClosedPipe.Error()) {
		return "", fmt.Errorf("unexpected error: %v", err)
	}
	return b.String(), nil
}

// Start with pre-defined scripts
func Start(scripts ...string) (resHtm string, changed bool, err error) {
	args := make([]string, 0, len(scripts)+2)
	args = append(args, "-s", "opossum")
	log.Infof("Start gojafs")

	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "gojafs", args...)
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", false, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", false, fmt.Errorf("stdout pipe: %w", err)
	}
	rwc := &ReadWriteCloser{
		Reader: stdout,
		Writer: stdin,
		Closer: stdin,
	}

	log.Infof("cmd.Start...")
	if err = cmd.Start(); err != nil {
		return "", false, fmt.Errorf("start: %w", err)
	}
	// Prevent Zombie processes after stopping
	go cmd.Wait()

	conn, err := client.NewConn(rwc)
	if err != nil {
		return "", false, fmt.Errorf("new conn: %w", err)
	}
	log.Infof("cmd.Connected...")
	u, err := user.Current()
	if err != nil {
		return
	}
	un := u.Username
	fsys, err = conn.Attach(nil, un, "")
	if err != nil {
		return
	}
	log.Infof("cmd.Attached...")

	resp, err := call("ctl", "start")
	if err != nil {
		return "", false, fmt.Errorf("%v", err)
	}

	if resp != "" {
		resHtm = resp
		changed = true
	}

	return
}

func Stop() {
	log.Infof("Stop devjs")
	if cancel != nil {
		cancel()
	}
}

func printCode(code string, maxWidth int) {
	if maxWidth > len(code) {
		maxWidth = len(code)
	}
	log.Infof("js code: %v", code[:maxWidth])
}

// TriggerClick, and return the result html
// ...then HTML5 parse it, diff the node tree
// (probably faster and cleaner than anything else)
func TriggerClick(selector string) (newHTML string, ok bool, err error) {
	newHTML, err = call("ctl", "click", selector)
	ok = newHTML != "" && err == nil
	return
}

func Srcs(doc *nodes.Node) (srcs []string) {
	srcs = make([]string, 0, 3)

	iterateJsElements(doc, func(src, inlineCode string) {
		if src = strings.TrimSpace(src); src != "" && !blocked(src) {
			srcs = append(srcs, src)
		}
	})

	return
}

func blocked(src string) bool {
	for _, s := range []string{
		"adsense",
		"adsystem",
		"adservice",
		"googletagservice",
		"googletagmanager",
		"script.ioam.de",
		"googlesyndication",
		"adserver",
		"nativeads",
		"prebid",
		".ads.",
		"google-analytics.com",
	} {
		if strings.Contains(src, s) {
			return true
		}
	}
	return false
}

func Scripts(doc *nodes.Node, downloads map[string]string) (codes []string) {
	codes = make([]string, 0, 3)

	iterateJsElements(doc, func(src, inlineCode string) {
		if strings.TrimSpace(inlineCode) != "" {
			log.Infof("JS.Scripts: inline code:")
			printCode(inlineCode, 20)
			codes = append(codes, inlineCode)
		} else if c, ok := downloads[src]; ok {
			log.Infof("JS.Scripts: referenced code (%v)", src)
			codes = append(codes, c)
		}
	})

	return
}

func iterateJsElements(doc *nodes.Node, fn func(src string, inlineCode string)) {
	var f func(n *nodes.Node)
	f = func(n *nodes.Node) {
		if n.Type() == html.ElementNode && n.Data() == "script" {
			isJS := true
			src := ""

			for _, a := range n.Attrs {
				switch strings.ToLower(a.Key) {
				case "type":
					t, err := opossum.NewContentType(a.Val, nil)
					if err != nil {
						log.Printf("t: %v", err)
					}
					if a.Val == "" || t.IsJS() {
						isJS = true
					} else {
						isJS = false
					}
				case "src":
					src = a.Val
				}
			}

			if isJS {
				fn(src, n.ContentString(true))
			}
		}
		for _, c := range n.Children {
			f(c)
		}
	}

	f(doc)

	return
}

// AJAX:
// https://stackoverflow.com/questions/7086858/loading-ajax-app-with-jsdom

// Babel on Goja:
// https://github.com/dop251/goja/issues/5#issuecomment-259996573

// Goja supports ES5.1 which is essentially JS assembly:
// https://github.com/dop251/goja/issues/76#issuecomment-399253779
