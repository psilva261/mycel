package browser

import (
	"context"
	"fmt"
	"github.com/psilva261/mycel"
	"github.com/psilva261/mycel/browser/fs"
	"github.com/psilva261/mycel/logger"
	"github.com/psilva261/mycel/js"
	"github.com/psilva261/mycel/nodes"
	"github.com/psilva261/mycel/style"
	"golang.org/x/net/html"
	"io/ioutil"
	"net/url"
	"strings"
	"testing"
)

func init() {
	log.Debug = true
	style.Init(nil)
}

type TestFetcher struct {}

func (tf *TestFetcher) Ctx() context.Context {
	return context.Background()
}

func (tf *TestFetcher) Origin() (u *url.URL) {
	u, _ = url.Parse("https://example.com")
	return
}

func (tf *TestFetcher) LinkedUrl(string) (*url.URL, error) {
	return nil, fmt.Errorf("not implemented")
}

func (tf *TestFetcher) Get(*url.URL) ([]byte, mycel.ContentType, error) {
	return nil, mycel.ContentType{}, fmt.Errorf("not implemented")
}

func TestProcessJS2SkipFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	f := &TestFetcher{}
	h := `
	<html>
	<body>
	<h1 id="title">Hello</h1>
	</body>
	</html>
	`
	buf := strings.NewReader(h)
	doc, err := html.Parse(buf)
	if err != nil {
		t.Fatalf(err.Error())
	}
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
	fs := fs.New()
	go fs.Srv9p()
	fs.SetDOM(nt)
	fs.Update("", h, nil, scripts)
	js.Start(f)
	s, h, _, err := processJS2(f)
	if err != nil {
		t.Errorf(err.Error())
	}
	t.Logf("h = %+v", h)
	if !strings.Contains(h, `<body style="display: none; ">`) {
		t.Fail()
	}
	s.Stop()
}
