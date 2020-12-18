package img

import (
	"bytes"
	"github.com/nfnt/resize"
	"encoding/base64"
	"fmt"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"image"
	"image/jpeg"
	"io"
	"opossum"
	"opossum/logger"
	"strings"
	"net/url"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

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
		ctStr = header[1]
	} else {
		ctStr = parts[0]
	}
	if ct, err = opossum.NewContentType(ctStr, nil); err != nil {
		return nil, ct, err
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
		r := bytes.NewReader(data)
		icon, _ := oksvg.ReadIconStream(r)
		if w == 0 || h == 0 {
			w = 20
			h = 20
		}
		icon.SetTarget(0, 0, float64(w), float64(h))
		rgba := image.NewRGBA(image.Rect(0, 0, w, h))
		icon.Draw(rasterx.NewDasher(w, h, rasterx.NewScannerGV(w, h, rgba, rgba.Bounds())), 1)
		buf := bytes.NewBufferString("")
		if err = jpeg.Encode(buf, rgba, nil); err != nil {
			return nil, fmt.Errorf("encode: %w", err)
		}
		data = buf.Bytes()
	} else if w != 0 || h != 0 {
		image, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("decode %v: %w", imgUrl, err)
		}
		// check err

		newImage := resize.Resize(uint(w), uint(h), image, resize.Lanczos3)

		// Encode uses a Writer, use a Buffer if you need the raw []byte
		buf := bytes.NewBufferString("")
		if err = jpeg.Encode(buf, newImage, nil); err != nil {
			return nil, fmt.Errorf("encode: %w", err)
		}
		data = buf.Bytes()
	}
	return bytes.NewReader(data), nil
}
