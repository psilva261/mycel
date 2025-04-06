package js

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/psilva261/mycel"
	"github.com/psilva261/mycel/logger"
	"github.com/psilva261/mycel/nodes"
	"golang.org/x/net/html"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var timeout = 60 * time.Second

var (
	instances sync.Map
)

type JS struct {
	service string
	cmd *exec.Cmd
	cancel  context.CancelFunc
}

func (js *JS) call(fn, cmd string, args ...string) (resp string, err error) {
	var rwc io.ReadWriteCloser
	for t := 100*time.Millisecond; t < 5*time.Second; t *= 2 {
		rwc, err = js.callSparkleCtl()
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
func Start(f mycel.Fetcher, scripts ...string) (js *JS, resHtm string, changed bool, err error) {
	js = &JS{
		service: fmt.Sprintf("sparkle.%d", os.Getpid()),
	}
	args := make([]string, 0, len(scripts)+2)
	if log.Debug {
		args = append(args, "-v")
	}
	args = append(args, "-s", js.service)
	log.Infof("Start sparklefs")

	var ctx context.Context
	ctx, js.cancel = context.WithCancel(f.Ctx())
	js.cmd = exec.CommandContext(ctx, "sparklefs", args...)
	js.cmd.Stderr = os.Stderr

	log.Infof("cmd.Start...")
	if err = js.cmd.Start(); err != nil {
		return nil, "", false, fmt.Errorf("cmd start: %w", err)
	}

	instances.Store(js, js)

	resp, err := js.call("ctl", "start")
	if err != nil {
		return nil, "", false, fmt.Errorf("call start: %v", err)
	}

	if resp != "" {
		resHtm = resp
		changed = true
	}

	return
}

func StopAll() {
	instances.Range(func(k, _ any) bool {
		js := k.(*JS)
		js.Stop()
		instances.Delete(js)
		return true
	})
}

func (js *JS) Stop() {
	log.Infof("Stop sparklefs")
	js.hangup()
	if js.cancel != nil {
		log.Infof("cancel()")
		js.cancel()
		js.cancel = nil
		if js.cmd != nil {
			// Prevent Zombie processes after stopping
			js.cmd.Wait()
			js.cmd = nil
		}
	}
	instances.Delete(js)
}

// TriggerClick, and return the result html
// ...then HTML5 parse it, diff the node tree
// (probably faster and cleaner than anything else)
func (js *JS) TriggerClick(selector string) (newHTML string, ok bool, err error) {
	newHTML, err = js.call("ctl", "click", selector)
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
					t, err := mycel.NewContentType(a.Val, nil)
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
