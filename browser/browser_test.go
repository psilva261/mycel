package browser

import (
	"github.com/mjl-/duit"
	"golang.org/x/net/html"
	"net/http"
	"net/url"
	"opossum/logger"
	"opossum/nodes"
	"opossum/style"
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

func TestLinkedUrl(t *testing.T) {
	items := []item{
		item{
			orig:   "https://news.ycombinator.com/item?id=24777268",
			href:   "news",
			expect: "https://news.ycombinator.com/news",
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
	body := grepBody(doc)
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

