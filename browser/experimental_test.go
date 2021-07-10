package browser

import (
	"golang.org/x/net/html"
	//"github.com/mjl-/duit"
	"github.com/psilva261/opossum/browser/fs"
	"github.com/psilva261/opossum/js"
	"github.com/psilva261/opossum/logger"
	"github.com/psilva261/opossum/nodes"
	"github.com/psilva261/opossum/style"
	"io/ioutil"
	"strings"
	"testing"
)

func init() {
	js.SetLogger(&logger.Logger{})
	logger.Init()
	SetLogger(&logger.Logger{})
	style.Init(nil, &logger.Logger{})
	fs.SetLogger(log)
	go fs.Srv9p()
}

func TestAtom(t *testing.T) {
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
	jq, err := ioutil.ReadFile("../js/jquery-3.5.1.js")
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
	fs.DOM = nt
	fs.Update(h, nil, scripts)
	js.NewJS(h, nil, nt)
	js.Start()
	h, _, err = processJS2()
	if err != nil { t.Errorf(err.Error()) }
	t.Logf("h = %+v", h)
	if !strings.Contains(h, `<body style="display: none;">`) {
		t.Fail()
	}
	js.Stop()
}
