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
	if q := a.QueryRef(); q != "p a" { t.Fatalf("%v", q) }
}