package style

import (
	"github.com/chris-ramon/douceur/css"
	"golang.org/x/net/html"
	"github.com/mjl-/duit"
	"strings"
	"testing"
)

func d(c string) Map {
	m := Map{
		Declarations: make(map[string]css.Declaration),
	}
	m.Declarations["color"] = css.Declaration{
		Property: "color",
		Value:    c,
	}
	return m
}

func TestColorHex(t *testing.T) {
	tr := d("red")
	hr := d("#ff0000")

	tri, ok := tr.colorHex("color")
	if !ok {
		t.Fail()
	}

	hri, ok := hr.colorHex("color")
	if !ok {
		t.Fail()
	}

	if tri != hri {
		t.Fatalf("tri=%x hri=%x", tri, hri)
	}
}

func TestColorHex3(t *testing.T) {
	m := d("#fff")

	c, ok := m.colorHex("color")
	if !ok {
		t.Fail()
	}

	if uint32(c) != 0xffffffff {
		t.Errorf("c=%x", c)
	}
}

func TestFetchNodeRules(t *testing.T) {
	data := `<body>
      		<h2 id="foo">a header</h2>
     		<h2 id="bar">another header</h2>
   		<p>Some text <b>in bold</b></p>
    	</body>`
	doc, err := html.Parse(strings.NewReader(data))
	if err != nil {
		t.Fail()
	}
	css := AddOnCSS + `
b {
	width: 100px!important;
}

@media only screen and (max-width: 600px) {
  body {
    background-color: lightblue;
  }
}
	`
	for _, w := range []int{400, 800} {
		t.Logf("w=%v", w)
		m, err := FetchNodeRules(doc, css, w)
		if err != nil {
			t.Fail()
		}
		t.Logf("m=%+v", m)

		var b *html.Node
		var body *html.Node

		var f func(n *html.Node)
		f = func(n *html.Node) {
			switch n.Data {
			case "b":
				b = n
			case "body":
				body = n
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}

		f(doc)

		importantFound := false
		for _, r := range m[b] {
			if r.Declarations[0].Important {
				importantFound = true
			}
		}
		if !importantFound {
			t.Fail()
		}

		if w == 400 {
			_ =m[body][0]
			if v := m[body][0].Declarations[0].Value; v != "lightblue" {
				t.Fatalf("%v", v)
			}
			t.Logf("%v", m[body][0].Name)
		} else {
			if _, ok := m[body]; ok {
				t.Fatalf("body ok")
			}
		}
	}
}

func TestFetchNodeRules2(t *testing.T) {
	data := `<h2 id="outer">
				<h2 id="sp1">[</h2>
				<h2 id="sp2">]</h2>
			</h2>`
	doc, err := html.Parse(strings.NewReader(data))
	if err != nil {
		t.Fail()
	}
	m, err := FetchNodeRules(doc, AddOnCSS, 1024)
	if err != nil {
		t.Fail()
	}
	t.Logf("m=%+v", m)

	var outer *html.Node
	var sp1 *html.Node
	var sp2 *html.Node

	var f func(n *html.Node)
	f = func(n *html.Node) {
		if len(n.Attr) == 1 {
			switch n.Attr[0].Val {
			case "outer":
				outer = n
			case "sp1":
				sp1 = n
			case "sp2":
				sp2 = n
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	/*t.Logf("outer=%+v", outer)
	t.Logf("sp1=%+v", sp1)
	t.Logf("sp2=%+v", sp2)*/

	for _, n := range []*html.Node{outer, sp1, sp2} {
		_, ok := m[n]
		if !ok {
			t.Errorf("not found: %+v", n)
		} else {
			t.Logf("success: %+v", n)
		}
	}
}

func TestFetchNodeMap(t *testing.T) {
	data := `<p>
      		<h2 id="foo">a header</h2>
     		<h2 id="bar">another header</h2>
   		<p>Some text <b>in bold</b></p>
    	</p>`
	doc, err := html.Parse(strings.NewReader(data))
	if err != nil {
		t.Fail()
	}
	m, err := FetchNodeMap(doc, AddOnCSS, 1024)
	if err != nil {
		t.Fail()
	}
	t.Logf("m=%+v", m)
}

func TestNewMapStyle(t *testing.T) {
	htms := []string{
		`<h2 style="color: green;">a header</h2>`,
		`<h2 style="color: green">a header</h2>`,
	}
	for _, htm := range htms {
		doc, err := html.Parse(strings.NewReader(htm))
		if err != nil {
			t.Fail()
		}

		h2 := grep(doc, "h2")
		m := NewMap(h2)

		if m.Declarations["color"].Value != "green" {
			t.Errorf("%+v", m)
		}
	}
}

func grep(nn *html.Node, tag string) *html.Node {
	var f func(n *html.Node) *html.Node
	f = func(n *html.Node) *html.Node {
		if n.Data == tag {
			return n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if m := f(c); m != nil {
				return m
			}
		}
		return nil
	}
	return f(nn)
}

func TestSmaller(t *testing.T) {
	d := css.Declaration{
		Important: false,
	}
	dd := css.Declaration{
		Important: true,
	}
	if !smaller(d, dd) {
		t.Fail()
	}
}

func TestApplyChildStyleInherit(t *testing.T) {
	parent := Map{
		Declarations: make(map[string]css.Declaration),
	}
	parent.Declarations["height"] = css.Declaration{
		Property: "height",
		Value:    "80px",
	}
	child := Map{
		Declarations: make(map[string]css.Declaration),
	}

	res := parent.ApplyChildStyle(child, true)
	if v := res.Declarations["height"].Value; v != "80px" {
		t.Fatalf(v)
	}
}

func TestApplyChildStyleInherit2(t *testing.T) {
	parent := Map{
		Declarations: make(map[string]css.Declaration),
	}
	child := Map{
		Declarations: make(map[string]css.Declaration),
	}
	parent.Declarations["font-size"] = css.Declaration{
		Property: "font-size",
		Value:    "12pt",
	}
	child.Declarations["font-size"] = css.Declaration{
		Property: "font-size",
		Value:    "inherit",
	}

	res := parent.ApplyChildStyle(child, true)
	if v := res.Declarations["font-size"].Value; v != "12pt" {
		t.Fatalf(v)
	}
}

func TestLength(t *testing.T) {
	lpx := map[string]float64{
		"auto": 0.0,
		"inherit": 0.0,
		"17px": 17.0,
		"10em": 110.0,
		"10vw": 128.0,
		"10vh": 108.0,
		"10%": 0,
	}
	for l, px := range lpx {
		f, _, err := length(nil, l)
		if err != nil {
			t.Fatalf("%v: %v", l, err)
		}
		if f != px {
			t.Fatalf("expected %v but got %v", px, f)
		}
	}
}

func TestTlbr(tt *testing.T) {
	cases := map[string]duit.Space{
		"1px 2px 3px 4px": duit.Space{1, 2, 3, 4},
		"1px 2px 3px": duit.Space{1, 2, 3, 2},
		"1px 2px": duit.Space{1, 2, 1, 2},
		"1px": duit.Space{1, 1, 1, 1},
	}
	for v, exp := range cases {
		m := Map{
			Declarations: make(map[string]css.Declaration),
		}
		m.Declarations["margin"] = css.Declaration{
			Property: "margin",
			Value:    v,
		}
		s, err := m.Tlbr("margin")
		if err != nil {
			tt.Errorf("%v", s)
		}
		if s != exp {
			tt.Errorf("%v: %v", s, exp)
		}
	}
}
