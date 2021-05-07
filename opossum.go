package opossum

import (
	"bytes"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/unicode"
	"io/ioutil"
	"mime"
	"github.com/psilva261/opossum/logger"
	"net/url"
	"strings"
)

var log *logger.Logger

func SetLogger(l *logger.Logger) {
	log = l
}

type Fetcher interface {
	Origin() *url.URL

	// LinkedUrl relative to current page
	LinkedUrl(string) (*url.URL, error)

	Get(*url.URL) ([]byte, ContentType, error)
}

type ContentType struct {
	MediaType string
	Params map[string]string
}

// NewContentType based on mime type string and url including file extension as fallback
func NewContentType(s string, u *url.URL) (c ContentType, err error) {
	if s == "" && u != nil && strings.Contains(u.String(), ".") {
		l := strings.Split(u.String(), ".")
		ext := l[len(l)-1]
		switch ext {
		case "jpg":
			return NewContentType("image/jpeg", u)
		case "png":
			return NewContentType("image/png", u)
		case "gif":
			return NewContentType("image/gif", u)
		default:
			return ContentType{}, nil
		}
	}
	c.MediaType, c.Params, err = mime.ParseMediaType(s)
	return
}

func (c ContentType) IsEmpty() bool {
	return c.MediaType == ""
}

func (c ContentType) IsHTML() bool {
	return c.MediaType == "text/html"
}

func (c ContentType) IsCSS() bool {
	return c.MediaType != "text/html"
}

func (c ContentType) IsJS() bool {
	for _, t := range []string{"application/javascript", "application/ecmascript", "text/javascript", "text/ecmascript"} {
		if t == c.MediaType {
			return true
		}
	}
	return false
}

func (c ContentType) IsPlain() bool {
	return c.MediaType == "text/plain"
}

func (c ContentType) IsDownload() bool {
	return c.MediaType == "application/octet-stream" ||
		c.MediaType == "application/zip"
}

func (c ContentType) IsSvg() bool {
	return c.MediaType == "image/svg+xml"
}

func (c ContentType) Charset() (cs string) {
	cs, ok := c.Params["charset"]
	if !ok {
		return "UTF-8"
	}
	return
}

func (c ContentType) Encoding() (e encoding.Encoding) {
	charset, ok := c.Params["charset"]
	if !ok || charset == "utf8" || charset == "utf-8" {
		return unicode.UTF8
	}
	e, err := htmlindex.Get(charset)
	if err != nil || e == nil {
		log.Errorf("encoding %v: %v", charset, err)
		return unicode.UTF8
	}
	return
}

func (c ContentType) Utf8(buf []byte) string {
	e := c.Encoding()

	if e == unicode.UTF8 {
		return string(buf)
	}

	r := bytes.NewReader(buf)
	cr := e.NewDecoder().Reader(r)

	updated, err := ioutil.ReadAll(cr)
	if err == nil {
		buf = updated
	} else {
		log.Errorf("utf8: unable to decode to %v: %v", e, err)
	}

	return string(buf)
}