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

var (
	fetcher opossum.Fetcher

	service string
	cmd *exec.Cmd
	cancel  context.CancelFunc
)

func SetFetcher(f opossum.Fetcher) {
	fetcher = f
}

func call(fn, cmd string, args ...string) (resp string, err error) {
	var rwc io.ReadWriteCloser
	for t := 100*time.Millisecond; t < 5*time.Second; t *= 2 {
		rwc, err = callSparkleCtl()
		if err == nil {
			break
		}
		<-time.After(t)
	}
	if err != nil {
		return "", fmt.Errorf("call sparkle ctl: %v", err)
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
	service = fmt.Sprintf("sparkle.%d", os.Getpid())
	args := make([]string, 0, len(scripts)+2)
	args = append(args, "-s", service)
	log.Infof("Start sparklefs")

	var ctx context.Context
	ctx, cancel = context.WithCancel(fetcher.Ctx())
	cmd = exec.CommandContext(ctx, "sparklefs", args...)
	cmd.Stderr = os.Stderr

	log.Infof("cmd.Start...")
	if err = cmd.Start(); err != nil {
		return "", false, fmt.Errorf("cmd start: %w", err)
	}

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
	log.Infof("Stop sparklefs")
	hangup()
	if cancel != nil {
		log.Infof("cancel()")
		cancel()
		cancel = nil
		if cmd != nil {
			// Prevent Zombie processes after stopping
			cmd.Wait()
			cmd = nil
		}
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
