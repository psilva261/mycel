// +build plan9

package style

import (
	"9fans.net/go/draw"
	"fmt"
	"github.com/psilva261/opossum/logger"
	"io/fs"
	"regexp"
	"strings"
	"os"
)

var (
	fonts map[int]*draw.Font
	fontHs []int
)

func initFontserver() {
	if dui == nil {
		// unit test
		return
	}
	if df := dui.Font(nil); df.Height >= 40 {
		dui.Display.DPI = 200
	}
}

func initFonts() {
	fonts = make(map[int]*draw.Font)
	fontHs = make([]int, 0, 5)
	if dui == nil {
		// unit tests
		return
	}
	def := dui.Display.Font.Name
	ms, err := fontsLike(def)
	if err != nil {
		log.Errorf("find fonts: %v", err)
	}

	log.Infof("fonts in directory: %+v\n", ms)
	for _, m := range ms {
		f, err := dui.Display.OpenFont(m)
		if err != nil {
			log.Errorf("open font: %v", err)
			continue
		}
		fonts[f.Height] = f
		fontHs = append(fontHs, f.Height)
	}
	log.Infof("font heights: %+v", fontHs)
}

var reNum = regexp.MustCompile(`(\d+(x\d+)?)`)

func fontId(fn string) string {
	return reNum.ReplaceAllString(fn, "")
}

func fontsLike(path string) (fts []string, err error) {
	fts = make([]string, 0, 5)

	if path == "*default*" {
		log.Infof("use default font")
		path = "/lib/font/bit/lucidasans/unicode.13.font"
	}

	l := strings.Split(path, "/")
	dn := strings.Join(l[:len(l)-1], "/")
	fn := l[len(l)-1]
	dir := os.DirFS(dn)
	ms, err := fs.Glob(dir, "*.font")
	if err != nil {
		return
	}

	log.Infof("fonts in directory: %+v\n", ms)
	for _, m := range ms {
		if fontId(fn) != fontId(m) {
			continue
		}
		log.Infof("add font %v\n", m)
		fts = append(fts, dn+"/"+m)
	}

	if len(fts) == 0 {
		return nil, fmt.Errorf("unable to find fonts in %v", dn)
	}

	return
}

func (cs Map) FontFilename() (fn string, ok bool) {
	if fonts == nil {
		initFonts()
	}
	h := matchClosestFontSize(2*cs.FontSize(), fontHs)
	f, ok := fonts[h]
	if !ok {
		return
	}
	return f.Name, true
}
