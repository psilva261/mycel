package browser

import (
	"9fans.net/go/draw"
	"bytes"
	"errors"
	"fmt"
	"github.com/psilva261/opossum"
	"github.com/psilva261/opossum/browser/cache"
	"github.com/psilva261/opossum/browser/duitx"
	"github.com/psilva261/opossum/browser/fs"
	"github.com/psilva261/opossum/browser/history"
	"github.com/psilva261/opossum/img"
	"github.com/psilva261/opossum/js"
	"github.com/psilva261/opossum/logger"
	"github.com/psilva261/opossum/nodes"
	"github.com/psilva261/opossum/style"
	"golang.org/x/net/html"
	"golang.org/x/net/publicsuffix"
	"image"
	"io/ioutil"
	"math"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/mjl-/duit"
)

const (
	EnterKey = 10
)

var debugPrintHtml = false

// cursor based on Clipart from Francesco 'Architetto' Rollandin
// OpenClipart SVG ID: 163773 from OCAL 0.18 release 16/11/2019
// https://freesvg.org/topo-architetto-francesc-01
// (Public Domain)
var cursor = [16 * 2]uint8{
	0b00000001, 0b11111100,
	0b00000111, 0b11111110,
	0b00001111, 0b11111111,
	0b00111111, 0b11111111,
	0b00111111, 0b11111111,
	0b11111111, 0b11111111,
	0b11110111, 0b01111111,
	0b00111011, 0b11111111,
	0b00010011, 0b00111011,
	0b00000111, 0b00110110,
	0b00000111, 0b11111100,
	0b00001100, 0b01111000,
}

var (
	ExperimentalJsInsecure bool
	EnableNoScriptTag      bool
)

var (
	browser  *Browser
	Style    = style.Map{}
	dui      *duit.DUI
	scroller *duitx.Scroll
	display  *draw.Display

	selected  int
	dragRect  draw.Rectangle
	fromLabel *duitx.Label

	colorCache = make(map[draw.Color]*draw.Image)
	imageCache = make(map[string]*draw.Image)
)

type Label struct {
	*duitx.Label

	n *nodes.Node
}

func NewLabel(t string, n *nodes.Node) *Label {
	return &Label{
		Label: &duitx.Label{
			Text: t + " ",
			Font: n.Font(),
		},
		n: n,
	}
}

func NewText(content []string, n *nodes.Node) (el []*Element) {
	tt := strings.Join(content, " ")

	// '\n' is nowhere visible
	tt = strings.Replace(tt, "\n", " ", -1)

	ts := strings.Split(tt, " ")
	ls := make([]*Element, 0, len(ts))

	for _, t := range ts {
		t = strings.TrimSpace(t)

		if t == "" {
			continue
		}

		l := &Element{
			UI: NewLabel(t, n),
			n:  n,
		}
		ls = append(ls, l)
	}

	return ls
}

func (ui *Label) Draw(dui *duit.DUI, self *duit.Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	c := ui.n.Map.Color()
	i, ok := colorCache[c]
	if !ok {
		var err error
		i, err = dui.Display.AllocImage(image.Rect(0, 0, 10, 10), draw.ARGB32, true, c)
		if err != nil {
			panic(err.Error())
		}
		colorCache[c] = i
	}
	var swap *draw.Image = dui.Regular.Normal.Text
	dui.Regular.Normal.Text = i
	ui.Label.Draw(dui, self, img, orig, m, force)
	dui.Regular.Normal.Text = swap
}

type CodeView struct {
	duit.UI
}

func NewCodeView(s string, n style.Map) (cv *CodeView) {
	log.Printf("NewCodeView(%+v)", s)
	cv = &CodeView{}
	edit := &duit.Edit{
		Font: Style.Font(),
	}
	lines := len(strings.Split(s, "\n"))
	edit.Append([]byte(s))
	cv.UI = &duitx.Box{
		Kids:   duit.NewKids(edit),
		Height: int(n.FontHeight()) * (lines + 2),
	}
	return
}

func (cv *CodeView) Mouse(dui *duit.DUI, self *duit.Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r duit.Result) {
	if m.Buttons == 8 || m.Buttons == 16 {
		//r.Consumed = true
		return
	}
	return cv.UI.Mouse(dui, self, m, origM, orig)
}

type Image struct {
	*duit.Image

	src string
}

func NewImage(n *nodes.Node) duit.UI {
	img, err := newImage(n)
	if err != nil {
		log.Errorf("could not load image: %v", err)
		return &duit.Label{}
	}
	return img
}

func newImage(n *nodes.Node) (ui duit.UI, err error) {
	var i *draw.Image
	var cached bool
	src := attr(*n.DomSubtree, "src")
	log.Printf("newImage: src: %v", src)

	if src == img.SrcZero {
		return
	}

	if display == nil {
		// probably called from a unit test
		return nil, fmt.Errorf("display nil")
	}

	if n.Data() == "picture" {
		src = newPicture(n)
	} else if n.Data() == "svg" {
		xml, err := n.Serialized()
		if err != nil {
			return nil, fmt.Errorf("serialize: %w", err)
		}
		log.Printf("newImage: xml: %v", xml)
		buf, err := img.Svg(xml, n.Width(), n.Height())
		if err == nil {
			var err error
			r := bytes.NewReader(buf)
			i, err = duit.ReadImage(display, r)
			if err != nil {
				return nil, fmt.Errorf("read image %v: %v", xml, err)
			}

			goto img_elem
		} else {
			return nil, fmt.Errorf("img svg %v: %v", xml, err)
		}
	} else if n.Data() == "img" {
		_, s := srcSet(n)
		if s != "" {
			src = s
		}
	}

	if src == "" {
		return nil, fmt.Errorf("no src in %+v", n.DomSubtree.Attr)
	}

	if i, cached = imageCache[src]; !cached {
		mw, _ := n.CssPx("max-width")
		w := n.Width()
		h := n.Height()
		r, err := img.Load(browser, src, mw, w, h)
		if err != nil {
			return nil, fmt.Errorf("load draw image: %w", err)
		}
		log.Printf("Read %v...", src)
		i, err = duit.ReadImage(display, r)
		if err != nil {
			return nil, fmt.Errorf("duit read image: %w", err)
		}
		log.Printf("Done reading %v", src)
		imageCache[src] = i
	}

img_elem:
	return NewElement(
		&Image{
			Image: &duit.Image{
				Image: i,
			},
			src: src,
		},
		n,
	), nil
}

func newPicture(n *nodes.Node) string {
	smallestImg := ""
	smallestW := 0

	for _, source := range n.FindAll("source") {
		w, src := srcSet(source)
		if src != "" && (smallestImg == "" || smallestW > w) {
			smallestImg = src
			smallestW = w
		}
	}

	return smallestImg
}

func srcSet(n *nodes.Node) (w int, src string) {
	bestImg := ""
	bestW := 0
	idealW := n.Width()
	scale := 1

	u := func(wd int) int {
		return int(math.Abs(float64(wd) - float64(idealW)))
	}

	if dui != nil {
		scale = int(dui.Scale(1))
	}

	for _, s := range strings.Split(n.Attr("srcset"), ",") {
		s = strings.TrimSpace(s)
		tmp := strings.Split(s, " ")
		src := ""
		s := ""
		src = tmp[0]
		if len(tmp) == 2 {
			s = tmp[1]
		}
		if s == "" || s == fmt.Sprintf("%vx", scale) {
			return 0, src
		}
		s = strings.TrimSuffix(s, "w")
		w, err := strconv.Atoi(s)
		if err != nil {
			continue
		}
		if bestImg == "" || u(bestW) > u(w) {
			bestImg = src
			bestW = w
		}
	}

	return bestW, bestImg
}

type Element struct {
	duit.UI
	n       *nodes.Node
	orig    image.Point
	IsLink  bool
	Click   func() duit.Event
	Changed func(*Element)

	m    draw.Mouse
	rect image.Rectangle
}

func NewElement(ui duit.UI, n *nodes.Node) *Element {
	if ui == nil {
		return nil
	}
	if n == nil {
		log.Errorf("NewElement: n is nil")
		return nil
	}
	if n.IsDisplayNone() {
		return nil
	}

	if n.Type() != html.TextNode {
		if box, ok := newBoxElement(n, false, ui); ok {
			ui = box
		}
	}

	el := &Element{
		UI: ui,
		n:  n,
	}
	n.Rectangular = el
	return el
}

func newBoxElement(n *nodes.Node, force bool, uis ...duit.UI) (box *duitx.Box, ok bool) {
	if len(uis) == 0 || (len(uis) == 1 && uis[0] == nil) {
		return nil, false
	}
	if n != nil && n.IsDisplayNone() {
		return nil, false
	}

	var err error
	var i *draw.Image
	var m, p duit.Space
	var zs duit.Space
	var h int
	var w int
	var mw int

	if n != nil {
		w = n.Width()
		if n.Data() != "body" {
			h = n.Height()
		}
		mw, err = n.CssPx("max-width")
		if err != nil {
			log.Printf("max-width: %v", err)
		}

		if bg, err := n.BoxBackground(); err == nil {
			i = bg
		} else {
			log.Printf("box background: %f", err)
		}

		if p, err = n.Tlbr("padding"); err != nil {
			log.Errorf("padding: %v", err)
		}
		if m, err = n.Tlbr("margin"); err != nil {
			log.Errorf("margin: %v", err)
		}

		if n.Css("display") == "inline" {
			// Actually this doesn't fix the problem to the full extend
			// exploded texts' elements might still do double and triple
			// horizontal pads/margins
			w = 0
			mw = 0
			m.Top = 0
			m.Bottom = 0
			p.Top = 0
			p.Bottom = 0
		}

		// TODO: make sure input fields can be put into a box
		if n.Data() == "input" {
			return nil, false
		}
	}

	if w == 0 && h == 0 && mw == 0 && i == nil && m == zs && p == zs && !force {
		return nil, false
	}

	contentBox := n == nil || n.Css("box-sizing") != "border-box"
	box = &duitx.Box{
		Kids:       duit.NewKids(uis...),
		Width:      w,
		Height:     h,
		MaxWidth:   mw,
		ContentBox: contentBox,
		Background: i,
		Margin:     m,
		Padding:    p,
		Dir:        duitFlexDir(n),
		Disp:       duitDisplay(n),
	}

	return box, true
}

func (el *Element) Rect() (r image.Rectangle) {
	if el == nil {
		log.Errorf("Rect: nil element")
		return
	}
	return el.rect.Add(el.orig)
}

func (el *Element) Draw(dui *duit.DUI, self *duit.Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	if el == nil {
		return
	}

	// It would be possible to avoid flickers under certain circumstances
	// of overlapping elements but the load for this is high:
	// if self.Draw == duit.DirtyKid {
	//	force = true
	// }
	//
	// Make boxes use full size for image backgrounds
	box, ok := el.UI.(*duitx.Box)
	if ok && box.Width > 0 && box.Height > 0 {
		uiSize := image.Point{X: box.Width, Y: box.Height}
		duit.KidsDraw(dui, self, box.Kids, uiSize, box.Background, img, orig, m, force)
	} else {
		el.UI.Draw(dui, self, img, orig, m, force)
	}
	el.orig = orig
}

func (el *Element) Layout(dui *duit.DUI, self *duit.Kid, sizeAvail image.Point, force bool) {
	if el == nil {
		return
	}

	// TODO: constrain field width as long as boxing them is deactivated
	_, ok := el.UI.(*duit.Field)
	if ok && sizeAvail.X > dui.Scale(300) {
		sizeAvail.X = dui.Scale(300)
	}

	// Make boxes use full size for image backgrounds
	box, ok := el.UI.(*duitx.Box)
	if ok && box.Width > 0 && box.Height > 0 {
		//dui.debugLayout(self)
		//if ui.Image == nil {
		//	self.R = image.ZR
		//} else {
		//	self.R = rect(ui.Image.R.Size())
		//}
		//duit.KidsLayout(dui, self, box.Kids, true)

		el.UI.Layout(dui, self, sizeAvail, force)
		self.R = image.Rect(0, 0, box.Width, box.Height)
	} else {
		el.UI.Layout(dui, self, sizeAvail, force)
	}

	el.rect = self.R

	return
}

func (el *Element) Mark(self *duit.Kid, o duit.UI, forLayout bool) (marked bool) {
	if el != nil {
		return el.UI.Mark(self, o, forLayout)
	}
	return
}

func NewSubmitButton(b *Browser, n *nodes.Node) *Element {
	var t string

	if v := attr(*n.DomSubtree, "value"); v != "" {
		t = v
	} else if c := strings.TrimSpace(n.ContentString(false)); c != "" {
		t = c
	} else {
		t = "Submit"
	}

	// TODO: would be better to deal with *nodes.Node but keeping the correct
	// references in the closure is tricky. Probably better to write a separate
	// type Button to avoid this problem completely.
	click := func() (r duit.Event) {
		f := n.Ancestor("form")

		if f == nil {
			return
		}

		if !b.loading {
			b.loading = true
			go b.submit(f.DomSubtree, n.DomSubtree)
		}

		return duit.Event{
			Consumed:   true,
			NeedLayout: true,
			NeedDraw:   true,
		}
	}

	btn := &duit.Button{
		Text:  t,
		Font:  n.Font(),
		Click: click,
	}
	return NewElement(btn, n)
}

func NewInputField(n *nodes.Node) *Element {
	t := attr(*n.DomSubtree, "type")
	if n.Css("width") == "" && n.Css("max-width") == "" {
		n.SetCss("max-width", "200px")
	}
	f := &duit.Field{
		Font:        n.Font(),
		Placeholder: attr(*n.DomSubtree, "placeholder"),
		Password:    t == "password",
		Text:        attr(*n.DomSubtree, "value"),
		Changed: func(t string) (e duit.Event) {
			setAttr(n.DomSubtree, "value", t)
			e.Consumed = true
			return
		},
		Keys: func(k rune, m draw.Mouse) (e duit.Event) {
			if k == 10 {
				f := n.Ancestor("form")
				if f == nil {
					return
				}
				if !browser.loading {
					browser.loading = true
					go browser.submit(f.DomSubtree, nil)
				}
				return duit.Event{
					Consumed:   true,
					NeedLayout: true,
					NeedDraw:   true,
				}
			}
			return
		},
	}
	return NewElement(f, n)
}

func NewSelect(n *nodes.Node) *Element {
	var l *duit.List
	l = &duit.List{
		Values: make([]*duit.ListValue, 0, len(n.Children)),
		Font:   n.Font(),
		Changed: func(i int) (e duit.Event) {
			v := l.Values[i]
			vv := fmt.Sprintf("%v", v.Value)
			if vv == "" {
				vv = v.Text
			}
			setAttr(n.DomSubtree, "value", vv)
			e.Consumed = true
			return
		},
	}
	for _, c := range n.Children {
		if c.Data() != "option" {
			continue
		}
		lv := &duit.ListValue{
			Text:     c.ContentString(false),
			Value:    c.Attr("value"),
			Selected: c.HasAttr("selected"),
		}
		l.Values = append(l.Values, lv)
	}
	if n.Css("width") == "" && n.Css("max-width") == "" {
		n.SetCss("max-width", "200px")
	}
	if n.Css("height") == "" {
		n.SetCss("height", fmt.Sprintf("%vpx", 4*n.Font().Height))
	}
	return NewElement(duit.NewScroll(l), n)
}

func NewTextArea(n *nodes.Node) *Element {
	t := n.ContentString(true)
	lines := len(strings.Split(t, "\n"))
	edit := &duit.Edit{
		Font: Style.Font(),
		Keys: func(k rune, m draw.Mouse) (e duit.Event) {
			// e.Consumed = true
			return
		},
	}
	edit.Append([]byte(t))

	if n.Css("height") == "" {
		n.SetCss("height", fmt.Sprintf("%vpx", (int(n.FontHeight())*(lines+2))))
	}

	el := NewElement(edit, n)
	el.Changed = func(e *Element) {
		ed := e.UI.(*duitx.Box).Kids[0].UI.(*duit.Edit)

		tt, err := ed.Text()
		if err != nil {
			log.Errorf("edit changed: %v", err)
			return
		}

		e.n.SetText(string(tt))
	}

	return el
}

func (el *Element) Display() duitx.Display {
	var n *nodes.Node
	if el != nil {
		n = el.n
	}
	return duitDisplay(n)
}

func (el *Element) FlexDir() duitx.Dir {
	var n *nodes.Node
	if el != nil {
		n = el.n
	}
	return duitFlexDir(n)
}

func (el *Element) Key(dui *duit.DUI, self *duit.Kid, k rune, m draw.Mouse, orig image.Point) (r duit.Result) {
	r = el.UI.Key(dui, self, k, m, orig)

	if el.Changed != nil {
		el.Changed(el)
	}

	return
}

func (el *Element) Mouse(dui *duit.DUI, self *duit.Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r duit.Result) {
	if m.Buttons == 4 {
		if el == nil {
			log.Infof("inspect nil element")
		} else {
			p, _ := el.n.Path()
			log.Infof("%v", p)
		}
	}

	if el == nil {
		return
	}

	x := m.Point.X
	y := m.Point.Y
	maxX := self.R.Dx()
	maxY := self.R.Dy()
	border := 5 > x || x > (maxX-5) || 5 > y || y > (maxY-5)

	if l, ok := el.UI.(*Label); ok && l != nil {
		fromLabel = l.Label
	}
	if el.n.Data() == "body" {
		if el.mouseSelect(dui, self, m, origM, orig) {
			return duit.Result{
				Consumed: true,
			}
		}
	} else if el.m.Buttons&1 == 1 && m.Buttons&1 == 0 && el.click() {
		return duit.Result{
			Consumed: true,
		}
	}
	/*if !border && el.IsLink {
		dui.Display.SwitchCursor(&draw.Cursor{
			Black: cursor,
		})
		if m.Buttons == 0 {
			//r.Consumed = true
			return r
		}
	} else {
		dui.Display.SwitchCursor(nil)
	}*/
	if border {
		el.m = draw.Mouse{}
	} else {
		el.m = m
	}
	return el.UI.Mouse(dui, self, m, origM, orig)
}

func (el *Element) mouseSelect(dui *duit.DUI, self *duit.Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (consumed bool) {
	mouseDrag := m != origM
	changed := false
	if mouseDrag {
		from := origM.Point.Add(orig)
		to := m.Point.Add(orig)
		r := draw.Rectangle{
			draw.Point{from.X, from.Y},
			draw.Point{to.X, to.Y},
		}
		// make sure the same coordinates are used
		// (TODO: should be consistent in the first place)
		if rc := r.Canon(); r == rc {
			r = r.Sub(r.Min.Sub(fromLabel.Rect().Min))
		} else {
			r = rc.Sub(rc.Max.Sub(fromLabel.Rect().Max))
		}
		if !rectsSimilar(dragRect, r) {
			TraverseTree(el, func(ui duit.UI) {
				l, ok := ui.(*duitx.Label)
				if !ok {
					return
				}
				sel := l.Rect().Overlaps(r)
				if sel == l.Selected {
					return
				}
				l.Selected = sel
				changed = true
				if sel {
					selected++
				} else {
					selected--
				}
			})
			dragRect = r
		}
		if m.Buttons&2 == 2 && el.m.Buttons&2 == 0 {
			var s string
			var last *duitx.Label
			TraverseTree(el, func(ui duit.UI) {
				l, ok := ui.(*duitx.Label)
				if ok && l.Selected {
					if last != nil && l.Rect().Min.Y > last.Rect().Min.Y {
						s += "\n"
					}
					s += l.Text
					last = l
					return
				}
			})
			s = strings.TrimSpace(s)
			s = strings.TrimFunc(s, func(r rune) bool {
				return !unicode.IsGraphic(r)
			})
			s = strings.Map(func(r rune) rune {
				if unicode.IsSpace(r) && r != '\n' {
					return ' '
				}
				return r
			}, s)
			dui.WriteSnarf([]byte(s))
		}
	} else if selected > 0 && m.Buttons == 1 {
		TraverseTree(browser.Website.UI, func(ui duit.UI) {
			l, ok := ui.(*duitx.Label)
			if ok && l.Selected {
				selected--
				changed = true
				l.Selected = false
			}
		})
		selected = 0
	}
	return changed
}

func rectsSimilar(r, rr draw.Rectangle) bool {
	deltas := []float64{
		math.Abs(float64(r.Min.X - rr.Min.X)),
		math.Abs(float64(r.Min.Y - rr.Min.Y)),
		math.Abs(float64(r.Max.X - rr.Max.X)),
		math.Abs(float64(r.Max.Y - rr.Max.Y)),
	}
	for _, d := range deltas {
		if d > float64(dui.Scale(10)) {
			return false
		}
	}
	return true
}

func (el *Element) FirstFocus(dui *duit.DUI, self *duit.Kid) *image.Point {
	// Provide custom implementation with nil check because of nil Elements.
	// (TODO: remove)
	if el == nil {
		return &image.Point{}
	}
	return el.UI.FirstFocus(dui, self)
}

func (el *Element) click() (consumed bool) {
	if ExperimentalJsInsecure {
		q := el.n.QueryRef()
		res, consumed, err := js.TriggerClick(q)
		if err != nil {
			log.Errorf("trigger click %v: %v", q, err)
		} else if consumed {
			offset := scroller.Offset
			browser.Website.layout(browser, res, ClickRelayout)
			scroller.Offset = offset
			dui.MarkLayout(dui.Top.UI)
			dui.MarkDraw(dui.Top.UI)
			dui.Render()
		}
	}

	if el.Click != nil {
		e := el.Click()
		return e.Consumed
	}

	return
}

// makeLink of el and its children
func (el *Element) makeLink(href string) {
	if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "javascript:") {
		return
	}

	u, err := browser.LinkedUrl(href)
	if err != nil {
		log.Errorf("makeLink from %v: %v", href, err)
		return
	}
	f := browser.SetAndLoadUrl(u)
	TraverseTree(el, func(ui duit.UI) {
		el, ok := ui.(*Element)
		if ok && el != nil {
			el.IsLink = true
			el.Click = f
			return
		}
		l, ok := ui.(*duit.Label)
		if ok && l != nil {
			l.Click = f
			return
		}
	})
}

func attr(n html.Node, key string) (val string) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return
}

func hasAttr(n html.Node, key string) bool {
	for _, a := range n.Attr {
		if a.Key == key {
			return true
		}
	}
	return false
}

func setAttr(n *html.Node, key, val string) {
	newAttr := html.Attribute{
		Key: key,
		Val: val,
	}
	for i, a := range n.Attr {
		if a.Key == key {
			n.Attr[i] = newAttr
			return
		}
	}
	n.Attr = append(n.Attr, newAttr)
}

func placeFunc(name string, place *duitx.Place) func(self *duit.Kid, sizeAvail image.Point) {
	return func(self *duit.Kid, sizeAvail image.Point) {
		for i, kid := range place.Kids {
			el := kid.UI.(*Element)
			if i == 0 {
				kid.UI.Layout(dui, self, sizeAvail, true)
				kid.R = self.R
			} else {
				kid.UI.Layout(dui, kid, sizeAvail, true)
				if t, err := el.n.CssPx("top"); err == nil {
					kid.R.Min.Y += t
					kid.R.Max.Y += t
				}
				if l, err := el.n.CssPx("left"); err == nil {
					kid.R.Max.X += l
					kid.R.Min.X += l
				}
				if r, err := el.n.CssPx("right"); err == nil {
					w := kid.R.Max.X
					kid.R.Max.X = sizeAvail.X - r
					kid.R.Min.X = sizeAvail.X - w
				}
			}
		}
	}
}

// arrangeAbsolute positioned elements, if any
func arrangeAbsolute(n *nodes.Node, elements ...*Element) (ael *Element, ok bool) {
	absolutes := make([]*Element, 0, 1)
	other := make([]*Element, 0, len(elements))

	for _, el := range elements {
		if el.n.Css("position") == "absolute" {
			absolutes = append(absolutes, el)
		} else {
			other = append(other, el)
		}
	}

	if len(absolutes) == 0 {
		return nil, false
	}

	bg, err := dui.Display.AllocImage(image.Rect(0, 0, 10, 10), draw.ARGB32, true, 0x00000000)
	if err != nil {
		log.Fatalf("%v", err)
	}
	uis := make([]duit.UI, 0, len(other)+1)
	na := Arrange(n, other...)
	if na != nil {
		uis = append(uis, na)
	}
	for _, a := range absolutes {
		uis = append(uis, a)
	}
	pl := &duitx.Place{
		Kids:       duit.NewKids(uis...),
		Background: bg,
	}
	pl.Place = placeFunc(n.QueryRef(), pl)

	return NewElement(pl, n), true
}

func Arrange(n *nodes.Node, elements ...*Element) *Element {
	if ael, ok := arrangeAbsolute(n, elements...); ok {
		return ael
	}

	ui := horizontalSeq(n, true, elements)
	if ui == nil {
		return nil
	}
	el := &Element{
		n:  n,
		UI: ui,
	}
	n.Rectangular = el
	return el
}

func horizontalSeq(parent *nodes.Node, wrap bool, es []*Element) duit.UI {
	if len(es) == 0 {
		return nil
	}

	finalUis := make([]duit.UI, 0, len(es))
	for _, el := range es {
		label, isLabel := el.UI.(*duit.Label)
		if isLabel {
			tts := strings.Split(label.Text, " ")
			for _, t := range tts {
				finalUis = append(finalUis, NewElement(&duit.Label{
					Text: t,
					Font: label.Font,
				}, el.n))
			}
		} else {
			if el != nil {
				finalUis = append(finalUis, el)
			}
		}
	}

	b, ok := newBoxElement(parent, true, finalUis...)
	if !ok {
		return nil
	}
	return b
}

func duitDisplay(n *nodes.Node) duitx.Display {
	if n == nil {
		return duitx.InlineBlock
	}
	if n.Css("float") == "left" {
		return duitx.InlineBlock
	} else if cl := n.Css("clear"); cl == "left" || cl == "both" {
		return duitx.Block
	}
	switch n.Css("display") {
	case "inline":
		return duitx.Inline
	case "block":
		return duitx.Block
	case "flex":
		return duitx.Flex
	default:
		return duitx.InlineBlock
	}
}

func duitFlexDir(n *nodes.Node) duitx.Dir {
	if n == nil {
		return 0
	}
	switch n.Css("flex-direction") {
	case "row":
		return duitx.Row
	case "column":
		return duitx.Column
	default:
		return 0
	}
}

func verticalSeq(es []*Element) duit.UI {
	if len(es) == 0 {
		return nil
	} else if len(es) == 1 {
		return es[0]
	}

	uis := make([]duit.UI, 0, len(es))
	colSpans := make([]int, 0, len(es))
	rowSpans := make([]int, 0, len(es))

	for _, e := range es {
		uis = append(uis, e)
		colSpans = append(colSpans, 1)
		rowSpans = append(rowSpans, 1)
	}

	return &duitx.Grid{
		Columns:  1,
		Rows:     len(uis),
		ColSpans: colSpans,
		RowSpans: rowSpans,
		Padding:  duit.NSpace(1, duit.SpaceXY(0, 3)),
		Halign:   []duit.Halign{duit.HalignLeft},
		Valign:   []duit.Valign{duit.ValignTop},
		Kids:     duit.NewKids(uis...),
	}
}

type Table struct {
	rows []*TableRow
}

func NewTable(n *nodes.Node) (t *Table) {
	t = &Table{
		rows: make([]*TableRow, 0, 10),
	}

	if n.Text != "" || n.DomSubtree.Data != "table" {
		log.Printf("invalid table root")
		return nil
	}

	trContainers := make([]*nodes.Node, 0, 2)
	for _, c := range n.Children {
		if c.DomSubtree.Data == "tbody" || c.DomSubtree.Data == "thead" {
			trContainers = append(trContainers, c)
		}
	}
	if len(trContainers) == 0 {
		trContainers = []*nodes.Node{n}
	}

	for _, tc := range trContainers {
		for _, c := range tc.Children {
			if txt := c.Text; txt != "" && strings.TrimSpace(txt) == "" {
				continue
			}
			if c.DomSubtree.Data == "tr" {
				row := NewTableRow(c)
				t.rows = append(t.rows, row)
			} else {
				log.Printf("unexpected row element '%v' (%v)", c.DomSubtree.Data, c.DomSubtree.Type)
			}
		}
	}
	return
}

func (t *Table) numColsMin() (min int) {
	min = t.numColsMax()
	for _, r := range t.rows {
		if l := len(r.columns); l < min {
			min = l
		}
	}
	return
}

func (t *Table) numColsMax() (max int) {
	for _, r := range t.rows {
		if l := len(r.columns); l > max {
			max = l
		}
	}
	return
}

func (t *Table) Element(r int, b *Browser, n *nodes.Node) *Element {
	numRows := len(t.rows)
	numCols := t.numColsMax()
	useOneGrid := t.numColsMin() == t.numColsMax()

	if numCols == 0 {
		return nil
	}

	if useOneGrid {
		uis := make([]duit.UI, 0, numRows*numCols)

		for _, row := range t.rows {
			for _, td := range row.columns {
				uis = append(uis, NodeToBox(r+1, b, td))
			}
		}

		colSpans := make([]int, 0, len(uis))
		rowSpans := make([]int, 0, len(uis))
		halign := make([]duit.Halign, 0, len(uis))
		valign := make([]duit.Valign, 0, len(uis))

		for j := 0; j < numCols; j++ {
			for i := 0; i < len(t.rows); i++ {
				colSpans = append(colSpans, 1)
				rowSpans = append(rowSpans, 1)
			}
			halign = append(halign, duit.HalignLeft)
			valign = append(valign, duit.ValignTop)
		}

		return NewElement(
			&duitx.Grid{
				Columns:  numCols,
				Rows:     len(t.rows),
				ColSpans: colSpans,
				RowSpans: rowSpans,
				Padding:  duit.NSpace(numCols, duit.SpaceXY(0, 3)),
				Halign:   halign,
				Valign:   valign,
				Kids:     duit.NewKids(uis...),
			},
			n,
		)
	} else {
		seqs := make([]*Element, 0, len(t.rows))

		for _, row := range t.rows {
			rowEls := make([]*Element, 0, len(row.columns))
			for _, col := range row.columns {
				ui := NodeToBox(r+1, b, col)
				if ui != nil {
					el := NewElement(ui, col)
					rowEls = append(rowEls, el)
				}
			}

			if len(rowEls) > 0 {
				seq := horizontalSeq(nil, false, rowEls)
				seqs = append(seqs, NewElement(seq, row.n))
			}
		}
		return NewElement(verticalSeq(seqs), n)
	}
}

type TableRow struct {
	n       *nodes.Node
	columns []*nodes.Node
}

func NewTableRow(n *nodes.Node) (tr *TableRow) {
	tr = &TableRow{
		n:       n,
		columns: make([]*nodes.Node, 0, 5),
	}

	if n.Type() != html.ElementNode || n.Data() != "tr" {
		log.Printf("invalid tr root")
		return nil
	}

	for _, c := range n.Children {
		if c.Type() == html.TextNode && strings.TrimSpace(c.Data()) == "" {
			continue
		}
		if c.DomSubtree.Data == "td" || c.DomSubtree.Data == "th" {
			tr.columns = append(tr.columns, c)
		} else {
			log.Printf("unexpected row element '%v' (%v)", c.Data(), c.Type())
		}
	}

	return tr
}

func grep(n *html.Node, tag string) *html.Node {
	var t *html.Node

	if n.Type == html.ElementNode {
		if n.Data == tag {
			return n
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		res := grep(c, tag)
		if res != nil {
			t = res
		}
	}

	return t
}

func NodeToBox(r int, b *Browser, n *nodes.Node) (el *Element) {
	if n.Attr("aria-hidden") == "true" || n.Attr("hidden") != "" {
		return
	}

	if n.IsDisplayNone() {
		return
	}

	if n.Type() == html.ElementNode {
		switch n.Data() {
		case "style", "script", "template":
			return
		case "input":
			t := n.Attr("type")
			if t == "" || t == "text" || t == "email" || t == "search" || t == "password" {
				return NewInputField(n)
			} else if t == "submit" {
				return NewSubmitButton(b, n)
			}
		case "select":
			return NewSelect(n)
		case "textarea":
			return NewTextArea(n)
		case "button":
			if t := n.Attr("type"); t == "" || t == "submit" {
				return NewSubmitButton(b, n)
			}

			btn := &duit.Button{
				Text: n.ContentString(false),
				Font: n.Font(),
			}

			return NewElement(btn, n)
		case "table":
			return NewTable(n).Element(r+1, b, n)
		case "picture", "img", "svg":
			return NewElement(NewImage(n), n)
		case "pre":
			return NewElement(
				NewCodeView(n.ContentString(true), n.Map),
				n,
			)
		case "li":
			var innerContent duit.UI

			if nodes.IsPureTextContent(*n) {
				t := n.ContentString(false)

				if ul := n.Ancestor("ul"); ul != nil {
					if ul.Css("list-style") != "none" && n.Css("list-style-type") != "none" {
						t = "• " + t
					}
				}
				innerContent = NewLabel(t, n)
			} else {
				return InnerNodesToBox(r+1, b, n)
			}

			return NewElement(innerContent, n)
		case "a":
			var href = n.Attr("href")
			var innerContent duit.UI
			if nodes.IsPureTextContent(*n) {
				innerContent = NewLabel(
					n.ContentString(false),
					n,
				)
			} else {
				innerContent = InnerNodesToBox(r+1, b, n)
			}
			if innerContent == nil {
				return nil
			}
			el := NewElement(innerContent, n)
			el.makeLink(href)
			return el
		case "noscript":
			if ExperimentalJsInsecure || !EnableNoScriptTag {
				return
			}
			fallthrough
		default:
			return InnerNodesToBox(r+1, b, n)
		}
	} else if n.Type() == html.TextNode {
		// Leaf text object

		if text := n.ContentString(false); text != "" {
			ui := NewLabel(text, n)

			return NewElement(ui, n)
		}
	}

	return
}

func isWrapped(n *nodes.Node) bool {
	isText := nodes.IsPureTextContent(*n)
	isCTag := false
	for _, t := range []string{"span", "i", "b", "tt"} {
		if n.Data() == t {
			isCTag = true
		}
	}
	return ((isCTag && n.IsInline()) || n.Type() == html.TextNode) && isText
}

func InnerNodesToBox(r int, b *Browser, n *nodes.Node) *Element {
	items := n.CBItems()
	els := make([]*Element, 0, len(items))

	for _, c := range items {
		if c.IsDisplayNone() {
			continue
		}
		if isWrapped(c) {
			ls := NewText(c.Content(false), c)
			els = append(els, ls...)
		} else if nodes.IsPureTextContent(*n) && n.IsInline() {
			// Handle text wrapped in unwrappable tags like p, div, ...
			ls := NewText(c.Content(false), items[0])
			if len(ls) == 0 {
				continue
			}
			el := NewElement(horizontalSeq(c, true, ls), c)
			if el == nil {
				continue
			}
			els = append(els, el)
		} else if el := NodeToBox(r+1, b, c); el != nil {
			els = append(els, el)
		}
	}

	if len(els) == 0 {
		return nil
	}

	return Arrange(n, els...)
}

func TraverseTree(ui duit.UI, f func(ui duit.UI)) {
	traverseTree(0, ui, f)
}

func traverseTree(r int, ui duit.UI, f func(ui duit.UI)) {
	if ui == nil {
		panic("null")
	}
	f(ui)
	switch v := ui.(type) {
	case nil:
		panic("null")
	case *duitx.Scroll:
		traverseTree(r+1, v.Kid.UI, f)
	case *duitx.Box:
		for _, kid := range v.Kids {
			traverseTree(r+1, kid.UI, f)
		}
	case *Element:
		if v == nil {
			// TODO: repair?!
			//panic("null element")
			return
		}
		traverseTree(r+1, v.UI, f)
	case *duitx.Grid:
		for _, kid := range v.Kids {
			traverseTree(r+1, kid.UI, f)
		}
	case *duit.Image:
	case *duit.Label, *duitx.Label:
	case *Label:
		traverseTree(r+1, v.Label, f)
	case *Image:
		traverseTree(r+1, v.Image, f)
	case *duit.Field:
	case *duit.Edit:
	case *duit.Button:
	case *duit.List:
	case *duitx.Place:
		for _, kid := range v.Kids {
			traverseTree(r+1, kid.UI, f)
		}
	case *duit.Scroll:
	case *CodeView:
	default:
		panic(fmt.Sprintf("unknown: %+v", v))
	}
}

func PrintTree(ui duit.UI) {
	if log.Debug && debugPrintHtml {
		printTree(0, ui)
	}
}

func printTree(r int, ui duit.UI) {
	for i := 0; i < r; i++ {
		fmt.Printf("  ")
	}
	if ui == nil {
		fmt.Printf("ui=nil\n")
		return
	}
	switch v := ui.(type) {
	case nil:
		fmt.Printf("v=nil\n")
		return
	case *duit.Scroll:
		fmt.Printf("duit.Scroll\n")
		printTree(r+1, v.Kid.UI)
	case *duitx.Box:
		fmt.Printf("Box\n")
		for _, kid := range v.Kids {
			printTree(r+1, kid.UI)
		}
	case *Element:
		if v == nil {
			fmt.Printf("v:*Element=nil\n")
			return
		}
		fmt.Printf("Element\n")
		printTree(r+1, v.UI)
	case *duitx.Grid:
		fmt.Printf("Grid %vx%v\n", len(v.Kids)/v.Columns, v.Columns)
		for _, kid := range v.Kids {
			printTree(r+1, kid.UI)
		}
	case *duit.Image:
		fmt.Printf("Image %v\n", v)
	case *duit.Label:
		t := v.Text
		if len(t) > 20 {
			t = t[:15] + "..."
		}
		fmt.Printf("Label %v\n", t)
	case *Label:
		t := v.Text
		if len(t) > 20 {
			t = t[:15] + "..."
		}
		fmt.Printf("Label %v\n", t)
	default:
		fmt.Printf("%+v\n", v)
	}
}

type Browser struct {
	history.History
	dui      *duit.DUI
	Website  *Website
	loading  bool
	client   *http.Client
	Download func(res chan *string)
	LocCh    chan string
	StatusCh chan string
}

func NewBrowser(_dui *duit.DUI, initUrl string) (b *Browser) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Fatal(err)
	}

	b = &Browser{
		client: &http.Client{
			Jar: jar,
		},
		dui:      _dui,
		Website:  &Website{},
		LocCh:    make(chan string, 10),
		StatusCh: make(chan string, 10),
	}

	u, err := url.Parse(initUrl)
	if err != nil {
		log.Fatalf("parse: %v", err)
	}
	b.History.Push(u, 0)

	browser = b
	b.Website.UI = &duit.Label{}
	style.SetFetcher(b)
	dui = _dui
	dui.Background, err = dui.Display.AllocImage(image.Rect(0, 0, 10, 10), draw.ARGB32, true, 0x00000000)
	if err != nil {
		log.Fatalf("%v", err)
	}
	display = dui.Display
	b.LoadUrl(u)

	if ExperimentalJsInsecure {
		fs.Client = &http.Client{}
		fs.Fetcher = browser
	}
	go fs.Srv9p()

	return
}

func (b *Browser) LinkedUrl(addr string) (a *url.URL, err error) {
	log.Printf("LinkedUrl: addr=%v, b.URL=%v", addr, b.URL())
	if strings.HasPrefix(addr, "//") {
		addr = b.URL().Scheme + ":" + addr
	} else if strings.HasPrefix(addr, "/") {
		addr = b.URL().Scheme + "://" + b.URL().Host + addr
	} else if !strings.HasPrefix(addr, "http") {
		if strings.HasSuffix(b.URL().Path, "/") {
			addr = "/" + b.URL().Path + addr
		} else {
			m := strings.LastIndex(b.URL().Path, "/")
			if m > 0 {
				folder := b.URL().Path[0:m]
				addr = "/" + folder + "/" + addr
			} else {
				addr = "/" + addr
			}
		}
		addr = strings.ReplaceAll(addr, "//", "/")
		addr = b.URL().Scheme + "://" + b.URL().Host + addr
	}
	return url.Parse(addr)
}

func (b *Browser) Origin() *url.URL {
	return b.History.URL()
}

func (b *Browser) Back() (e duit.Event) {
	if !b.loading {
		b.History.Back()
		b.LocCh <- b.History.URL().String()
		b.LoadUrl(b.History.URL())
	}
	e.Consumed = true
	return
}

func (b *Browser) SetAndLoadUrl(u *url.URL) func() duit.Event {
	return func() duit.Event {
		// Stop updating existing widgets
		if scroller != nil {
			scroller.Free()
			scroller = nil
		}
		b.showBodyMessage("")

		if !b.loading {
			b.LocCh <- u.String()
			b.LoadUrl(u)
		}

		return duit.Event{
			Consumed: true,
		}
	}
}

func (b *Browser) Loading() bool {
	return b.loading
}

func (b *Browser) showBodyMessage(msg string) {
	b.Website.UI = &duit.Label{
		Text: msg,
		Font: style.Map{}.Font(),
	}
	dui.MarkLayout(dui.Top.UI)
	dui.MarkDraw(dui.Top.UI)
	dui.Render()
}

// LoadUrl after from location field,
func (b *Browser) LoadUrl(url *url.URL) (e duit.Event) {
	b.loading = true
	go b.loadUrl(url)
	e.Consumed = true

	return
}

func (b *Browser) loadUrl(url *url.URL) {
	b.StatusCh <- fmt.Sprintf("Load %v...", url)
	buf, contentType, err := b.get(url, true)
	if err != nil {
		log.Errorf("error loading %v: %v", url, err)
		if er := errors.Unwrap(err); er != nil {
			err = er
		}
		b.showBodyMessage(err.Error())
		b.loading = false
		return
	}
	if contentType.IsHTML() || contentType.IsPlain() || contentType.IsEmpty() {
		b.render(contentType, buf)
	} else {
		res := make(chan *string, 1)
		b.Download(res)

		log.Infof("Download unhandled content type: %v", contentType)

		fn := <-res

		if fn != nil && *fn != "" {
			log.Infof("Download to %v", *fn)
			f, _ := os.Create(*fn)
			f.Write(buf)
			f.Close()
		}
		dui.Call <- func() {
			b.loading = false
		}
	}
}

func (b *Browser) render(ct opossum.ContentType, buf []byte) {
	log.Printf("Empty some cache...")
	cache.Tidy()
	imageCache = make(map[string]*draw.Image)

	b.Website.ContentType = ct
	htm := ct.Utf8(buf)
	b.Website.layout(b, htm, InitialLayout)

	log.Printf("Render...")
	dui.Call <- func() {
		TraverseTree(b.Website.UI, func(ui duit.UI) {
			// just checking for nil elements. That would be a bug anyway and it's better
			// to notice it before it gets rendered

			if ui == nil {
				panic("nil")
			}
		})
		PrintTree(b.Website.UI)
		if scroller != nil {
			scroller.Offset = b.History.Scroll()
		}
		dui.MarkLayout(dui.Top.UI)
		dui.MarkDraw(dui.Top.UI)
		dui.Render()
		b.loading = false
	}
	log.Printf("Rendering done")
}

func (b *Browser) Get(uri *url.URL) (buf []byte, contentType opossum.ContentType, err error) {
	c, ok := cache.Get(uri.String())
	if ok {
		log.Printf("use %v from cache", uri)
	} else {
		c.Addr = uri.String()
		c.Buf, c.ContentType, err = b.get(uri, false)
		if err == nil {
			cache.Set(c)
		}
	}

	return c.Buf, c.ContentType, err
}

func (b *Browser) get(uri *url.URL, isNewOrigin bool) (buf []byte, contentType opossum.ContentType, err error) {
	log.Infof("Get %v", uri.String())
	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return
	}
	req.Header.Add("User-Agent", "opossum")
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, opossum.ContentType{}, fmt.Errorf("error loading %v: %w", uri, err)
	}
	defer resp.Body.Close()
	buf, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, opossum.ContentType{}, fmt.Errorf("error reading")
	}
	contentType, err = opossum.NewContentType(resp.Header.Get("Content-Type"), resp.Request.URL)
	if isNewOrigin {
		of := 0
		if scroller != nil {
			of = scroller.Offset
		}
		b.History.Push(resp.Request.URL, of)
		log.Printf("b.History is now %s", b.History.String())
		b.LocCh <- b.URL().String()
	}
	return
}

func (b *Browser) PostForm(uri *url.URL, data url.Values) (buf []byte, contentType opossum.ContentType, err error) {
	b.StatusCh <- "Posting..."
	fb := strings.NewReader(escapeValues(b.Website.ContentType, data).Encode())
	req, err := http.NewRequest("POST", uri.String(), fb)
	if err != nil {
		return
	}
	req.Header.Add("User-Agent", "opossum")
	req.Header.Set("Content-Type", fmt.Sprintf("application/x-www-form-urlencoded; charset=%v", b.Website.Charset()))
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, opossum.ContentType{}, fmt.Errorf("error loading %v: %w", uri, err)
	}
	defer resp.Body.Close()
	b.History.Push(resp.Request.URL, scroller.Offset)
	buf, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, opossum.ContentType{}, fmt.Errorf("error reading")
	}
	contentType, err = opossum.NewContentType(resp.Header.Get("Content-Type"), resp.Request.URL)
	return
}
