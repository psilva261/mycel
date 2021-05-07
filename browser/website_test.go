package browser

import (
	"github.com/psilva261/opossum"
	"golang.org/x/net/html"
	"net/url"
	"strings"
	"testing"
)

func TestFormData(t *testing.T) {
	htm := `<form>
		<input name=a value=1>
		<textarea name=b>2</textarea>
	</form>`
	doc, err := html.Parse(
		strings.NewReader(string(htm)),
	)
	if err != nil {
		t.Fatalf(err.Error())
	}
	f := grep(doc, "form")
	data := formData(f, nil)
	if len(data) != 2 {
		t.Fatalf("%+v", f)
	}
}

func TestPercentEncoding(t *testing.T) {
	htm := `<form>
		<input name=a value=ツ>
	</form>`
	doc, err := html.Parse(
		strings.NewReader(string(htm)),
	)
	if err != nil {
		t.Fatalf(err.Error())
	}
	f := grep(doc, "form")
	data := formData(f, nil)
	if len(data) != 1 {
		t.Fatalf("%+v", f)
	}

	uri, err := url.Parse("http://example.com")
	if err != nil {
		t.Fatalf(err.Error())
	}

	q := uri.Query()
	for k, vs := range data {
		q.Set(k, vs[0])
	}

	ct := opossum.ContentType{
		MediaType: "text/html",
		Params: map[string]string{
			"charset": "UTF-8",
		},
	}
	res := escapeValues(ct, q).Encode()
	if res != "a=%E3%83%84"  {
		t.Errorf("%v", res)
	}

	ct.Params["charset"] = "ISO-8859-1"
	res = escapeValues(ct, q).Encode()
	if res != "a=%26%2312484%3B"  {
		t.Errorf("%v", res)
	}
}
