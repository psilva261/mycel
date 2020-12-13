package img

import (
	//"9fans.net/go/draw"
	"bytes"
	"github.com/nfnt/resize"
	"encoding/base64"
	"fmt"
	//"github.com/mjl-/duit"
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

func parseDataUri(addr string) (data []byte, err error) {
	if strings.Contains(addr, "charset=UTF-8") {
		return nil, fmt.Errorf("cannot handle charset")
	}
	parts := strings.Split(addr, ",")
	e := base64.RawStdEncoding
	if strings.HasSuffix(addr, "=") {
		e = base64.StdEncoding
	}
	if data, err = e.DecodeString(parts[1]); err != nil {
		return nil, fmt.Errorf("decode %v src: %w", addr, err)
	}
	return
}

// Load and resize to w and h if != 0
func Load(f opossum.Fetcher, src string, w, h int) (r io.Reader, err error) {
	var imgUrl *url.URL
	var data []byte
	if strings.HasPrefix(src, "data:") {
		if data, err = parseDataUri(src); err != nil {
			return nil, fmt.Errorf("parse data uri %v: %w", src, err)
		}
	} else {
		if imgUrl, err = f.LinkedUrl(src); err != nil {
			return nil, err
		}
		if data, _, err = f.Get(imgUrl); err != nil {
			return nil, fmt.Errorf("get %v: %w", imgUrl, err)
		}
	}

	if w != 0 || h != 0 {
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
