package img

import (
	"github.com/psilva261/opossum/logger"
	"testing"
)

func init() {
	f := false
	logger.Quiet = &f
	logger.Init()
	log = &logger.Logger{Debug: true}
	SetLogger(&logger.Logger{})
}

func TestParseDataUri(t *testing.T) {
	srcs := []string{"data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP//yH5BAEAAAAALAAAAAABAAEAAAIBRAA7",
		"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNgYAAAAAMAASsJTYQAAAAASUVORK5CYII=",
		// svg examples from github.com/tigt/mini-svg-data-uri (MIT License, (c) 2018 Taylor Hunt)
		"data:image/svg+xml,%3csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 50 50' %3e%3cpath d='M22 38V51L32 32l19-19v12C44 26 43 10 38 0 52 15 49 39 22 38z'/%3e %3c/svg%3e",
		"data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA1MCA1MCI+PHBhdGggZD0iTTIyIDM4VjUxTDMyIDMybDE5LTE5djEyQzQ0IDI2IDQzIDEwIDM4IDAgNTIgMTUgNDkgMzkgMjIgMzh6Ii8+PC9zdmc+",
		`data:image/svg+xml;charset=utf-8,%3Csvg xmlns=http://www.w3.org/2000/svg%3E%3C/svg%3E`,
	}

	for _, src := range srcs {
		data, _, err := parseDataUri(src)
		if err != nil {
			t.Fatalf(err.Error())
		}
		t.Logf("%v", data)
	}
}

func TestSvg(t *testing.T) {
       xml := `
               <svg fill="currentColor" height="24" viewBox="0 0 24 24" width="24">
               </svg>
       `

       _, err := Svg(xml, 0, 0)
       if err != nil {
               t.Fatalf(err.Error())
       }
}

func TestSvgUnquoted(t *testing.T) {
       xml := `
               <svg fill=currentColor height=24 viewBox=0 0 24 24 width=24>
               	<g fill=green></g>
               	<g fill=yellow/>
               </svg>
       `
       xml=`<svg xmlns=http://www.w3.org/2000/svg viewBox=0 0 37 37 fill=#000000><path class=border fill=blue stroke=green/></svg>`

       _, err := Svg(xml, 0, 0)
       if err != nil {
               t.Fatalf(err.Error())
       }
}

func TestQuoteAttrsInTag(t *testing.T) {
	cases := map[string]string{
		`<svg xmlns=http://www.w3.org/2000/svg viewBox=0 0 37 37 fill=#000000>`: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 37 37" fill="#000000">`,
		`<path class=border fill=yellow stroke=green d=M29.2 21.3 0z/>`: `<path class="border" fill="yellow" stroke="green" d="M29.2 21.3 0z"/>`,
		`</svg>`: `</svg>`,
	}
	for c, exp := range cases {
		q := quoteAttrsInTag(c)
		if q != exp {
			t.Errorf("%+v != %+v", q, exp)
		}
	}
}

