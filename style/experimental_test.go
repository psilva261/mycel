package style

import (
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
