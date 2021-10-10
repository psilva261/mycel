package js

import (
	"github.com/psilva261/opossum/browser/fs"
	"github.com/psilva261/opossum/logger"
	"github.com/psilva261/opossum/nodes"
	"github.com/psilva261/opossum/style"
	"golang.org/x/net/html"
	"io/ioutil"
	"strings"
	"testing"
)

const simpleHTML = `
<html>
<body>
<h1 id="title">Hello</h1>
</body>
</html>
`

func init() {
	log.Debug = true
	go fs.Srv9p()
}

func TestJQueryHide(t *testing.T) {
	buf, err := ioutil.ReadFile("jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	script := `
	$(document).ready(function() {
		$('h1').hide();
	});
	`

	r := strings.NewReader(simpleHTML)
	doc, err := html.Parse(r)
	if err != nil {
		t.Fatalf(err.Error())
	}
	nt := nodes.NewNodeTree(doc, style.Map{}, make(map[*html.Node]style.Map), nil)
	fs.SetDOM(nt)
	fs.Update(simpleHTML, nil, []string{string(buf), script})

	NewJS(simpleHTML, nil, nil)
	resHtm, changed, err := Start(string(buf), script)
	if !changed {
		t.Fatalf("changed=%v", changed)
	}
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("resHtm=%v", resHtm)

	r = strings.NewReader(resHtm)
	doc, err = html.Parse(r)
	if err != nil {
		t.Fatalf(err.Error())
	}
	nt = nodes.NewNodeTree(doc, style.Map{}, make(map[*html.Node]style.Map), nil)
	if v := nt.Find("h1").Css("display"); v != "none" {
		t.Fatalf("%v", v)
	}
	Stop()
}
