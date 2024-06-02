package nodes

import (
	"fmt"
	"github.com/psilva261/mycel"
	"github.com/psilva261/mycel/style"
	"golang.org/x/net/html"
	"strings"
	"testing"
)

func TestPath(t *testing.T) {
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
	fmt.Printf("%v\n", a.Data())
	if p, _ := a.Path(); p != mycel.PathPrefix+"/0/1/0/2" {
		t.Fatalf("%v", p)
	}
}

func TestQuery(t *testing.T) {
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
	res, _ := nt.Query("b")
	if len(res) != 1 || res[0].Data() != "b" {
		t.Errorf("%+v", res)
	}
	res, _ = nt.Query("HTML > :nth-child(2) > :nth-child(1) > :nth-child(2)")
	if len(res) != 1 || res[0].Data() != "i" {
		t.Errorf("%+v", res[0])
	}
}
