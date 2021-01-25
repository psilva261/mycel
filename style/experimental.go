package style

import (
	"9fans.net/go/draw"
	"github.com/chris-ramon/douceur/css"
	"fmt"
	"github.com/mjl-/duit"
	"image"
	"github.com/psilva261/opossum"
	"github.com/psilva261/opossum/img"
	"strings"
)

var colorCache = make(map[draw.Color]*draw.Image)
var fetcher opossum.Fetcher

func SetFetcher(f opossum.Fetcher) {
	fetcher = f
}

var TextNode = Map{
	Declarations: map[string]css.Declaration{
		"display": css.Declaration{
			Property: "display",
			Value:    "inline",
		},
	},
}

func (cs Map) BoxBackground() (i *draw.Image, err error) {
	if ExperimentalUseBoxBackgrounds {
		var bgImg *draw.Image

		bgImg = cs.backgroundImage()

		if bgImg == nil {
			bgColor, ok := cs.backgroundColor()
			if !ok {
				return
			}
			log.Printf("bgColor=%+v", bgColor)
			i, ok = colorCache[bgColor]
			if !ok {
				var err error
				i, err = dui.Display.AllocImage(image.Rect(0, 0, 10, 10), draw.ARGB32, true, bgColor)
				if err != nil {
					return nil, fmt.Errorf("alloc img: %w", err)
				}
				colorCache[bgColor] = i
			}
		} else {
			i = bgImg
		}
	}
	return
}

func (cs Map) backgroundColor() (c draw.Color, ok bool) {
	_, ok = cs.Declarations["background-color"]
	if ok {
		c, ok = cs.colorHex("background-color")
		if !ok {
			return
		}
		return c, true
	}
	_, ok = cs.Declarations["background"]
	if ok {
		c, ok = cs.colorHex("background")
		if !ok {
			return
		}
		return c, true
	}
	return
}

func backgroundImageUrl(decl css.Declaration) (url string, ok bool) {
	if v := decl.Value; strings.Contains(v, "url(") && strings.Contains(v, ")") {
		v = strings.ReplaceAll(v, `"`, "")
		v = strings.ReplaceAll(v, `'`, "")
		from := strings.Index(v, "url(")
		if from < 0 {
			log.Printf("bg img: no url: %v", decl.Value)
			return
		}
		from += len("url(")
		imgUrl := v[from:]
		to := strings.Index(imgUrl, ")")
		if to < 0 {
			log.Printf("bg img: no ): %v", decl.Value)
			return
		}
		imgUrl = imgUrl[:to]
		return imgUrl, true
	} else {
		log.Printf("bg img: missing ( or ) '%+v'", decl.Value)
		return
	}
}

func (cs Map) backgroundImage() (i *draw.Image) {
	decl, ok := cs.Declarations["background-image"]
	if !ok {
		decl, ok = cs.Declarations["background"]
	}

	if ok {
		imgUrl, ok := backgroundImageUrl(decl)
		if !ok {
			log.Printf("bg img not ok")
			return
		}

		w := cs.Width()
		h := cs.Height()

		r, err := img.Load(fetcher, imgUrl, w, h)
		if err != nil {
			log.Errorf("bg img load %v: %v", imgUrl, err)
			return nil
		}

		i, err = duit.ReadImage(dui.Display, r)
		if err != nil {
			log.Errorf("bg read image %v: %v", imgUrl, err)
			return
		}
		return i
	}
	return
}