package browser

import (
	"net/url"
	"opossum/logger"
	"testing"
)

func init() {
	SetLogger(&logger.Logger{})
}

type item struct {
	orig   string
	href   string
	expect string
}

func TestParseDataUri(t *testing.T) {
	srcs := []string{"data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP//yH5BAEAAAAALAAAAAABAAEAAAIBRAA7",
		"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNgYAAAAAMAASsJTYQAAAAASUVORK5CYII=",
	}

	for _, src := range srcs {
		data, err := parseDataUri(src)
		if err != nil {
			t.Fatalf(err.Error())
		}
		t.Logf("%v", data)
	}
}

func TestLinkedUrl(t *testing.T) {
	items := []item{
		item{
			orig:   "https://news.ycombinator.com/item?id=24777268",
			href:   "news",
			expect: "https://news.ycombinator.com/news",
		},
	}

	for _, i := range items {
		b := Browser{}
		origin, err := url.Parse(i.orig)
		if err != nil {
			panic(err.Error())
		}
		b.URL = origin
		res, err := b.LinkedUrl(i.href)
		if err != nil {
			panic(err.Error())
		}
		if res.String() != i.expect {
			t.Fatalf("got %v but expected %v", res, i.expect)
		}
		t.Logf("res=%v, i.expect=%v", res, i.expect)
	}
}

func TestNilPanic(t *testing.T) {
	//f, err := os.Open()
}
