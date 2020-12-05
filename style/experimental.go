package style

import (
	"9fans.net/go/draw"
	"bytes"
	"github.com/chris-ramon/douceur/css"
	"fmt"
	"github.com/mjl-/duit"
	"image"
	"opossum"
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
			bgColor := cs.backgroundColor()
			log.Printf("bgColor=%+v", bgColor)
			var ok bool
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

func (cs Map) backgroundColor() draw.Color {
	_, ok := cs.Declarations["background-color"]
	if ok {
		return draw.Color(cs.colorHex("background-color"))
	}
	_, ok = cs.Declarations["background"]
	if ok {
		return draw.Color(cs.colorHex("background"))
	}
	return draw.Color(uint32(draw.White))
}

func (cs Map) backgroundImage() (img *draw.Image) {
	decl, ok := cs.Declarations["background"]
	log.Printf("decl=%+v\n", decl)
	if ok {
		log.Printf("bg img ok")
		if strings.Contains(decl.Value, "url(") && strings.Contains(decl.Value, ")") {
			from := strings.Index(decl.Value, "url(")
			if from < 0 {
				log.Printf("bg img: no url: %v", decl.Value)
				return
			}
			from += len("url('")
			imgUrl := decl.Value[from:]
			to := strings.Index(imgUrl, ")")
			if to < 0 {
				log.Printf("bg img: no ): %v", decl.Value)
				return
			}
			to -= len("'")
			imgUrl = imgUrl[:to]
			uri, err := fetcher.LinkedUrl(imgUrl)
			if err != nil {
				log.Printf("bg img interpet url: %v", err)
				return nil
			}
			buf, contentType, err := fetcher.Get(*uri)
			if err != nil {
				log.Printf("bg img get %v (%v): %v", uri, contentType, err)
				return nil
			}
			r := bytes.NewReader(buf)
			log.Printf("Read %v...", imgUrl)
			img, err = duit.ReadImage(dui.Display, r)
			if err != nil {
				log.Printf("bg read image: %v", err)
				return
			}
			return img
		} else {
			log.Printf("bg img: missing fixes '%+v'", decl.Value)
		}
	} else {
		log.Printf("bg img not ok")
	}
	return
}