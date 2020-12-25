package browser

import (
	"fmt"
	"image"
	"strings"
	"opossum/domino"
	"opossum/nodes"
	"time"

	"9fans.net/go/draw"
	"github.com/mjl-/duit"
)

func (el *Element) slicedDraw(dui *duit.DUI, self *duit.Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) bool {
	//fmt.Printf("m.Point.y=%v\n", m.Point.Y)
	if experimentalUseSlicedDrawing {
		//offset := scroller.GetOffset()
		offset := -1
		panic("not implemented")
		fmt.Printf("orig=%v    m.Point.y=%v   offset=%v\n", orig.Y,m.Point.Y,offset)
		if (m.Point.Y-offset < -10 || m.Point.Y-offset > 1200) && isLeaf(el.UI) {
			return true
		}
	}
	return false
}

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
	case *duit.Grid:
		return false
	case *duit.Image:
		return true
	case *duit.Label:
		return true
	case *ColoredLabel:
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

func CleanTree(ui duit.UI) {
	if ui == nil {
		panic("nil root")
	}
	TraverseTree(ui, func(ui duit.UI) {
		if ui == nil {
			panic("null")
		}
		switch v := ui.(type) {
		case nil:
			panic("null")
		case *duit.Scroll:
			panic("like nil root")
		case *duit.Box:
			//realKids := make([])
		case *Element:
			if v == nil {
				panic("null element")
			}
		case *duit.Grid:
		case *duit.Image:
		case *duit.Label:
		case *ColoredLabel:
		case *duit.Button:
		case *Image:
		case *duit.Field:
		case *CodeView:
		default:
			panic(fmt.Sprintf("unknown: %+v", v))
		}
	})
}

func processJS(htm string) (resHtm string, err error) {
	_ = strings.Replace(htm, "window.", "", -1)
	d := domino.NewDomino(htm)
	d.Start()
	if err = d.ExecInlinedScripts(); err != nil {
		return "", fmt.Errorf("exec <script>s: %w", err)
	}
	time.Sleep(time.Second)
	resHtm, changed, err := d.TrackChanges()
	log.Infof("processJS: changes = %v", changed)
	d.Stop()
	return
}

func processJS2(d *domino.Domino, doc *nodes.Node, scripts []string) (resHtm string, err error) {
	code := ""
	for _, script := range scripts {
		code += `
			try {
		` + script + `;
		` + fmt.Sprintf(`
			console.log('==============');
			console.log('Success!');
			console.log('==============');
		`) + `
			} catch(e) {
				console.log('==============');
				console.log('Catch:');
				console.log(e);
				console.log('==============');
			}
		`
	}
	if err = d.Exec/*6*/(code); err != nil {
		return "", fmt.Errorf("exec <script>s: %w", err)
	}
	time.Sleep(time.Second)
	resHtm, changed, err := d.TrackChanges()
	if err != nil {
		return "", fmt.Errorf("track changes: %w", err)
	}
	log.Printf("processJS: changed = %v", changed)
	return
}
