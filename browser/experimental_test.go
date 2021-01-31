package browser

import (
	"golang.org/x/net/html"
	//"github.com/mjl-/duit"
	"github.com/psilva261/opossum/domino"
	"github.com/psilva261/opossum/logger"
	"github.com/psilva261/opossum/nodes"
	"github.com/psilva261/opossum/style"
	"io/ioutil"
	"strings"
	"testing"
)

func init() {
	quiet := false
	logger.Quiet = &quiet
	f := false
	domino.DebugDumpJS = &f
	domino.SetLogger(&logger.Logger{})
	logger.Init()
	SetLogger(&logger.Logger{})
	style.Init(nil, &logger.Logger{})
}

func TestAtom(t *testing.T) {
	//var ui duit.UI
	//ui = &Atom{}
}

func TestProcessJS2SkipFailure(t *testing.T) {
	h := `
	<html>
	<body>
	<h1 id="title">Hello</h1>
	</body>
	</html>
	`
	buf := strings.NewReader(h)
	doc, err := html.Parse(buf)
	if err != nil { t.Fatalf(err.Error()) }
	nt := nodes.NewNodeTree(doc, style.Map{}, make(map[*html.Node]style.Map), nil)
	d := domino.NewDomino(h, nt)
	d.Start()
	jq, err := ioutil.ReadFile("../domino/jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	scripts := []string{
		string(jq),
		`throw 'fail';`,
		`throw 'fail';`,
		`$('body').hide()`,
		`throw 'fail';`,
	}
	h, err = processJS2(d, scripts)
	if err != nil { t.Errorf(err.Error()) }
	t.Logf("h = %+v", h)
	if !strings.Contains(h, `<body style="display: none;">`) {
		t.Fail()
	}
	d.Stop()
}
