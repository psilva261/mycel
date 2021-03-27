package browser

import (
	"9fans.net/go/draw"
	"bytes"
	"errors"
	"fmt"
	"golang.org/x/net/html"
	"golang.org/x/net/publicsuffix"
	"image"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"github.com/psilva261/opossum"
	"github.com/psilva261/opossum/img"
	"github.com/psilva261/opossum/logger"
	"github.com/psilva261/opossum/nodes"
	"github.com/psilva261/opossum/style"
	"os"
	"strconv"
	"strings"

	"github.com/mjl-/duit"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

const debugPrintHtml = false

const EnterKey = 10

var cursor = [16*2]uint8{
	0b11111111, 0b11111110,
	0b11111111, 0b11111100,
	0b11111111, 0b11111000,
	0b11111111, 0b11110000,
	0b11111111, 0b11100000,
	0b11111111, 0b11000000,
	0b11111111, 0b10000000,
	0b11111111, 0b10000000,
	0b11111111, 0b11111111,
	0b11111111, 0b11111110,
	0b11110111, 0b11111110,
	0b11100011, 0b11110000,
}

var DebugDumpCSS *bool
var ExperimentalJsInsecure *bool
var EnableNoScriptTag *bool

var browser *Browser // TODO: limit global objects;
//       at least put them in separate pkgs
//       with well chosen private/public
var Style = style.Map{}
var dui *duit.DUI
var colorCache = make(map[draw.Color]*draw.Image)
var imageCache = make(map[string]*draw.Image)
var cache = make(map[string]struct {
	opossum.ContentType
	buf []byte
})
var log *logger.Logger
var scroller *Scroll
var display *draw.Display

func SetLogger(l *logger.Logger) {
	log = l
}

type Label struct {
	*duit.Label

	n *nodes.Node
}

func NewLabel(t string, n *nodes.Node) *Label {
	return &Label{
		Label: &duit.Label{
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
			n: n,
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
	formatted := ""
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		formatted += strings.TrimSpace(line) + "\n"
	}
	edit.Append([]byte(formatted))
	cv.UI = &Box{
		Kids:   duit.NewKids(edit),
		Height: n.Font().Height * (len(lines)+2),
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
		if  err != nil {
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
	}

	if src == "" {
		return nil, fmt.Errorf("no src in %+v", n.Attrs)
	}

	if i, cached = imageCache[src]; !cached {
		r, err := img.Load(browser, src, n.Width(), n.Height())
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
	scale := 1

	if dui != nil {
		scale = int(dui.Scale(1))
	}

	for _, source := range n.FindAll("source") {
		for _, s := range strings.Split(source.Attr("srcset"), ",") {
			s = strings.TrimSpace(s)
			tmp := strings.Split(s, " ")
			src := ""
			s := ""
			src = tmp[0]
			if len(tmp) == 2 {
				s = tmp[1]
			}
			if s == "" || s == fmt.Sprintf("%vx", scale) {
				return src
			}
			s = strings.TrimSuffix(s, "w")
			w, err := strconv.Atoi(s)
			if err != nil {
				continue
			}
			if smallestImg == "" || smallestW > w {
				smallestImg = src
				smallestW = w
			}
		}
	}

	return smallestImg
}

type Element struct {
	duit.UI
	n       *nodes.Node
	IsLink bool
	Click  func() duit.Event
	Changed func(*Element)
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
		if box, ok := newBoxElement(ui, n); ok {
			ui = box
		}
	}

	return &Element{
		UI: ui,
		n: n,
	}
}

func newBoxElement(ui duit.UI, n *nodes.Node) (box *Box, ok bool) {
	if ui == nil {
		return nil, false
	}
	if n.IsDisplayNone() {
		return nil, false
	}

	var err error
	var i *draw.Image
	var m, p duit.Space
	zs := duit.Space{}
	w := n.Width()
	h := n.Height()
	mw, err := n.CssPx("max-width")
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

	if w == 0 && h == 0 && mw == 0 && i == nil && m == zs && p == zs {
		return nil, false
	}

	box = &Box{
		Kids:       duit.NewKids(ui),
		Width:      w,
		Height:     h,
		MaxWidth: mw,
		ContentBox: true,
		Background: i,
		Margin: m,
		Padding: p,
	}

	return box, true
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
	box, ok := el.UI.(*Box)
	if ok && box.Width > 0 && box.Height > 0 {
		uiSize := image.Point{X: box.Width, Y: box.Height}
		duit.KidsDraw(dui, self, box.Kids, uiSize, box.Background, img, orig, m, force)
	} else {
		el.UI.Draw(dui, self, img, orig, m, force)
	}
}

func (el *Element) Layout(dui *duit.DUI, self *duit.Kid, sizeAvail image.Point, force bool) {
	if el == nil {
		return
	}
	// Make boxes use full size for image backgrounds
	box, ok := el.UI.(*Box)
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
	} else if c := strings.TrimSpace(n.ContentString()); c != "" {
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
		Text: t,
		Font: n.Font(),
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
		Font: n.Font(),
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
			Text: c.ContentString(),
			Value: c.Attr("value"),
			Selected: c.HasAttr("selected"),
		}
		l.Values = append(l.Values, lv)
	}
	if n.Css("width") == "" && n.Css("max-width") == "" {
		n.SetCss("max-width", "200px")
	}
	if n.Css("height") == "" {
		n.SetCss("height", fmt.Sprintf("%vpx", 4 * n.Font().Height))
	}
	return NewElement(NewScroll(l), n)
}

func NewTextArea(n *nodes.Node) *Element {
	t := n.ContentString()
	formatted := ""
	lines := strings.Split(t, "\n")
	for _, line := range lines {
		formatted += line + "\n"
	}
	edit := &duit.Edit{
		Font: Style.Font(),
		Keys: func(k rune, m draw.Mouse) (e duit.Event) {
			// e.Consumed = true
			return
		},
	}
	edit.Append([]byte(formatted))

	if n.Css("height") == "" {
		n.SetCss("height", fmt.Sprintf("%vpx", (n.Font().Height * (len(lines)+2))))
	}

	el := NewElement(edit, n)
	el.Changed = func(e *Element) {
		ed := e.UI.(*Box).Kids[0].UI.(*duit.Edit)

		tt, err := ed.Text()
		if err != nil {
			log.Errorf("edit changed: %v", err)
			return
		}

		e.n.SetText(string(tt))
	}

	return el
}

func (el *Element) Key(dui *duit.DUI, self *duit.Kid, k rune, m draw.Mouse, orig image.Point) (r duit.Result) {
	r = el.UI.Key(dui, self, k, m, orig)

	if el.Changed != nil {
		el.Changed(el)
	}

	return
}

func (el *Element) Mouse(dui *duit.DUI, self *duit.Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r duit.Result) {
	if m.Buttons == 2 {
		if el == nil {
			log.Infof("inspect nil element")
		} else {
			log.Infof("inspect el %+v %+v %+v", el, el.n, el.UI)
		}
	}

	if el == nil {
		return
	}

	if m.Buttons == 1 {
		if el.click() {
			return duit.Result{
				Consumed: true,
			}
		}
	}
	x := m.Point.X
	y := m.Point.Y
	maxX := self.R.Dx()
	maxY := self.R.Dy()
	if 5 <= x && x <= (maxX-5) && 5 <= y && y <= (maxY-5) && el.IsLink {
		dui.Display.SetCursor(&draw.Cursor{
			Set: cursor,
		})
		if m.Buttons == 0 {
			r.Consumed = true
			return r
		}
	} else {
		dui.Display.SetCursor(nil)
	}

	return el.UI.Mouse(dui, self, m, origM, orig)
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
	if el.Click != nil {
		e := el.Click()
		return e.Consumed
	}

	if !*ExperimentalJsInsecure {
		return
	}

	q := el.n.QueryRef()
	res, consumed, err := browser.Website.d.TriggerClick(q)
	if err != nil {
		log.Errorf("trigger click %v: %v", q, err)
		return
	}

	if !consumed {
		return
	}
	log.Infof("click processed")

	offset := scroller.Offset
	browser.Website.html = res
	browser.Website.layout(browser, ClickRelayout)
	scroller.Offset = offset
	dui.MarkLayout(dui.Top.UI)
	dui.MarkDraw(dui.Top.UI)
	dui.Render()

	return
}

// makeLink of el and its children
func (el *Element) makeLink(href string) {
	if href == "" || strings.HasPrefix(href, "#") || strings.Contains(href, "javascript:void(0)") {
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

func Arrange(n *nodes.Node, elements ...*Element) *Element {
	if n.IsFlex() {
		if n.IsFlexDirectionRow() {
			return NewElement(horizontalSeq(true, elements), n)
		} else {
			return NewElement(verticalSeq(elements), n)
		}
	}

	rows := make([][]*Element, 0, 10)
	currentRow := make([]*Element, 0, 10)
	flushCurrentRow := func() {
		if len(currentRow) > 0 {
			rows = append(rows, currentRow)
			currentRow = make([]*Element, 0, 10)
		}
	}

	for _, e := range elements {
		isInline := e.n.IsInline() || e.n.Type() == html.TextNode
		if !isInline {
			flushCurrentRow()
		}
		currentRow = append(currentRow, e)
		if !isInline {
			flushCurrentRow()
		}
	}
	flushCurrentRow()
	if len(rows) == 0 {
		return nil
	} else if len(rows) == 1 {
		if len(rows[0]) == 0 {
			return nil
		} else if len(rows[0]) == 1 {
			return rows[0][0]
		}
		s := horizontalSeq(true, rows[0])
		if el, ok := s.(*Element); ok {
			return el
		}
		return NewElement(s, n)
	} else {
		seqs := make([]*Element, 0, len(rows))
		for _, row := range rows {
			seq := horizontalSeq(true, row)
			if el, ok := seq.(*Element); ok {
				seqs = append(seqs, el)
			} else {
				seqs = append(seqs, NewElement(seq, n))
			}
		}
		s := verticalSeq(seqs)
		if el, ok := s.(*Element); ok {
			return el
		}
		return NewElement(s, n)
	}
}

func horizontalSeq(wrap bool, es []*Element) duit.UI {
	if len(es) == 0 {
		return nil
	} else if len(es) == 1 {
		return es[0]
	}

	halign := make([]duit.Halign, 0, len(es))
	valign := make([]duit.Valign, 0, len(es))

	for i := 0; i < len(es); i++ {
		halign = append(halign, duit.HalignLeft)
		valign = append(valign, duit.ValignTop)
	}

	uis := make([]duit.UI, 0, len(es))
	for _, e := range es {
		uis = append(uis, e)
	}

	if wrap {
		finalUis := make([]duit.UI, 0, len(uis))
		for _, ui := range uis {
			PrintTree(ui)
			el, ok := ui.(*Element)

			if ok {
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
					finalUis = append(finalUis, ui)
				}
			} else {
				finalUis = append(finalUis, ui)
			}
		}

		return &Box{
			Padding: duit.SpaceXY(6, 4),
			Margin:  duit.SpaceXY(6, 4),
			Kids:    duit.NewKids(finalUis...),
		}
	} else {
		return &duit.Grid{
			Columns: len(es),
			Padding: duit.NSpace(len(es), duit.SpaceXY(0, 3)),
			Halign:  halign,
			Valign:  valign,
			Kids:    duit.NewKids(uis...),
		}
	}
}

func verticalSeq(es []*Element) duit.UI {
	if len(es) == 0 {
		return nil
	} else if len(es) == 1 {
		return es[0]
	}

	uis := make([]duit.UI, 0, len(es))
	for _, e := range es {
		uis = append(uis, e)
	}

	return &duit.Grid{
		Columns: 1,
		Padding: duit.NSpace(1, duit.SpaceXY(0, 3)),
		Halign:  []duit.Halign{duit.HalignLeft},
		Valign:  []duit.Valign{duit.ValignTop},
		Kids:    duit.NewKids(uis...),
	}
}

func check(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s\n", msg, err)
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

		halign := make([]duit.Halign, 0, len(uis))
		valign := make([]duit.Valign, 0, len(uis))

		for i := 0; i < numCols; i++ {
			halign = append(halign, duit.HalignLeft)
			valign = append(valign, duit.ValignTop)
		}

		return NewElement(
			&duit.Grid{
				Columns: numCols,
				Padding: duit.NSpace(numCols, duit.SpaceXY(0, 3)),
				Halign:  halign,
				Valign:  valign,
				Kids:    duit.NewKids(uis...),
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
				seq := horizontalSeq(false, rowEls)
				seqs = append(seqs, NewElement(seq, row.n))
			}
		}
		return NewElement(verticalSeq(seqs), n)
	}
}

type TableRow struct {
	n *nodes.Node
	columns []*nodes.Node
}

func NewTableRow(n *nodes.Node) (tr *TableRow) {
	tr = &TableRow{
		n: n,
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
			if t == "" || t == "text" || t == "search" || t == "password" {
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
				Text: n.ContentString(),
				Font: n.Font(),
			}

			return NewElement(btn, n)
		case "table":
			return NewTable(n).Element(r+1, b, n)
		case "picture", "img", "svg":
			return NewElement(NewImage(n), n)
		case "pre":
			return NewElement(
				NewCodeView(n.ContentString(), n.Map),
				n,
			)
		case "li":
			var innerContent duit.UI

			if nodes.IsPureTextContent(*n) {
				t := n.ContentString()

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
			href := n.Attr("href")
			el = InnerNodesToBox(r+1, b, n)
			el.makeLink(href)
		case "noscript":
			if *ExperimentalJsInsecure || !*EnableNoScriptTag {
				return
			}
			fallthrough
		default:
			// Internal node object
			return InnerNodesToBox(r+1, b, n)
		}
	} else if n.Type() == html.TextNode {
		// Leaf text object

		if text := n.ContentString(); text != "" {
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
	els := make([]*Element, 0, len(n.Children))

	for _, c := range n.Children {
		if c.IsDisplayNone() {
			continue
		}
		if isWrapped(c) {
			ls := NewText(c.Content(), c)
			els = append(els, ls...)
		} else if nodes.IsPureTextContent(*n) {
			// Handle text wrapped in unwrappable tags like p, div, ...
			ls := NewText(c.Content(), c.Children[0])
			if len(ls) == 0 {
				continue
			}
			el := NewElement(horizontalSeq(true, ls), c)
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
	} else if len(els) == 1 {
		return els[0]
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
	case *Scroll:
		traverseTree(r+1, v.Kid.UI, f)
	case *Box:
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
	case *duit.Grid:
		for _, kid := range v.Kids {
			traverseTree(r+1, kid.UI, f)
		}
	case *duit.Image:
	case *duit.Label:
	case *Label:
		traverseTree(r+1, v.Label, f)
	case *Image:
		traverseTree(r+1, v.Image, f)
	case *duit.Field:
	case *duit.Edit:
	case *duit.Button:
	case *duit.List:
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
	case *Box:
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
	case *duit.Grid:
		fmt.Printf("duit.Grid %vx%v\n", len(v.Kids)/v.Columns, v.Columns)
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

type History struct {
	urls []*url.URL
}

func (h History) URL() *url.URL {
	return h.urls[len(h.urls)-1]
}

func (h *History) Push(u *url.URL) {
	if len(h.urls) > 0 && h.urls[len(h.urls)-1].String() == u.String() {
		return
	}
	h.urls = append(h.urls, u)
}

func (h *History) Back() {
	if len(h.urls) > 1 {
		h.urls = h.urls[:len(h.urls)-1]
	}
}

func (h *History) String() string {
	addrs := make([]string, len(h.urls))
	for i, u := range h.urls {
		addrs[i] = u.String()
	}
	return strings.Join(addrs, ", ")
}

type Browser struct {
	History
	dui       *duit.DUI
	Website   *Website
	StatusBar *duit.Label
	LocationField *duit.Field
	loading bool
	client    *http.Client
	Download func(done chan int) chan string
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
		dui: _dui,
		StatusBar: &duit.Label{
			Text: "",
		},
		Website: &Website{},
	}

	b.LocationField = &duit.Field{
		Text:    initUrl,
		Font:    Style.Font(),
		Keys:    func(k rune, m draw.Mouse) (e duit.Event) {
			if k == EnterKey && !b.loading {
				b.loading = true
				a := b.LocationField.Text
				if !strings.HasPrefix(strings.ToLower(a), "http") {
					a = "http://" + a
				}
				u, err := url.Parse(a)
				if err != nil {
					log.Errorf("parse url: %v", err)
					return
				}
				return b.LoadUrl(u)
			}
			return
		},
	}

	u, err := url.Parse(initUrl)
	if err != nil {
		log.Fatalf("parse: %v", err)
	}
	b.History.Push(u)

	buf, _, err := b.get(u, true)
	if err != nil {
		log.Fatalf("get: %v", err)
	}
	b.Website.html = string(buf)

	browser = b
	style.SetFetcher(b)
	dui = _dui
	dui.Background, err = dui.Display.AllocImage(image.Rect(0, 0, 10, 10), draw.ARGB32, true, 0x00000000)
	if err != nil {
		log.Fatalf("%v", err)
	}
	display = dui.Display

	b.Website.layout(b, InitialLayout)

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
	if len(b.History.urls) > 0 && !b.loading {
		b.loading = true
		b.History.Back()
		b.LocationField.Text = b.History.URL().String()
		b.LoadUrl(b.History.URL())
	}
	e.Consumed = true
	return
}

func (b *Browser) SetAndLoadUrl(u *url.URL) func() duit.Event {
	return func() duit.Event {
		if !b.loading {
			b.loading = true
			b.LocationField.Text = u.String()
			b.LoadUrl(u)
		}

		return duit.Event{
			Consumed: true,
		}
	}
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
	go b.loadUrl(url)
	e.Consumed = true

	return
}

func (b *Browser) loadUrl(url *url.URL) {
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
		b.render(buf)
	} else {
		done := make(chan int)
		res := b.Download(done)

		log.Infof("Download unhandled content type: %v", contentType)

		fn := <-res

		if fn != "" {
			log.Infof("Download to %v", fn)
			f, _ := os.Create(fn)
			f.Write(buf)
			f.Close()
		}
		dui.Call <- func() {
			done <- 1
			b.loading = false
		}
	}
}

func (b *Browser) render(buf []byte) {
	log.Printf("Empty cache...")
	cache = make(map[string]struct {
		opossum.ContentType
		buf []byte
	})
	imageCache = make(map[string]*draw.Image)

	b.Website.html = string(buf) // TODO: correctly interpret UTF8
	b.Website.layout(b, InitialLayout)

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
		dui.MarkLayout(dui.Top.UI)
		dui.MarkDraw(dui.Top.UI)
		dui.Render()
		b.loading = false
	}
	log.Printf("Rendering done")
}

func (b *Browser) Get(uri *url.URL) (buf []byte, contentType opossum.ContentType, err error) {
	c, ok := cache[uri.String()]
	if ok {
		log.Printf("use %v from cache", uri)
	} else {
		c.buf, c.ContentType, err = b.get(uri, false)
		if err == nil {
			cache[uri.String()] = c
		}
	}

	return c.buf, c.ContentType, err
}

func (b *Browser) statusBarMsg(msg string, emptyBody bool) {
	if dui == nil || dui.Top.UI == nil {
		return
	}

	dui.Call <- func() {
		if msg == "" {
			b.StatusBar.Text = ""
		} else {
			b.StatusBar.Text += msg + "\n"
		}
		if emptyBody {
			b.Website.UI = &duit.Label{}
		}

		dui.MarkLayout(dui.Top.UI)
		dui.MarkDraw(dui.Top.UI)
		dui.Render()
	}
}

func (b *Browser) get(uri *url.URL, isNewOrigin bool) (buf []byte, contentType opossum.ContentType, err error) {
	msg := fmt.Sprintf("Get %v", uri.String())
	log.Printf(msg)
	b.statusBarMsg(msg, true)
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
	log.Printf("%v\n", resp.Header)
	if err == nil && (contentType.IsHTML() || contentType.IsCSS() || contentType.IsPlain()) {
		buf = contentType.Utf8(buf)
	}
	if isNewOrigin {
		b.History.Push(resp.Request.URL)
		log.Printf("b.History is now %s", b.History.String())
		b.LocationField.Text = b.URL().String()
	}
	return
}

func (b *Browser) PostForm(uri *url.URL, data url.Values) (buf []byte, contentType opossum.ContentType, err error) {
	b.Website.UI = &duit.Label{Text: "Posting..."}
	dui.MarkLayout(dui.Top.UI)
	dui.MarkDraw(dui.Top.UI)
	dui.Render()
	req, err := http.NewRequest("POST", uri.String(), strings.NewReader(data.Encode()))
	if err != nil {
		return
	}
	req.Header.Add("User-Agent", "opossum")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, opossum.ContentType{}, fmt.Errorf("error loading %v: %w", uri, err)
	}
	defer resp.Body.Close()
	b.History.Push(resp.Request.URL)
	buf, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, opossum.ContentType{}, fmt.Errorf("error reading")
	}
	contentType, err = opossum.NewContentType(resp.Header.Get("Content-Type"), resp.Request.URL)
	return
}

