package style

import (
	"9fans.net/go/draw"
	"fmt"
	"github.com/psilva261/opossum"
	"github.com/psilva261/opossum/img"
	"github.com/psilva261/opossum/logger"
	"image"
	"strings"
)

var colorCache = make(map[draw.Color]*draw.Image)
var fetcher opossum.Fetcher

func SetFetcher(f opossum.Fetcher) {
	fetcher = f
}

var TextNode = Map{
	Declarations: map[string]Declaration{
		"display": Declaration{
			Prop: "display",
			Val:  "inline",
		},
	},
}

func (cs Map) BoxBackground() (i *draw.Image, err error) {
	var bgImg *draw.Image

	if bgImg = cs.backgroundImage(); bgImg != nil {
		return bgImg, nil
	}

	if bgImg = cs.BackgroundGradient(); bgImg != nil {
		return bgImg, nil
	}

	if bgImg == nil {
		bgColor, ok := cs.backgroundColor()
		if !ok {
			return
		}
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
	return
}

func (cs Map) backgroundColor() (c draw.Color, ok bool) {
	d, ok := cs.Declarations["background-color"]
	if ok {
		c, ok = colorHex(d.Val)
		if !ok {
			return
		}
		return c, true
	}
	d, ok = cs.Declarations["background"]
	if ok {
		c, ok = colorHex(d.Val)
		if !ok {
			return
		}
		return c, true
	}
	return
}

// BackgroundGradient is a stub implemention right now (TODO)
func (cs Map) BackgroundGradient() (img *draw.Image) {
	var err error
	c, ok := cs.backgroundGradient()
	if !ok {
		return
	}
	img, err = dui.Display.AllocImage(image.Rect(0, 0, 10, 10), draw.ARGB32, true, c)
	if err != nil {
		log.Errorf("alloc img: %v", err)
		img = nil
	}
	return
}

func (cs Map) backgroundGradient() (c draw.Color, ok bool) {
	d, ok := cs.Declarations["background"]
	if !ok {
		return
	}
	v := strings.TrimSpace(d.Val)
	if strings.HasPrefix(v, "linear-gradient(") {
		v = strings.TrimPrefix(v, "linear-gradient(")
	} else {
		return c, false
	}
	v = strings.TrimSuffix(v, ")")

	colors := make([]draw.Color, 0, 2)

	for i := 0; i < len(v); {
		m := strings.Index(v[i:], ",")
		op := strings.Index(v[i:], "(")
		cl := strings.Index(v[i:], ")")
		if m < 0 {
			break
		}
		var arg string
		if cl > 0 && op < m && m < cl {
			arg = v[i : i+cl+1]
			i += cl + 1
		} else {
			arg = v[i : i+m]
			i += m + 1
		}

		arg = strings.ReplaceAll(arg, " ", "")
		c, ok := colorHex(arg)
		if ok {
			colors = append(colors, c)
		}
	}
	if len(colors) >= 2 {
		from := colors[0]
		to := colors[1]
		c := linearGradient(from, to, 0.5, 0, 1)
		return c, true
	}
	return
}

func linearGradient(from, to draw.Color, x, y, xmax float64) (c draw.Color) {
	fr, fg, fb, fa := from.RGBA()
	tr, tg, tb, ta := to.RGBA()
	d := x / xmax
	r := uint32(float64(fr) + d*float64(tr-fr))
	g := uint32(float64(fg) + d*float64(tg-fg))
	b := uint32(float64(fb) + d*float64(tb-fb))
	a := uint32(float64(fa) + d*float64(ta-fa))
	cc := (r / 256) << 24
	cc = cc | ((g / 256) << 16)
	cc = cc | ((b / 256) << 8)
	cc = cc | (a / 256)
	return draw.Color(cc)
}

func backgroundImageUrl(decl Declaration) (url string, ok bool) {
	if v := decl.Val; strings.Contains(v, "url(") && strings.Contains(v, ")") {
		v = strings.ReplaceAll(v, `"`, "")
		v = strings.ReplaceAll(v, `'`, "")
		from := strings.Index(v, "url(")
		if from < 0 {
			log.Printf("bg img: no url: %v", v)
			return
		}
		from += len("url(")
		imgUrl := v[from:]
		to := strings.Index(imgUrl, ")")
		if to < 0 {
			log.Printf("bg img: no ): %v", v)
			return
		}
		imgUrl = imgUrl[:to]
		return imgUrl, true
	} else {
		log.Printf("bg img: missing ( or ) '%+v'", v)
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

		var err error
		i, err = img.Load(dui, fetcher, imgUrl, 0, w, h, true)
		if err != nil {
			log.Errorf("bg img load %v: %v", imgUrl, err)
			return
		}
	}
	return
}
