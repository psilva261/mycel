package js

import (
	"context"
	"fmt"
	"github.com/psilva261/opossum"
	"github.com/psilva261/opossum/browser/fs"
	"github.com/psilva261/opossum/logger"
	"github.com/psilva261/opossum/nodes"
	"github.com/psilva261/opossum/style"
	"golang.org/x/net/html"
	"io/ioutil"
	"net/url"
	"strings"
	"testing"
	"time"
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
	SetFetcher(&TestFetcher{})
	go fs.Srv9p()
	<-time.After(2*time.Second)
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

func (tf *TestFetcher) Get(*url.URL) ([]byte, opossum.ContentType, error) {
	return nil, opossum.ContentType{}, fmt.Errorf("not implemented")
}

func TestJQueryHide(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
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
	fs.Update("", simpleHTML, nil, []string{string(buf), script})

	resHtm, changed, err := Start(string(buf), script)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if !changed {
		t.Fatalf("changed=%v", changed)
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
