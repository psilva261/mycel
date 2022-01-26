package img

import (
	"bytes"
	"context"
	"github.com/psilva261/opossum"
	"github.com/psilva261/opossum/logger"
	"image"
	"image/png"
	"net/url"
	"testing"
)

func init() {
	log.Debug = true
}

func TestParseDataUri(t *testing.T) {
	srcs := []string{"data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP//yH5BAEAAAAALAAAAAABAAEAAAIBRAA7",
		"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNgYAAAAAMAASsJTYQAAAAASUVORK5CYII=",
		// svg examples from github.com/tigt/mini-svg-data-uri (MIT License, (c) 2018 Taylor Hunt)
		"data:image/svg+xml,%3csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 50 50' %3e%3cpath d='M22 38V51L32 32l19-19v12C44 26 43 10 38 0 52 15 49 39 22 38z'/%3e %3c/svg%3e",
		"data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA1MCA1MCI+PHBhdGggZD0iTTIyIDM4VjUxTDMyIDMybDE5LTE5djEyQzQ0IDI2IDQzIDEwIDM4IDAgNTIgMTUgNDkgMzkgMjIgMzh6Ii8+PC9zdmc+",
		`data:image/svg+xml;charset=utf-8,%3Csvg xmlns=http://www.w3.org/2000/svg%3E%3C/svg%3E`,
		// additional example
		`data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 20 30' width='20' height='30' %3E%3C/svg%3E`,
	}

	for _, src := range srcs {
		data, _, err := parseDataUri(src)
		if err != nil {
			t.Fatalf(err.Error())
		}
		t.Logf("%v", string(data))
	}
}

func TestEmpty(t *testing.T) {
	src := "data:image/svg+xml"
	_, _, err := parseDataUri(src)
	if err == nil {
		t.Fatalf(err.Error())
	}
}

func TestSvg(t *testing.T) {
	xmls := []string{
		`
               <svg fill="currentColor" height="24" viewBox="0 0 24 24" width="24">
               </svg>
       `,
		`
               <svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 20 30' width='20' height='30'>
               </svg>
       `,
	}

	for _, xml := range xmls {
		_, err := svg(xml, 0, 0)
		if err != nil {
			t.Fatalf(err.Error())
		}
	}
}

func TestSvgUnquoted(t *testing.T) {
	xml := `
               <svg fill=currentColor height=24 viewBox=0 0 24 24 width=24>
               	<g fill=green></g>
               	<g fill=yellow/>
               </svg>
       `
	xml = `<svg xmlns=http://www.w3.org/2000/svg viewBox=0 0 37 37 fill=#000000><path class=border fill=blue stroke=green/></svg>`

	_, err := svg(xml, 0, 0)
	if err != nil {
		t.Fatalf(err.Error())
	}
}

func TestQuoteAttrsInTag(t *testing.T) {
	cases := map[string]string{
		`<svg xmlns=http://www.w3.org/2000/svg viewBox=0 0 37 37 fill=#000000>`: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 37 37" fill="#000000">`,
		`<path class=border fill=yellow stroke=green d=M29.2 21.3 0z/>`:         `<path class="border" fill="yellow" stroke="green" d="M29.2 21.3 0z"/>`,
		`</svg>`: `</svg>`,
	}
	for c, exp := range cases {
		q := quoteAttrsInTag(c)
		if q != exp {
			t.Errorf("%+v != %+v", q, exp)
		}
	}
}

type MockBrowser struct {
	data []byte
}

func (b *MockBrowser) Ctx() context.Context {
	return context.Background()
}

func (b *MockBrowser) Origin() *url.URL { return nil }

func (b *MockBrowser) LinkedUrl(string) (*url.URL, error) { return nil, nil }

func (b *MockBrowser) Get(*url.URL) ([]byte, opossum.ContentType, error) {
	return b.data, opossum.ContentType{}, nil
}

func TestLoad(t *testing.T) {
	rows := [][]int{
		{1700, 0, 0, 1600, 900},
		{160, 0, 0, 160, 90},
		{0, 0, 45, 80, 45},
		{0, 0, 0, 1600, 900},
		{0, 800, 800, 800, 450},
	}
	for _, r := range rows {
		t.Logf("test case %+v", r)
		mw, w, h, xNew, yNew := r[0], r[1], r[2], r[3], r[4]
		dst := image.NewRGBA(image.Rect(0, 0, 1600, 900))
		buf := bytes.NewBufferString("")
		if err := png.Encode(buf, dst); err != nil {
			t.Fail()
		}
		b := &MockBrowser{buf.Bytes()}
		img, err := load(b, "", mw, w, h)
		if err != nil {
			t.Errorf("load: %v", err)
		}
		dx := img.Bounds().Max.X
		dy := img.Bounds().Max.Y
		if dx != xNew || dy != yNew {
			t.Errorf("unexpected size %v x %v", dx, dy)
		}
	}
}

func TestNewSizes(t *testing.T) {
	x0 := 400
	y0 := 300

	x1, y1, _ := newSizes(x0, y0, 100, 0)
	if x1 != 100 || y1 != 75 {
		t.Fail()
	}

	x1, y1, _ = newSizes(x0, y0, 0, 100)
	if x1 != 133 || y1 != 100 {
		t.Fail()
	}

	// Enforce aspect ratio based on width
	x1, y1, _ = newSizes(x0, y0, 800, 800)
	if x1 != 800 || y1 != 600 {
		t.Fail()
	}
}
