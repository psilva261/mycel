package nodes

import (
	"bytes"
	"encoding/json"
	"github.com/psilva261/opossum/style"
	"golang.org/x/net/html"
	"strings"
	"testing"
)

func TestFilterText(t *testing.T) {
	in := "eben­falls"
	exp := "ebenfalls"
	if out := filterText(in); out != exp {
		t.Fatalf("%+v", out)
	}
}

func TestQueryRef(t *testing.T) {
	buf := strings.NewReader(`
	<html>
		<body>
			<p>
				<b>bold stuff</b>
				<i>italic stuff</i>
				<a>link</a>
			</p>
		</body>
	</html>`)
	doc, err := html.Parse(buf)
	if err != nil {
		t.Fatalf(err.Error())
	}
	nt := NewNodeTree(doc, style.Map{}, make(map[*html.Node]style.Map), nil)
	p := nt.Children[0].Children[1].Children[0]
	a := p.Children[2]
	if q := a.QueryRef(); q != "p:nth-child(1) > a:nth-child(1)" {
		t.Fatalf("%v", q)
	}
}

func TestSetText(t *testing.T) {
	buf := strings.NewReader("<textarea>initial</textarea>")
	doc, err := html.Parse(buf)
	if err != nil {
		t.Fatalf(err.Error())
	}
	n := NewNodeTree(doc, style.Map{}, make(map[*html.Node]style.Map), nil)
	if s := n.ContentString(false); s != "initial" {
		t.Fatalf(s)
	}
	n.SetText("123")
	if s := n.ContentString(false); s != "123" {
		t.Fatalf(s)
	}
}

func TestNewNodeTree(t *testing.T) {
	buf := strings.NewReader(`
	<html>
		<body style="width: 900px; height: 700px; font-size: 12px;">
			<p>
				<b style="height: 100px;">bold stuff</b>
			</p>
		</body>
	</html>`)
	doc, err := html.Parse(buf)
	if err != nil {
		t.Fatalf(err.Error())
	}
	n := NewNodeTree(doc, style.Map{}, make(map[*html.Node]style.Map), nil)
	body := n.Find("body")
	bodyW := body.Map.Css("width")
	bodyH := body.Map.Css("height")
	bodyF := body.Map.Css("font-size")
	if bodyW != "900px" || bodyH != "700px" || bodyF != "12px" {
		t.Fatalf("<%v> w=%v h=%v f=%v", body.Data(), bodyW, bodyH, bodyF)
	}
	b := n.Find("b")
	bW := b.Map.Css("width")
	bH := b.Map.Css("height")
	bF := b.Map.Css("font-size")
	if bW != "" || bH != "100px" /* || bF != "12px"*/ {
		t.Fatalf("<%v> w=%v h=%v f=%v", b.Data(), bW, bH, bF)
	}
	text := b.Children[0]
	textF := text.Map.Css("font-size")
	if textF != "12px" || text.Text != "bold stuff" {
		t.Fatalf("%+v", text)
	}
}

func TestJsonCycles(t *testing.T) {
	buf := strings.NewReader(`
	<html>
		<body style="width: 900px; height: 700px; font-size: 12px;">
			<p>
				<b style="height: 100px;">bold stuff</b>
			</p>
		</body>
	</html>`)
	doc, err := html.Parse(buf)
	if err != nil {
		t.Fatalf(err.Error())
	}
	n := NewNodeTree(doc, style.Map{}, make(map[*html.Node]style.Map), nil)
	body := n.Find("body")
	_ = body

	b := bytes.NewBufferString("")
	enc := json.NewEncoder(b)
	if err := enc.Encode(n); err != nil {
		t.Fatalf("%+v", err)
	}
}

func TestContainingBlock(t *testing.T) {
	tests := map[string]string{
		"body": `
			<html>
				<body>
					<div>
						<a style="position: absolute;">link</a>
					</div>
				</body>
			</html>
		`,
		"div": `
			<html>
				<body>
					<div style="position: relative;">
						<a style="position: absolute;">link</a>
					</div>
				</body>
			</html>
		`,
		"main": `
			<html>
				<body>
					<main style="position: relative;">
						<article>
							<a style="position: absolute;">link</a>
						</article>
					</main>
				</body>
			</html>
		`,
	}
	for cbTag, htm := range tests {
		doc, err := html.Parse(strings.NewReader(htm))
		if err != nil {
			t.Fatalf(err.Error())
		}
		nt := NewNodeTree(doc, style.Map{}, make(map[*html.Node]style.Map), nil)
		cb := nt.Find(cbTag)
		a := nt.Find("a")
		if a.CB() != cb {
			t.Fail()
		}
	}
}

func TestCBItems(t *testing.T) {
	tests := map[string]map[string][]string{
		`
			<html>
				<body>
					<div>
						<a style="position: absolute;">link</a>
					</div>
				</body>
			</html>
		`: {
			"body": {"a", "div"},
			"div":  {},
			"a":    {"link"},
		},
		`
			<html>
				<body>
					<div style="position: relative;">
						<a style="position: absolute;">link</a>
					</div>
				</body>
			</html>
		`: {
			"body": {"div"},
			"div":  {"a"},
			"a":    {"link"},
		},
		`
			<html>
				<body>
					<main style="position: relative;">
						<article>
							<a style="position: absolute;">link</a>
						</article>
					</main>
				</body>
			</html>
		`: {
			"body":    {"main"},
			"main":    {"a", "article"},
			"article": {},
			"a":       {"link"},
		},
	}
	for htm, m := range tests {
		doc, err := html.Parse(strings.NewReader(htm))
		if err != nil {
			t.Fatalf(err.Error())
		}
		nt := NewNodeTree(doc, style.Map{}, make(map[*html.Node]style.Map), nil)
		for from, tos := range m {
			t.Logf("from: %v", from)
			f := nt.Find(from)
			cbis := f.CBItems()
			if len(cbis) != len(tos) {
				t.Errorf("len(cbis)=%+v", cbis)
			} else {
				t.Logf("lengths match")
			}
			for i, cbi := range cbis {
				t.Logf("%+v %v", cbi.Data(), cbi.Type())
				if strings.TrimSpace(cbi.Data()) != tos[i] {
					t.Fail()
				}
			}
		}
	}
}
