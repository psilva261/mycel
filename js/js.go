package js

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/psilva261/opossum"
	"github.com/psilva261/opossum/logger"
	"github.com/psilva261/opossum/nodes"
	"golang.org/x/net/html"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

var timeout = 60 * time.Second

type ReadWriteCloser struct {
	io.Reader
	io.Writer
	io.Closer
}

var (
	fetcher opossum.Fetcher
	nt      *nodes.Node

	service string
	cancel  context.CancelFunc
)

func NewJS(html string, fetcher opossum.Fetcher, nn *nodes.Node) {
	nt = nn
	return
}

func call(fn, cmd string, args ...string) (resp string, err error) {
	rwc, err := callGojaCtl()
	if err != nil {
		return "", fmt.Errorf("call goja ctl: %v", err)
	}
	defer rwc.Close()
	rwc.Write([]byte(cmd + "\n"))
	for _, arg := range args {
		rwc.Write([]byte(arg + "\n"))
	}
	r := bufio.NewReader(rwc)
	b := bytes.NewBuffer([]byte{})
	_, err = io.Copy(b, r)
	if err != nil && !strings.Contains(err.Error(), io.ErrClosedPipe.Error()) {
		return "", fmt.Errorf("unexpected error: %v", err)
	}
	return b.String(), nil
}

// Start with pre-defined scripts
func Start(scripts ...string) (resHtm string, changed bool, err error) {
	service = fmt.Sprintf("goja.%d", os.Getpid())
	args := make([]string, 0, len(scripts)+2)
	args = append(args, "-s", service)
	log.Infof("Start gojafs")

	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "gojafs", args...)
	cmd.Stderr = os.Stderr

	log.Infof("cmd.Start...")
	if err = cmd.Start(); err != nil {
		return "", false, fmt.Errorf("cmd start: %w", err)
	}
	// Prevent Zombie processes after stopping
	go cmd.Wait()

	<-time.After(5 * time.Second)

	resp, err := call("ctl", "start")
	if err != nil {
		return "", false, fmt.Errorf("call start: %v", err)
	}

	if resp != "" {
		resHtm = resp
		changed = true
	}

	return
}

func Stop() {
	log.Infof("Stop gojafs")
	hangup()
	if cancel != nil {
		cancel()
	}
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
			codes = append(codes, inlineCode)
		} else if c, ok := downloads[src]; ok {
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

			for _, a := range n.DomSubtree.Attr {
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
