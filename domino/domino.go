package domino

import (
	"errors"
	"fmt"
	"github.com/dop251/goja"
	"github.com/dop251/goja/parser"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/jvatic/goja-babel"
	"golang.org/x/net/html"
	"io/ioutil"
	"github.com/psilva261/opossum"
	"github.com/psilva261/opossum/logger"
	"github.com/psilva261/opossum/nodes"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var DebugDumpJS *bool
var log *logger.Logger
var timeout = 60*time.Second

func SetLogger(l *logger.Logger) {
	log = l
}

type Mutation struct {
	time.Time
	Type int
	Sel string
}

type Domino struct {
	fetcher   opossum.Fetcher
	loop       *eventloop.EventLoop
	html       string
	nt           *nodes.Node
	outputHtml string
	domChange chan Mutation
}

func NewDomino(html string, fetcher opossum.Fetcher, nt *nodes.Node) (d *Domino) {
	d = &Domino{
		html: html,
		fetcher: fetcher,
		nt: nt,
		domChange: make(chan Mutation, 100),
	}
	return
}

func (d *Domino) Start() {
	log.Printf("Start event loop")
	d.loop = eventloop.NewEventLoop()

	d.loop.Start()
	log.Printf("event loop started")
}

func (d *Domino) Stop() {
	d.loop.Stop()
}

func IntrospectError(err error, script string) {
	prefix := "Line "
	i := strings.Index(err.Error(), prefix)
	if i > 0 {
		i += len(prefix)
		s := err.Error()[i:]
		yxStart := strings.Split(s, " ")[0]
		yx := strings.Split(yxStart, ":")
		y, _ := strconv.Atoi(yx[0])
		x, _ := strconv.Atoi(yx[1])
		lines := strings.Split(script, "\n")

		if y - 1 > len(lines) - 1 {
			y = len(lines)
		}

		if wholeLine := lines[y-1]; len(wholeLine) > 100 {
			from := x - 50
			to := x + 50
			if from < 0 {
				from = 0
			}
			if to >= len(wholeLine) {
				to = len(wholeLine) - 1
			}
			log.Printf("the line: %v", wholeLine[from:to])
		} else {
			if y > 0 && len(lines[y-1]) < 120 {
				log.Printf("%v: %v", y-1, lines[y-1])
			}
			if y < len(lines) {
				log.Printf("%v: %v", y, lines[y])
			}
			if y+1 < len(lines) && len(lines[y+1]) < 120 {
				log.Printf("%v: %v", y+1, lines[y+1])
			}
		}
	}
}

func printCode(code string, maxWidth int) {
	if maxWidth > len(code) {
		maxWidth = len(code)
	}
	log.Infof("js code: %v", code[:maxWidth])
}

func srcLoader(fn string) ([]byte, error) {
	path := filepath.FromSlash(fn)
	if !strings.Contains(path, "domino-lib/") || !strings.HasSuffix(path, ".js") {
		return nil, require.ModuleFileDoesNotExistError
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) || errors.Is(err, syscall.EISDIR) {
			err = require.ModuleFileDoesNotExistError
		} else {
			log.Errorf("srcLoader: handling of require('%v') is not implemented", fn)
		}
	}
	return data, err
}

func (d *Domino) Exec(script string, initial bool) (res string, err error) {
	script = strings.Replace(script, "const ", "var ", -1)
	script = strings.Replace(script, "let ", "var ", -1)
	script = strings.Replace(script, "<!--", "", -1)
	SCRIPT := `
		global = {};
		//global.__domino_frozen__ = true; // Must precede any require('domino')
		var domino = require('domino-lib/index');
		var Element = domino.impl.Element; // etc

		Object.assign(this, domino.createWindow(opossum.html, 'http://example.com'));
		window = this;
		window.parent = window;
		window.top = window;
		window.self = window;
		addEventListener = function() {};
		removeEventListener = function() {};
		window.location.href = 'http://example.com';
		var ___fq;
		___fq = function(pre, el) {
			var i, p;
			
			if (!el) {
				return undefined;
			}
			p = el.parentElement;

			if (p) {
				for (i = 0; i < p.children.length; i++) {
					if (p.children[i] === el) {
						return ___fq('', p) + ' > :nth-child(' + (i+1) + ')';
					}
				}
			} else {
				return el.tagName;
			}
		};
		document._setMutationHandler(function(a) {
			// a provides attributes type, target and node or attr
			// (cf Object.keys(a))
			opossum.mutated(a.type, ___fq('yolo', a.target));
		});
		window.getComputedStyle = function(el, pseudo) {
			this.el = el;
			this.getPropertyValue = function(prop) {
				return opossum.style(___fq('', el), pseudo, prop, arguments[2]);
			};
			return this;
		};
		Element.prototype.getClientRects = function() { /* I'm a stub */ return []; }
		window.screen = {
			width: 1280,
			height: 1024
		};
		window.screenX = 0;
		window.screenY = 25;
		location = window.location;
		navigator = {
			platform: 'plan9(port)',
			userAgent: 'opossum'
		};
		HTMLElement = domino.impl.HTMLElement;

		function XMLHttpRequest() {
			var _method, _uri;
			var h = {};
			var ls = {};

			this.readyState = 0;

			var cb = function(data, err) {
				if (data !== '') {
					this.responseText = data;
					this.readyState = 4;
					this.state = 200;
					this.status = 200;
					if (ls['load']) ls['load'].bind(this)();
					if (this.onload) this.onload.bind(this)();
					if (this.onreadystatechange) this.onreadystatechange.bind(this)();
				}
			}.bind(this);

			this.addEventListener = function(k, fn) {
				ls[k] = fn;
			};
			this.open = function(method, uri) {
				_method = method;
				_uri = uri;
			};
			this.setRequestHeader = function(k, v) {
				h[k] = v;
			};
			this.send = function(data) {
				opossum.xhr(_method, _uri, h, data, cb);
				this.readyState = 2;
			};
			this.getAllResponseHeaders = function() {
				return '';
			};
		}
	` + script
	if !initial {
		SCRIPT = script
	}

	if *DebugDumpJS {
		ioutil.WriteFile("main.js", []byte(SCRIPT), 0644)
	}

	ready := make(chan goja.Value)
	errCh := make(chan error)
	intCh := make(chan int)
	go func() {
		d.loop.RunOnLoop(func(vm *goja.Runtime) {
			log.Printf("RunOnLoop")

			if initial {
				vm.SetParserOptions(parser.WithDisableSourceMaps)

				// find domino-lib folder
				registry := require.NewRegistry(
					require.WithGlobalFolders(
						".",     // standalone
						"..",    // tests
						"../..", // go run
					),
					require.WithLoader(
						require.SourceLoader(srcLoader),
					),
				)

				console.Enable(vm)
				registry.Enable(vm)

				type S struct {
					Buf  string `json:"buf"`
					HTML string `json:"html"`
					Referrer func() string `json:"referrer"`
					Style func(string, string, string, string) string `json:"style"`
					XHR func(string, string, map[string]string, string, func(string, string)) `json:"xhr"`
					Mutated func(int, string) `json:"mutated"`
				}

				vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
				vm.Set("opossum", S{
					HTML: d.html,
					Buf:  "yolo",
					Referrer: func() string { return "https://example.com" },
					Style: func(sel, pseudo, prop, prop2 string) string {
						res, err := d.nt.Query(sel)
						if err != nil {
							log.Errorf("query %v: %v", sel, err)
							return ""
						}
						if len(res) != 1 {
							log.Errorf("query %v: %v", res, err)
							return ""
						}
						return res[0].Css(prop)
					},
					XHR: d.xhr,
					Mutated: d.mutated,
				})
			}

			go func() {
				for _ = range intCh {
					vm.Interrupt("halt")
				}
			}()

			vv, err := vm.RunString(SCRIPT)
			if err != nil {
				IntrospectError(err, script)
				errCh <- fmt.Errorf("run program: %w", err)
			} else {
				ready <- vv
			}
		})
	}()

	for {
		select {
			case v := <-ready:
				log.Infof("ready")
				<-time.After(10 * time.Millisecond)
				if v != nil {
					res = v.String()
				}
				goto cleanup
			case er := <- errCh:
				log.Infof("err")
				<-time.After(10 * time.Millisecond)
				err = fmt.Errorf("event loop: %w", er)
				goto cleanup
			case <-time.After(timeout):
				log.Errorf("Interrupt JS after %v", timeout)
				intCh <- 1
		}
	}

cleanup:
	close(ready)
	close(errCh)
	close(intCh)

	return
}

func (d *Domino) Exec6(script string) (res string, err error) {
	babel.Init(4) // Setup 4 transformers (can be any number > 0)
	r, err := babel.Transform(strings.NewReader(script), map[string]interface{}{
		"plugins": []string{
			"transform-react-jsx",
			"transform-es2015-block-scoping",
		},
	})
	if err != nil {
		return "", fmt.Errorf("babel: %v", err)
	}
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read all: %v", err)
	}
	return d.Exec(string(buf), true)
}

// CloseDoc fires DOMContentLoaded to trigger $(document).ready(..)
func (d *Domino) CloseDoc() (err error) {
	_, err = d.Exec("if (this.document) document.close();", false)
	return
}

// TriggerClick, and return the result html
// ...then HTML5 parse it, diff the node tree
// (probably faster and cleaner than anything else)
func (d *Domino) TriggerClick(selector string) (newHTML string, ok bool, err error) {
	res, err := d.Exec(`
		var sel = '` + selector + `';
		var el = document.querySelector(sel);

		console.log('query ' + sel);

		if (!el) {
			console.log('el is null/undefined');
			null;
		} else if (el._listeners && el._listeners.click) {
			var fn = el.click.bind(el);

			if (fn) {
				console.log('  call click handler...');
				fn();
			}

			!!fn;
		} else {
			false;
		}
	`, false)

	if ok = res == "true"; ok {
		newHTML, ok, err = d.TrackChanges()
	}

	return
}

// Put change into html (e.g. from input field mutation)
func (d *Domino) PutAttr(selector, attr, val string) (ok bool, err error) {
	res, err := d.Exec(`
		var sel = '` + selector + `';
		var el = document.querySelector(sel);
		el.attr('` + attr + `', '` + val + `');
		!!el;
	`, false)

	ok = res == "true"

	return
}

func (d *Domino) TrackChanges() (html string, changed bool, err error) {
	outer:
	for {
		select {
		case m := <-d.domChange:
			log.Printf("mutation received @ %v for %v", m.Time, m.Sel)
			changed = true
		default:
			break outer
		}
	}
	
	if changed {
		html, err = d.Exec("document.querySelector('html').innerHTML;", false)
		if err != nil {
			return
		}
	}
	d.outputHtml = html
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
			log.Infof("domino.Scripts: inline code:")
			printCode(inlineCode, 20)
			codes = append(codes, inlineCode)
		} else if c, ok := downloads[src]; ok {
			log.Infof("domino.Scripts: referenced code (%v)", src)
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
				fn(src, nodes.ContentFrom(*n))
			}
		}
		for _, c := range n.Children {
			f(c)
		}
	}

	f(doc)

	return
}

func (d *Domino) xhr(method, uri string, h map[string]string, data string, cb func(data string, err string)) {
	c := &http.Client{}
	u, err := d.fetcher.LinkedUrl(uri)
	if err != nil {
		cb("", err.Error())
		return
	}
	if u.Host != d.fetcher.Origin().Host {
		log.Infof("origin: %v", d.fetcher.Origin())
		log.Infof("uri: %v", uri)
		cb("", "cannot do crossorigin request to " + u.String())
		return
	}
	fmt.Printf("data=%+v\n", data)
	req, err := http.NewRequest(method, u.String(), strings.NewReader(data))
	if err != nil {
		cb("", err.Error())
		return
	}
	for k, v := range h {
		req.Header.Add(k, v)
	}
	resp, err := c.Do(req)
	if err != nil {
		cb("", err.Error())
		return
	}
	defer resp.Body.Close()
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		cb("", err.Error())
		return
	}
	cb(string(bs), "")
}

func (d *Domino) mutated(t int, q string) {
	m := Mutation{
		Time: time.Now(),
		Type: t,
		Sel: q,
	}
	log.Printf("mutation received: %+v", m)
	select {
	case d.domChange <- m:
	default:
		log.Printf("dom changes backlog full")
	}
}

// AJAX:
// https://stackoverflow.com/questions/7086858/loading-ajax-app-with-jsdom

// Babel on Goja:
// https://github.com/dop251/goja/issues/5#issuecomment-259996573

// Goja supports ES5.1 which is essentially JS assembly:
// https://github.com/dop251/goja/issues/76#issuecomment-399253779
