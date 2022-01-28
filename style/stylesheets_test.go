package style

import (
	"github.com/mjl-/duit"
	"github.com/psilva261/opossum/logger"
	"golang.org/x/net/html"
	"strings"
	"testing"
)

func init() {
	log.Debug = true
}

func TestColorHex(t *testing.T) {
	tri, ok := colorHex("red")
	if !ok {
		t.Fail()
	}

	hri, ok := colorHex("#ff0000")
	if !ok {
		t.Fail()
	}

	if tri != hri {
		t.Fatalf("tri=%x hri=%x", tri, hri)
	}

	if _, ok := colorHex("rgb(1,2)"); ok {
		t.Fail()
	}
}

func TestColorHex3(t *testing.T) {
	c, ok := colorHex("#fff")
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
		m, _, err := FetchNodeRules(doc, css, w)
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
			for _, d := range r.Declarations {
				if d.Specificity[0] != 0 || d.Specificity[1] != 0 || d.Specificity[2] != 1 {
					t.Fail()
				}
			}
		}
		if !importantFound {
			t.Fail()
		}

		if w == 400 {
			_ = m[body][0]
			if v := m[body][0].Declarations[0].Val; v != "lightblue" {
				t.Fatalf("%v", v)
			}
			t.Logf("%v", m[body][0])
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
	m, _, err := FetchNodeRules(doc, AddOnCSS, 1024)
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

func TestMergeNodeMaps(t *testing.T) {
	nodeMap := make(map[*html.Node]Map)
	data := `<p>
      		<a class="link" href="http://example.com">Test</a>
    	</p>`
	doc, err := html.Parse(strings.NewReader(data))
	if err != nil {
		t.Fail()
	}
	a := grep(doc, "a")
	m, err := FetchNodeMap(doc, AddOnCSS, 1024)
	if err != nil {
		t.Fail()
	}
	MergeNodeMaps(nodeMap, m)
	if nodeMap[a].Css("color") != "blue" {
		t.Fatalf("%v", nodeMap[a])
	}
	m2, err := FetchNodeMap(doc, `.link { color: red; }`, 1024)
	if err != nil {
		t.Fail()
	}
	MergeNodeMaps(nodeMap, m2)
	if nodeMap[a].Css("color") != "red" {
		t.Fatalf("%v", nodeMap[a])
	}
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

		if m.Declarations["color"].Val != "green" {
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
	d := Declaration{
		Important: false,
	}
	dd := Declaration{
		Important: true,
	}
	if !smaller(d, dd) {
		t.Fail()
	}
}

func TestApplyChildStyleInherit(t *testing.T) {
	parent := Map{
		Declarations: make(map[string]Declaration),
	}
	parent.Declarations["height"] = Declaration{
		Prop: "height",
		Val:  "80px",
	}
	child := Map{
		Declarations: make(map[string]Declaration),
	}

	res := parent.ApplyChildStyle(child, true)
	if v := res.Declarations["height"].Val; v != "80px" {
		t.Fatalf(v)
	}
}

func TestApplyChildStyleInherit2(t *testing.T) {
	parent := Map{
		Declarations: make(map[string]Declaration),
	}
	child := Map{
		Declarations: make(map[string]Declaration),
	}
	parent.Declarations["font-size"] = Declaration{
		Prop: "font-size",
		Val:  "12pt",
	}
	child.Declarations["font-size"] = Declaration{
		Prop: "font-size",
		Val:  "inherit",
	}

	res := parent.ApplyChildStyle(child, true)
	if v := res.Declarations["font-size"].Val; v != "12pt" {
		t.Fatalf(v)
	}
}

func TestApplyChildStyleInherit3(t *testing.T) {
	parent := Map{
		Declarations: make(map[string]Declaration),
	}
	child := Map{
		Declarations: make(map[string]Declaration),
	}
	parent.Declarations["font-size"] = Declaration{
		Prop: "font-size",
		Val:  "12pt",
	}
	child.Declarations["font-size"] = Declaration{
		Prop: "font-size",
		Val:  "13pt",
	}

	res := parent.ApplyChildStyle(child, true)
	if v := res.Declarations["font-size"].Val; v != "13pt" {
		t.Fatalf(v)
	}
}

func TestApplyChildStyleInherit4(t *testing.T) {
	parent := Map{
		Declarations: make(map[string]Declaration),
	}
	child := Map{
		Declarations: make(map[string]Declaration),
	}
	parent.Declarations["font-size"] = Declaration{
		Prop:        "font-size",
		Val:         "12pt",
		Specificity: [3]int{0, 2, 0},
	}
	child.Declarations["font-size"] = Declaration{
		Prop:        "font-size",
		Val:         "13pt",
		Specificity: [3]int{0, 1, 0},
	}

	res := parent.ApplyChildStyle(child, true)
	if v := res.Declarations["font-size"].Val; v != "12pt" {
		t.Fatalf(v)
	}
}

func TestCalc(t *testing.T) {
	tests := map[string]float64{
		"calc(1px+2px)":         3.0,
		"calc(1px + 2px)":       3.0,
		"calc(1em+2px)":         13.0,
		"calc(1em+(2px-1px))":   12.0,
		"calc(1em+(2px-1.5px))": 11.5,
	}
	for x, px := range tests {
		f, _, err := length(nil, x)
		if err != nil {
			t.Fatalf("%v: %v", x, err)
		}
		if f != px {
			t.Fatalf("expected %v but got %v", px, f)
		}
	}
}

func TestCalc2(t *testing.T) {
	fails := []string{
		"calc(a+2px)",
		"calc(if(1)2)",
		"calc(quit)",
		"calc(1;)",
		"calc()",
		"calc(" + strings.Repeat("1", 51) + ")",
	}
	for _, x := range fails {
		_, _, err := length(nil, x)
		if err == nil {
			t.Fatalf("%v: %v", x, err)
		}
	}
}

func TestLength(t *testing.T) {
	lpx := map[string]float64{
		"auto":    0.0,
		"inherit": 0.0,
		"17px":    17.0,
		"10em":    110.0,
		"10ex":    110.0,
		"10vw":    128.0,
		"10vh":    108.0,
		"10%":     0,
		"101.6mm": 400,
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
		"1px 2px 3px":     duit.Space{1, 2, 3, 2},
		"1px 2px":         duit.Space{1, 2, 1, 2},
		"1px":             duit.Space{1, 1, 1, 1},
	}
	for v, exp := range cases {
		m := Map{
			Declarations: make(map[string]Declaration),
		}
		m.Declarations["margin"] = Declaration{
			Prop: "margin",
			Val:  v,
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

func TestCssVars(t *testing.T) {
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
:root {
	--emph: red;
	--h: 10px;
}

b {
	color: var(--emph);
}
	`

	_, rv, err := FetchNodeRules(doc, css, 1280)
	if err != nil {
		t.Fail()
	}

	if len(rv) != 2 || rv["--emph"] != "red" || rv["--h"] != "10px" {
		t.Fail()
	}

	var b *html.Node
	var f func(n *html.Node)
	f = func(n *html.Node) {
		if n.Data == "b" {
			b = n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	nm, err := FetchNodeMap(doc, css, 1280)
	if err != nil {
		t.Fail()
	}
	d := nm[b]
	t.Logf("d=%+v", d)
	if d.Declarations["color"].Val != "red" {
		t.Fatalf("%+v", d.Declarations)
	}
}
