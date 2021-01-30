package nodes

import (
	"golang.org/x/net/html"
	"github.com/psilva261/opossum/style"
	"strings"
	"testing"
)

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
	if err != nil { t.Fatalf(err.Error()) }
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