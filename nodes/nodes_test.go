package nodes

import (
	"golang.org/x/net/html"
	"github.com/psilva261/opossum/style"
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
	if err != nil { t.Fatalf(err.Error()) }
	nt := NewNodeTree(doc, style.Map{}, make(map[*html.Node]style.Map), nil)
	p := nt.Children[0].Children[1].Children[1]
	a := p.Children[5]
	if q := a.QueryRef(); q != "p:nth-child(1) > a:nth-child(3)" { t.Fatalf("%v", q) }
}

func TestSetText(t *testing.T) {
	buf := strings.NewReader("<textarea>initial</textarea>")
	doc, err := html.Parse(buf)
	if err != nil { t.Fatalf(err.Error()) }
	n := NewNodeTree(doc, style.Map{}, make(map[*html.Node]style.Map), nil)
	if s := ContentFrom(*n); s != "initial" {
		t.Fatalf(s)
	}
	n.SetText("123")
	if s := ContentFrom(*n); s != "123" {
		t.Fatalf(s)
	}
}

func TestNewNodeTree(t *testing.T) {
	buf := strings.NewReader(`
	<html>
		<body style="width: 900px; height: 700px; font-size: 12px">
			<p>
				<b style="height: 100px;">bold stuff</b>
			</p>
		</body>
	</html>`)
	doc, err := html.Parse(buf)
	if err != nil { t.Fatalf(err.Error()) }
	n := NewNodeTree(doc, style.Map{}, make(map[*html.Node]style.Map), nil)
	body := n.Find("body")
	bodyW := body.Map.Css("width")
	bodyH := body.Map.Css("height")
	bodyF := body.Map.Css("font-size")
	if bodyW != "900px" || bodyH != "700px"/* || bodyF != "12px"*/  {
		t.Fatalf("<%v> w=%v h=%v f=%v", body.Data(), bodyW, bodyH, bodyF)
	}
	b := n.Find("b")
	bW := b.Map.Css("width")
	bH := b.Map.Css("height")
	bF := b.Map.Css("font-size")
	if bW != "" || bH != "100px"/* || bF != "12px"*/  {
		t.Fatalf("<%v> w=%v h=%v f=%v", b.Data(), bW, bH, bF)
	}
}
