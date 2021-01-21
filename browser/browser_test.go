package browser

import (
	"github.com/mjl-/duit"
	"golang.org/x/net/html"
	"net/http"
	"net/url"
	"github.com/chris-ramon/douceur/css"
	"github.com/psilva261/opossum/logger"
	"github.com/psilva261/opossum/nodes"
	"github.com/psilva261/opossum/style"
	"strings"
	"testing"
)

func init() {
	quiet := false
	logger.Quiet = &quiet
	js := false
	ExperimentalJsInsecure = &js
	logger.Init()
	SetLogger(&logger.Logger{})
}

type item struct {
	orig   string
	href   string
	expect string
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
		if d == "inline" {
			b := v.UI.(*duit.Box)
			if len(b.Kids) != 3 {
				t.Fatalf("%+v", b)
			}
		} else {
			if g := v.UI.(*duit.Grid); g.Columns != 1 || len(g.Kids) != 3 {
				t.Fatalf("%+v", g)
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
	}

	for _, i := range items {
		b := Browser{}
		origin, err := url.Parse(i.orig)
		if err != nil {
			panic(err.Error())
		}
		b.History.Push(origin)
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
	//f, err := os.Open()
}

func TestNodeToBoxNoscript(t *testing.T) {
	enable := true
	EnableNoScriptTag = &enable
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
	b.History.Push(u)
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
