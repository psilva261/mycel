package img

import (
	"bytes"
	"github.com/nfnt/resize"
	"encoding/base64"
	"fmt"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"image"
	"image/png"
	"io"
	"github.com/psilva261/opossum"
	"github.com/psilva261/opossum/logger"
	"strings"
	"net/url"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

const SrcZero = "//:0"

var log *logger.Logger

func SetLogger(l *logger.Logger) {
	log = l
}

func parseDataUri(addr string) (data []byte, ct opossum.ContentType, err error) {
	addr = strings.TrimPrefix(addr, "data:")
	if strings.Contains(addr, "charset=UTF-8") {
		return nil, ct, fmt.Errorf("cannot handle charset")
	}
	parts := strings.Split(addr, ",")
	
	var ctStr string

	if strings.Contains(parts[0], ";") {
		header := strings.Split(parts[0], ";")
		ctStr = header[0]
	} else {
		ctStr = parts[0]
	}
	if ct, err = opossum.NewContentType(ctStr, nil); err != nil {
		return nil, ct, fmt.Errorf("content type: %v: %w", ctStr, err)
	}
	
	if strings.Contains(addr, "base64") {
		e := base64.RawStdEncoding
		if strings.HasSuffix(addr, "=") {
			e = base64.StdEncoding
		}
		if data, err = e.DecodeString(parts[1]); err != nil {
			return nil, ct, fmt.Errorf("base64 decode %v src: %w", addr, err)
		}
	} else {
		out, err := url.QueryUnescape(parts[1])
		if err != nil {
			return nil, ct, fmt.Errorf("url decode: %w", err)
		}
		data = []byte(out)
	}
	return
}

func quoteAttrsInTag(s string) string {
	eqs := make([]int, 0, 5)
	offset := 0

	for {
		i := strings.Index(s[offset:], "=")
		if i >= 0 {
			eqs = append(eqs, i+offset)
			offset += i + 1
		} else {
			break
		}
	}

	keyStarts := make([]int, len(eqs))
	for i, eq := range eqs {
		j := strings.LastIndex(s[:eq], " ")
		keyStarts[i] = j
	}
	
	valueEnds := make([]int, len(keyStarts))
	for i, _ := range keyStarts {
		if i+1 < len(keyStarts) {
			valueEnds[i] = keyStarts[i+1]
		} else {
			off := eqs[i]
			jj := strings.Index(s[off:], ">")
			valueEnds[i] = jj+off
			if s[valueEnds[i]-1:valueEnds[i]] == "/" {
				valueEnds[i]--
			}
		}
	}

	for i := len(eqs) - 1; i >= 0; i-- {
		s = s[:valueEnds[i]] + `"` + s[valueEnds[i]:]
		s = s[:eqs[i]+1] + `"` + s[eqs[i]+1:]
	}

	return s
}

func quoteAttrs(s string) string {
	if strings.Contains(s, `"`) {
		return s
	}

	tagStarts := make([]int, 0, 5)
	tagEnds := make([]int, 0, 5)

	offset := 0
	for {
		i := strings.Index(s[offset:], "<")
		if i >= 0 {
			tagStarts = append(tagStarts, i+offset)
			offset += i + 1
		} else {
			break
		}
	}

	offset = 0
	for {
		i := strings.Index(s[offset:], ">")
		if i >= 0 {
			tagEnds = append(tagEnds, i+offset)
			offset += i + 1
		} else {
			break
		}
	}
	
	if len(tagStarts) != len(tagEnds) {
		log.Errorf("quoteAttrs: len(tagStarts) != len(tagEnds)")
		return s
	}

	for i := len(tagStarts) - 1; i >= 0; i-- {
		from := tagStarts[i]
		to := tagEnds[i] + 1
		q := quoteAttrsInTag(s[from:to])
		s = s[:tagStarts[i]] + q + s[tagEnds[i]+1:]
	}

	return s
}

// Svg returns the svg+xml encoded as jpg with the sizing defined in
// viewbox unless w and h != 0
func Svg(data string, w, h int) (bs []byte, err error) {
	data = strings.ReplaceAll(data, "currentColor", "black")
	data = strings.ReplaceAll(data, "inherit", "black")
	data = quoteAttrs(data)

	r := bytes.NewReader([]byte(data))
	icon, err := oksvg.ReadIconStream(r)
	if err != nil {
		return nil, fmt.Errorf("read icon stream: %w", err)
	}

	if w == 0 || h == 0 {
		w = int(icon.ViewBox.W)
		h = int(icon.ViewBox.H)
	}

	icon.SetTarget(0, 0, float64(w), float64(h))
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	icon.Draw(rasterx.NewDasher(w, h, rasterx.NewScannerGV(w, h, rgba, rgba.Bounds())), 1)

	buf := bytes.NewBufferString("")
	if err = png.Encode(buf, rgba); err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}

	return buf.Bytes(), nil
}

// Load and resize to w and h if != 0
func Load(f opossum.Fetcher, src string, w, h int) (r io.Reader, err error) {
	var imgUrl *url.URL
	var data []byte
	var contentType opossum.ContentType

	if strings.HasPrefix(src, "data:") {
		if data, contentType, err = parseDataUri(src); err != nil {
			return nil, fmt.Errorf("parse data uri %v: %w", src, err)
		}
	} else {
		if imgUrl, err = f.LinkedUrl(src); err != nil {
			return nil, err
		}
		if data, contentType, err = f.Get(imgUrl); err != nil {
			return nil, fmt.Errorf("get %v: %w", imgUrl, err)
		}
	}

	if contentType.IsSvg() {
		data, err = Svg(string(data), w, h)
		if err != nil {
			return nil, fmt.Errorf("svg: %v", err)
		}
	} else if w != 0 || h != 0 {
		image, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("decode %v: %w", imgUrl, err)
		}

		newImage := resize.Resize(uint(w), uint(h), image, resize.Lanczos3)

		// Encode uses a Writer, use a Buffer if you need the raw []byte
		buf := bytes.NewBufferString("")
		if err = png.Encode(buf, newImage); err != nil {
			return nil, fmt.Errorf("encode: %w", err)
		}
		data = buf.Bytes()
	}

	return bytes.NewReader(data), nil
}
