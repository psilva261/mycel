package domino

import (
	"fmt"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/jvatic/goja-babel"
	"golang.org/x/net/html"
	"io/ioutil"
	"log"
	"opossum"
	"opossum/nodes"
	"strconv"
	"strings"
	"time"
)

var DebugDumpJS *bool

type Domino struct {
	loop       *eventloop.EventLoop
	vm         *goja.Runtime
	html       string
	outputHtml string
	domChanged chan int
}

func NewDomino(html string) (d *Domino) {
	d = &Domino{
		html: html,
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
		log.Printf("line %v, column %v", y, x)
		lines := strings.Split(script, "\n")

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
			log.Printf("%v: %v", y, lines[y])
			if y+1 < len(lines) && len(lines[y+1]) < 120 {
				log.Printf("%v: %v", y+1, lines[y+1])
			}
		}
	}
}

func (d *Domino) Exec(script string) (err error) {
	script = strings.Replace(script, "const ", "var ", -1)
	script = strings.Replace(script, "let ", "var ", -1)
	script = strings.Replace(script, "<!--", "", -1)
	SCRIPT := `
	    global = {};
	    //global.__domino_frozen__ = true; // Must precede any require('domino')
	    var domino = require('domino-lib/index');
	    var Element = domino.impl.Element; // etc

	    // JSDOM also knows the style tag
	    // https://github.com/jsdom/jsdom/issues/2485
		Object.assign(this, domino.createWindow(s.html, 'http://example.com'));
		window = this;
		window.parent = window;
		window.top = window;
		window.self = window;
		addEventListener = function() {};
		window.location.href = 'http://example.com';
		navigator = {};
		HTMLElement = domino.impl.HTMLElement;
	    // Fire DOMContentLoaded
	    // to trigger $(document)readfy!!!!!!!
	    document.close();
	` + script
	if *DebugDumpJS {
		ioutil.WriteFile("main.js", []byte(SCRIPT), 0644)
	}
	prg, err := goja.Compile("main.js", SCRIPT, false)
	if err != nil {
		IntrospectError(err, SCRIPT)
		return fmt.Errorf("compile: %w", err)
	}
	ready := make(chan int)
	go func() {
		d.loop.RunOnLoop(func(vm *goja.Runtime) {
			log.Printf("RunOnLoop")
			registry := require.NewRegistry(
				require.WithGlobalFolders(".", ".."),
			)
			console.Enable(vm)
			req := registry.Enable(vm)
			_ = req

			vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
			type S struct {
				Buf  string `json:"buf"`
				HTML string `json:"html"`
			}
			d.vm = vm

			vm.Set("s", S{
				HTML: d.html,
				Buf:  "yolo",
			})
			_, err := vm.RunProgram(prg)
			if err != nil {
				log.Printf("run program: %v", err)
				IntrospectError(err, script)
			}
			ready <- 1
		})
	}()
	<-ready
	<-time.After(10 * time.Millisecond)
	//res = fmt.Sprintf("%v", v.Export())
	if _, _, err = d.TrackChanges(); err != nil {
		return fmt.Errorf("track changes: %w", err)
	}
	return
}

func (d *Domino) Exec6(script string) (err error) {
	babel.Init(4) // Setup 4 transformers (can be any number > 0)
	r, err := babel.Transform(strings.NewReader(script), map[string]interface{}{
		"plugins": []string{
			"transform-react-jsx",
			"transform-es2015-block-scoping",
		},
	})
	if err != nil {
		return fmt.Errorf("babel: %v", err)
	}
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read all: %v", err)
	}
	return d.Exec(string(buf))
}

func (d *Domino) Export(expr string) (res string, err error) {
	v, err := d.vm.RunString(expr)
	if err != nil {
		return "", fmt.Errorf("export: %w", err)
	}
	if v != nil {
		res = fmt.Sprintf("%v", v.Export())
	}
	return
}

// TriggerClick, and return the result html
// ...then HTML5 parse it, diff the node tree
// (probably faster and cleaner than anything else)
func (d *Domino) TriggerClick(selector string) (newHTML string, ok bool, err error) {
	res, err := d.vm.RunString(`
		var sel = '` + selector + `';
		console.log('sel=' + sel);
		var sell = document.querySelector(sel);
		console.log('sell=' + sell);
		var selfn = sell.click.bind(sell);
		console.log('selfn=' + selfn);
		if (selfn) {
			selfn();
		}
		!!selfn;
	`)

	ok = fmt.Sprintf("%v", res) == "true"

	return
}

// Put change into html (e.g. from input field mutation)
func (d *Domino) PutAttr(selector, attr, val string) (ok bool, err error) {
	res, err := d.vm.RunString(`
		var sel = '` + selector + `';
		console.log('sel=' + sel);
		var sell = document.querySelector(sel);
		console.log('sell=' + sell);
		sell.attr('` + attr + `', '` + val + `');
		!!sell;
	`)

	ok = fmt.Sprintf("%v", res) == "true"

	return
}

func (d *Domino) TrackChanges() (html string, changed bool, err error) {
	html, err = d.Export("document.querySelector('html').innerHTML;")
	if err != nil {
		return
	}
	changed = d.outputHtml != html
	d.outputHtml = html
	return
}

// https://stackoverflow.com/a/26716182
// TODO: eval is evil
func (d *Domino) ExecInlinedScripts() (err error) {
	return d.Exec(`
	navigator = {};

    var scripts = Array.prototype.slice.call(document.getElementsByTagName("script"));
    for (var i = 0; i < scripts.length; i++) {
        if (scripts[i].src != "") {
            var tag = document.createElement("script");
            tag.src = scripts[i].src;
            document.getElementsByTagName("head")[0].appendChild(tag);
        }
        else {
        	try {
            	eval.call(window, scripts[i].innerHTML);
            } catch(e) {
            	console.log(e);
            }
        }
    }
	`)
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
	for _, s := range []string{"adsense", "adsystem", "adservice", "googletagservice", "googletagmanager", "script.ioam.de","googlesyndication","adserver", "nativeads", "prebid", ".ads."} {
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

			for _, a := range n.Attr {
				switch strings.ToLower(a.Key) {
				case "type":
					t, err := opossum.NewContentType(a.Val)
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

// AJAX:
// https://stackoverflow.com/questions/7086858/loading-ajax-app-with-jsdom

// Babel on Goja:
// https://github.com/dop251/goja/issues/5#issuecomment-259996573

// Goja supports ES5.1 which is essentially JS assembly:
// https://github.com/dop251/goja/issues/76#issuecomment-399253779
