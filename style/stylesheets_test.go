package style

import (
	"github.com/chris-ramon/douceur/css"
	"golang.org/x/net/html"
	"opossum/logger"
	"strings"
	"testing"
)

func init() {
	quiet := true
	logger.Quiet = &quiet
	logger.Init()
	log = &logger.Logger{Debug: true}
}

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

	tri := tr.colorHex("color")
	hri := hr.colorHex("color")
	if tri != hri {
		t.Fatalf("tri=%x hri=%x", tri, hri)
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
			if m[body][0].Declarations[0].Value != "lightblue" {
				t.Fail()
			}
			t.Logf("%v", m[body][0].Name)
		} else {
			if _, ok := m[body]; ok {
				t.Fail()
			}
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

	res := parent.ApplyChildStyle(child)
	if v := res.Declarations["height"].Value; v != "80px" {
		t.Fatalf(v)
	}
}

/*func TestApplyChildStyleMultiply(t *testing.T) {
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
	child.Declarations["height"] = css.Declaration{
		Property: "height",
		Value:    "50%",
	}

	res := parent.ApplyChildStyle(child)
	if v := res.Declarations["height"].Value; v != "40px" {
		t.Fatalf(v)
	}
}*/
