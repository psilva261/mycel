package browser

import (
	"9fans.net/go/draw"
	"fmt"
	"github.com/mjl-/duit"
	"golang.org/x/net/html"
	"image"
	"net/http"
	"net/url"
	"github.com/chris-ramon/douceur/css"
	"github.com/psilva261/opossum/browser/duitx"
	"github.com/psilva261/opossum/logger"
	"github.com/psilva261/opossum/nodes"
	"github.com/psilva261/opossum/style"
	"strings"
	"testing"
)

var (
	_ duitx.Boxable = &Element{}
)

func init() {
	debugPrintHtml = false
	log.Debug = true
	style.Init(nil)
}

type item struct {
	orig   string
	href   string
	expect string
}

func TestElementClick(t *testing.T) {
	el := Element{}
	for _, b := range []bool{true, false} {
		el.Click = func() (e duit.Event) {
			e.Consumed = b
			return
		}
		if el.click() != b {
			t.Fail()
		}
	}
}

func TestArrange(t *testing.T) {
	htm := `
		<div>
			<h1>title 1</h1>
			<h2>title 2</h2>
			<h3>title 3</h3>
		</div>
	`
	for _, d := range []string{"inline", "block"} {
		doc, err := html.ParseWithOptions(
			strings.NewReader(string(htm)),
			html.ParseOptionEnableScripting(false),
		)
		if err != nil {
			t.Fatalf(err.Error())
		}
		nodeMap := make(map[*html.Node]style.Map)
		nt := nodes.NewNodeTree(doc, style.Map{}, nodeMap, nil)
		h1 := nt.Find("h1")
		h2 := nt.Find("h2")
		h3 := nt.Find("h3")

		m := style.Map{
			Declarations: make(map[string]css.Declaration),
		}
		m.Declarations["display"] = css.Declaration{
			Property: "display",
			Value:    d,
		}
		h1.Map = m
		h2.Map = m
		h3.Map = m

		es := []*Element{
			&Element{n: h1},
			&Element{n: h2},
			&Element{n: h3},
		}
		v := Arrange(nt, es...)
		for _, e := range es {
			if e.n.IsInline() != (d == "inline") {
				t.Fatalf("%+v", e)
			}
		}
		PrintTree(v)
		b := v.UI.(*duitx.Box)
		if len(b.Kids) != 3 {
			t.Fatalf("%v %+v", len(b.Kids), b)
		}
		for _, k := range b.Kids {
			disp := k.UI.(duitx.Boxable).Display()
			if d == "inline" {
				if disp != duitx.Inline {
					t.Fail()
				}
			} else {
				if disp != duitx.Block {
					t.Fail()
				}
			}
		}
	}
}

func TestLinkedUrl(t *testing.T) {
	items := []item{
		item{
			orig:   "https://news.ycombinator.com/item?id=24777268",
			href:   "news",
			expect: "https://news.ycombinator.com/news",
		},
		item{
			orig: "https://golang.org/pkg/",
			href: "net/http",
			expect: "https://golang.org/pkg/net/http",
		},
		item{
			orig: "https://example.com/",
			href: "info",
			expect: "https://example.com/info",
		},
	}

	for _, i := range items {
		b := Browser{}
		origin, err := url.Parse(i.orig)
		if err != nil {
			panic(err.Error())
		}
		b.History.Push(origin, 0)
		res, err := b.LinkedUrl(i.href)
		if err != nil {
			panic(err.Error())
		}
		if res.String() != i.expect {
			t.Fatalf("got %v but expected %v", res, i.expect)
		}
		t.Logf("res=%v, i.expect=%v", res, i.expect)
	}
}

func TestNilPanic(t *testing.T) {
}

func TestNodeToBoxNoscript(t *testing.T) {
	enable := true
	EnableNoScriptTag = enable
	htm := `
		<body>
			<noscript>
				<a href="https://example.com">Link</a>
			</noscript>
			<a>Other</a>
			<input value=123>
		</body>
	`
	doc, err := html.ParseWithOptions(
		strings.NewReader(string(htm)),
		html.ParseOptionEnableScripting(false),
	)
	if err != nil {
		t.Fatalf(err.Error())
	}
	nodeMap := make(map[*html.Node]style.Map)
	body := grep(doc, "body")
	b := &Browser{}
	b.client = &http.Client{}
	browser = b
	u, err := url.Parse("https://example.com")
	if err != nil {
		log.Fatalf("parse: %v", err)
	}
	b.History.Push(u, 0)
	nt := nodes.NewNodeTree(body, style.Map{}, nodeMap, nil)
	boxed := NodeToBox(0, b, nt)
	numInputs := 0
	TraverseTree(boxed, func(ui duit.UI) {
		if _, ok := ui.(*duit.Field); ok {
			numInputs++
		}
	})
	if numInputs != 1 {
		t.Fail()
	}
}

func digestHtm(htm string) (nt *nodes.Node, boxed *Element, err error) {
	doc, err := html.ParseWithOptions(
		strings.NewReader(string(htm)),
		html.ParseOptionEnableScripting(false),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("parse html: %w", err)
	}
	body := grep(doc, "body")
	b := &Browser{}
	b.client = &http.Client{}
	browser = b
	u, err := url.Parse("https://example.com")
	if err != nil {
		return nil, nil, fmt.Errorf("parse url: %w", err)
	}
	b.History.Push(u, 0)
	nm, err := style.FetchNodeMap(doc, style.AddOnCSS, 1280)
	if err != nil {
		return nil, nil, fmt.Errorf("FetchNodeMap: %w", err)
	}

	nt = nodes.NewNodeTree(body, style.Map{}, nm, nil)
	boxed = NodeToBox(0, b, nt)

	return
}

func explodeRow(e *Element) (cols []*duit.Kid, ok bool) {
	for {
		el, ok := e.UI.(*Element)
		if ok {
			e = el
		} else {
			break
		}
	}
	el := e.UI.(*duitx.Box)
	return el.Kids, true
}

func TestInlining(t *testing.T) {
	htm := `
		<body>
			<span id="outer">(<a href="http://example.com"><span>example.com</span></a></span>
		</body>
	`
	nt, boxed, err := digestHtm(htm)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}

	// 1. nodes are row-like
	outerSpan := nt.Find("span")
	if outerSpan.Attr("id") != "outer" || len(outerSpan.Children) != 2 || outerSpan.IsFlex() {
		t.Fatalf(" node")
	}
	bracket := outerSpan.Children[0]
	if bracket.Data() != "(" || !bracket.IsInline() {
		t.Errorf("bracket, is inline: %v", bracket.IsInline())
	}
	a := outerSpan.Children[1]
	if a.Data() != "a" || !a.IsInline() {
		t.Errorf("a, is inline: %v, %+v %+v", a.IsInline(), a, nil)
	}

	// 2. Elements are row-like
	kids, ok := explodeRow(boxed)
	if !ok || len(kids) != 1 {
		t.Errorf("boxed: %+v", boxed)
	}
	kids, ok = explodeRow(kids[0].UI.(*Element))
	if !ok || len(kids) != 2 {
		t.Errorf("boxed: %+v", boxed)
	}
	bel := kids[0].UI.(*Element)
	ael := kids[1].UI.(*Element)
	if bel.n.Data() != "(" {
		t.Errorf("bel: %+v", bel)
	}
	if ael.n.Data() != "a" {
		ael.n.PrintTree()
		t.Errorf("ael: %+v %+v '%v'", ael, ael.n, ael.n.Data())
	}
	if !ael.IsLink || ael.Click == nil {
		t.Errorf("ael: %+v %+v '%v'", ael, ael.n, ael.n.Data())
	}
}

func TestInlining2(t *testing.T) {
	htm := `
		<body>
			<span id="outer">
				<span id="sp1">[</span>
				<a href="http://example.com">edit</a>
				<span id="sp2">]</span>
			</span>
		</body>
	`
	nt, boxed, err := digestHtm(htm)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}

	// 1. nodes are row-like
	outerSpan := nt.Find("span")
	if outerSpan.Attr("id") != "outer" || len(outerSpan.Children) != 7 || outerSpan.IsFlex() {
		t.Errorf("node: %+v", outerSpan)
	}
	bracket := outerSpan.Children[0]
	if /*bracket.Data() != "(" || */!bracket.IsInline() {
		t.Errorf("bracket, is inline: %v %+v %+v", bracket.IsInline(), bracket, bracket.Data())
	}
	sp1 := outerSpan.Children[1]
	if sp1.Data() != "span" || !sp1.IsInline() {
		t.Errorf("sp1, is inline: %v, %+v %+v", sp1.IsInline(), sp1, sp1.Data())
	}

	// 2. Elements are row-like
	kids, ok := explodeRow(boxed)
	if !ok || len(kids) != 1 {
		t.Errorf("boxed: %+v, kids: %+v", boxed, kids)
	}
	kids, ok = explodeRow(kids[0].UI.(*Element))
	if !ok || len(kids) != 3 {
		t.Errorf("boxed: %+v, kids: %+v", boxed, kids)
	}
	sel := kids[0].UI.(*Element)
	ael := kids[1].UI.(*Element)
	if sel.n.Data() != "span" {
		t.Errorf("sel: %+v", sel)
	}
	if ael.n.Data() != "a" {
		ael.n.PrintTree()
		t.Errorf("ael: %+v %+v", ael, ael.n)
	}
	if !ael.IsLink || ael.Click == nil {
		t.Errorf("ael: %+v %+v", ael, ael.n)
	}
}

func TestInlining3(t *testing.T) {
	htm := `
		<body>
			<p>
				<span>
					<tt>bind&nbsp;-ac&nbsp;/dist/plan9front&nbsp;/</tt>
				</span>
			</p>
			<p>
				<span>
					<tt>git/pull&nbsp;-u&nbsp;gits://git.9front.org/plan9front/plan9front</tt>
				</span>
			</p>
		</body>
	`
	nt, boxed, err := digestHtm(htm)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}

	// 1. nodes are 2 rows
	ps := nt.FindAll("p")
	if len(ps) != 2 || ps[0].IsInline() || ps[1].IsInline() {
		t.Fail()
	}
	// 1a. nodes' children are inline
	for i := 0; i < 2; i++ {
		p := ps[i]
		span := p.Find("span")
		tt := span.Find("tt")
		if !span.IsInline() || !tt.IsInline() {
			t.Fail()
		}
	}
	
	PrintTree(boxed)
	// 2. Elements are 2 rows

	kids, ok := explodeRow(boxed)
	if !ok || len(kids) != 2 {
		t.Errorf("boxed: %+v, kids: %+v", boxed, kids)
	}
	for _, k := range kids {
		if k.UI.(duitx.Boxable).Display() != duitx.Block {
			t.Fail()
		}
	}
}

func TestSpansCanBeWrapped(t *testing.T) {
	htm := `
		<body>
			<span>
				A text with multiple words.
			</span>
		</body>
	`
	_, boxed, err := digestHtm(htm)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}

	n := 0

	TraverseTree(boxed, func(ui duit.UI) {
		if el, ok := ui.(*Label); ok {
			n++
			fmt.Printf("n data=%v\n", el.n.Data())
			fmt.Printf("n cls=%v\n", el.n.Attr("class"))
		}
	})
	if n != 5 {
		t.Errorf("%v", n)
	}
}

func TestAlwaysOneElement(t *testing.T) {
	h := `
		<!DOCTYPE html>
		<html>
			<body>
				<div class="wrapper">
					<main>main content</main>
					<footer>footer</footer>
				</div>
			</body>
		</html>
	`
	_, boxed, err := digestHtm(h)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}
	n := 0

	TraverseTree(boxed, func(ui duit.UI) {
		if el, ok := ui.(*Element); ok && el.n.Attr("class") == "wrapper" {
			n++
			fmt.Printf("n data=%v\n", el.n.Data())
			fmt.Printf("n cls=%v\n", el.n.Attr("class"))
		}
	})
	if n != 1 {
		t.Errorf("%v", n)
	}
}

func TestTextArea(t *testing.T) {
	htm := `
		<textarea height="100">
		</textarea>
	`
	nt, _, err := digestHtm(htm)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}

	ta := nt.Find("textarea")
	el := NewTextArea(ta)
	// Trigger key press to trigger call to changed
	el.Key(nil, nil, 'a', draw.Mouse{}, image.Point{})
}

func TestNewPicture(t *testing.T) {
	htm := `
	<picture itemprop="contentUrl">
		<source srcset="https://example.com/2040 2040w,https://example.com/1880 1880w,https://example.com/1400 1400w" media="(-webkit-min-device-pixel-ratio: 1.25), (min-resolution: 120dpi)">
    		<source srcset="https://example.com/1020 1020w,https://example.com/940 940w,https://example.com/700 700w">
    		<img src="https://example.com/465" height="5000" width="7000" loading="lazy">
    	</picture>
	`
	nt, _, err := digestHtm(htm)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}

	p := nt.Find("picture")
	src := newPicture(p)
	if src != "https://example.com/700" {
		t.Error()
	}
}

func TestSrcSet(t *testing.T) {
	htm := `
		<img width="800" height="429" style="width: 600px" src="/t.jpg" srcset="/t.jpg 800w, /t-300x165.jpg 300w, /t-768x421.jpg 768w, /t-561x308.jpg 561w, /t-364x200.jpg 364w, /t-728x399.jpg 728w, /t-608x334.jpg 608w, /t-758x416.jpg 758w, /t-87x48.jpg 87w, /t-175x96.jpg 175w, /t-313x172.jpg 313w" sizes="(max-width: 800px) 100vw, 800px">
	`
	nt, _, err := digestHtm(htm)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}
	img := nt.Find("img")
	w, src := srcSet(img)
	t.Logf("%v, %v, %v", nt.Data(), w, src)
	if w != 608 || src != "/t-608x334.jpg" {
		t.Error()
	}
}

func TestWidths(t *testing.T) {
	htm := `
<html>
	<body style="width: 100%">
		<h1>
			Info
		</h1>
		<main style="width: 50%">
			<nav style="width: 33%">
			</nav>
			<article id="lo">
				<h2>
					General information
				</h2>
				<p style="width: 90%">
					Supplementary information
				</p>
			</article>
		</main>
	</body>
</html>
	`
	nt, _, err := digestHtm(htm)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}
	if nt.Data() != "body" || nt.Width() != 1280 {
		t.Fail()
	}
	if main := nt.Find("main"); main.Width() != 640 {
		t.Fail()
	}
	if p := nt.Find("p"); p.Width() != 576 {
		t.Fail()
	}
}
