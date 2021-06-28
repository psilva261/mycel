package browser

import (
	"fmt"
	"image"
	"github.com/psilva261/opossum/browser/duitx"
	"github.com/psilva261/opossum/domino"
	"9fans.net/go/draw"
	"github.com/mjl-/duit"
	"strings"
)

type AtomBox struct {
	Left, Right, Bottom, Top int
}

// Atom is div/span with contentEditable=true/false, i.e. it should be able
// to render practically anything
type Atom struct {
	// BackgroundImgSrc to read image from provided cache
	// it's okay when the pointer is empty -> defered loading
	BackgroundImgSrc string
	BackgroundColor draw.Color
	BorderWidths AtomBox
	Color draw.Color
	Margin AtomBox
	Padding AtomBox
	Wrap bool

	// Children []*Atom TODO: future; at the same time rething where
        //                                      to put Draw functions etc./if to rely on
        //                                      type Kid
	Text  string           // Text to draw, wrapped at glyph boundary.
	Font  *draw.Font       `json:"-"` // For drawing text.
	Click func()

	lines []string
	size  image.Point
	m     draw.Mouse
}

func (ui *Atom) font(dui *duit.DUI) *draw.Font {
	return dui.Font(ui.Font)
}

func (ui *Atom) Draw(dui *duit.DUI, self *duit.Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	// dui.debugDraw(self)

	p := orig
	font := ui.font(dui)
	for _, line := range ui.lines {
		img.String(p, dui.Regular.Normal.Text, image.ZP, font, line)
		p.Y += font.Height
	}
}

func isLeaf(ui duit.UI) bool {
	if ui == nil {
		return true
	}
	switch /*v := */ui.(type) {
	case nil:
		return true
	case *duit.Scroll:
		return false
	case *duit.Box:
		return false
	case *Element:
		return false
	case *duitx.Grid:
		return false
	case *duit.Image:
		return true
	case *duit.Label:
		return true
	case *Label:
		return false
	case *duit.Button:
		return true
	case *Image:
		return false
	case *duit.Field:
		return true
	case *CodeView:
		return false
	default:
		return false
	}
}

func processJS2(d *domino.Domino, scripts []string) (resHtm string, changed bool, err error) {
	initialized := false
	for _, script := range scripts {
		if _, err := d.Exec/*6*/(script, !initialized); err != nil {
			if strings.Contains(err.Error(), "halt at") {
				return "", false, fmt.Errorf("execution halted: %w", err)
			}
			log.Errorf("exec <script>: %v", err)
		}
		initialized = true
	}

	if err = d.CloseDoc(); err != nil {
		return "", false, fmt.Errorf("close doc: %w", err)
	}

	resHtm, changed, err = d.TrackChanges()
	if err != nil {
		return "", false, fmt.Errorf("track changes: %w", err)
	}
	log.Printf("processJS: changed = %v", changed)
	return
}
