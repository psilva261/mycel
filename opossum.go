package opossum

import (
	"bytes"
	"golang.org/x/text/encoding/charmap"
	"io/ioutil"
	"mime"
	"opossum/logger"
	"net/url"
	"strings"
)

var log *logger.Logger

func SetLogger(l *logger.Logger) {
	log = l
}

type Fetcher interface {
	// LinkedUrl relative to current page
	LinkedUrl(string) (*url.URL, error)

	Get(*url.URL) ([]byte, ContentType, error)
}

type ContentType struct {
	MediaType string
	Params map[string]string
}

func NewContentType(s string) (c ContentType, err error) {
	c.MediaType, c.Params, err = mime.ParseMediaType(s)
	return
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

func (c ContentType) Utf8(buf []byte) []byte {
	charset, ok := c.Params["charset"]
	if !ok || charset == "utf8" || charset == "utf-8" {
		return buf
	}
	if strings.ToLower(charset) == "iso-8859-1" {
		r := bytes.NewReader(buf)
	    cr := charmap.ISO8859_1.NewDecoder().Reader(r)

		updated, err := ioutil.ReadAll(cr)
		if err == nil {
			buf = updated
		} else {
			log.Errorf("utf8: unable to decode to %v: %v", charset, err)
		}
	}
	return buf
}