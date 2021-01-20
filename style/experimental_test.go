package style

import (
	"9fans.net/go/draw"
	"github.com/chris-ramon/douceur/css"
	"testing"
)

func TestBackgroundImageUrl(t *testing.T) {
	suffix := ""
	for _, quote := range []string{"", "'", `"`} {
		url := "/foo.png"
		decl := css.Declaration{
			Value: "url(" + quote + url + quote + ")" + suffix,
		}
		imgUrl, ok := backgroundImageUrl(decl)
		if !ok {
			t.Fatalf("not ok")
		}
		if imgUrl != url {
			t.Fatalf("expected %+v but got %+v", url, imgUrl)
		}
	}
}

func TestBackgroundColor(t *testing.T) {
	colors := map[string]draw.Color{
		"#000000": draw.Black,
		"#ffffff": draw.White,
	}

	for _, k := range []string{"background", "background-color"} {
		m := Map{
			Declarations: make(map[string]css.Declaration),
		}
		for hex, d := range colors {
			m.Declarations[k] = css.Declaration{
				Property: k,
				Value:    hex,
			}

			if b := m.backgroundColor(); b != d {
				t.Fatalf("%v", b)
			}
		}
	}
}
