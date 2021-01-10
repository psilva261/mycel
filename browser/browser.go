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
	"strings"

	"github.com/mjl-/duit"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

const debugPrintHtml = false
const stashElements = true
const experimentalUseSlicedDrawing = false

const EnterKey = 10

var cursor = [2 * 16]uint8{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 90, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}

var DebugDumpCSS *bool
var ExperimentalJsInsecure *bool
var EnableNoScriptTag *bool

var browser *Browser // TODO: limit global objects;
//       at least put them in separate pkgs
//       with good choiced private/public
var Style = style.Map{}
var dui *duit.DUI
var colorCache = make(map[draw.Color]*draw.Image)
var imageCache = make(map[string]*draw.Image)
var cache = make(map[string]struct {
	opossum.ContentType
	buf []byte
})
var log *logger.Logger
var scroller *duit.Scroll
var display *draw.Display

func SetLogger(l *logger.Logger) {
	log = l
}

type ColoredLabel struct {
	*duit.Label

	n *nodes.Node
}

func (ui *ColoredLabel) Draw(dui *duit.DUI, self *duit.Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	// TODO: hacky function, might lead to crashes and memory leaks
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
	/*edit.Keys = func(k rune, m draw.Mouse) (e duit.Event) {
		//log.Printf("k=%v (c %v    p %v)", k, unicode.IsControl(k), unicode.IsPrint(k))
		if unicode.IsPrint(k) {
			e.Consumed = true
		}
		return
	}*/
	formatted := ""
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		formatted += strings.TrimSpace(line) + "\n"
	}
	log.Printf("formatted=%+v", formatted)
	edit.Append([]byte(formatted))
	cv.UI = &duit.Box{
		Kids:   duit.NewKids(edit),
		Height: (int(n.FontSize()) + 4) * (len(lines)+2),
	}
	return
}

func (cv *CodeView) Mouse(dui *duit.DUI, self *duit.Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r duit.Result) {
	//log.Printf("m=%+v",m.Buttons)
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

	if display == nil {
		// probably called from a unit test
		return nil, fmt.Errorf("display nil")
	}

	if n.Data() == "svg" {
		xml, err := n.Serialized()
		if  err != nil {
			return nil, fmt.Errorf("serialize: %w", err)
		}
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

type Element struct {
	duit.UI
	n       *nodes.Node
	IsLink bool
	Click  func() duit.Event
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

	if stashElements {
		existingEl, ok := ui.(*Element)
		if ok && existingEl != nil {
			return &Element{
				UI: existingEl.UI,
				n: existingEl.n,
			}
		}
	}
	return &Element{
		UI: ui,
		n: n,
	}
}

func NewBoxElement(ui duit.UI, n *nodes.Node) *Element {
	if ui == nil {
		return nil
	}
	if n.IsDisplayNone() {
		return nil
	}

	var i *draw.Image
	w := n.Width()
	h := n.Height()

	if w == 0 && h == 0 {
		return NewElement(ui, n)
	}
	if bg, err := n.BoxBackground(); err == nil {
		_=bg
		//i = bg
	} else {
		log.Printf("box background: %f", err)
	}
	box := &duit.Box{
		Kids:       duit.NewKids(ui),
		Width:      w,
		Height:     h,
		Background: i,
	}
	el := NewElement(box, n)
	return el
}

func (el *Element) Draw(dui *duit.DUI, self *duit.Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	if el == nil {
		return
	}
	if el.slicedDraw(dui, self, img, orig, m, force) {
		return
	}
	box, ok := el.UI.(*duit.Box)
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
	box, ok := el.UI.(*duit.Box)
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
	} else if c := strings.TrimSpace(nodes.ContentFrom(*n)); c != "" {
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

		b.submit(f.DomSubtree, n.DomSubtree)

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
	return NewElement(
		&duit.Box{
			Kids: duit.NewKids(&duit.Field{
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
						browser.submit(n.Ancestor("form").DomSubtree, nil)
						return duit.Event{
							Consumed:   true,
							NeedLayout: true,
							NeedDraw:   true,
						}
					}
					return
				},
			}),
			MaxWidth: 200,
		},
		n,
	)
}

func (el *Element) Mouse(dui *duit.DUI, self *duit.Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r duit.Result) {
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

func (el *Element) click() (consumed bool) {
	if el.Click != nil {
		el.Click()
		return
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

	browser.Website.html = res
	browser.Website.layout(browser, ClickRelayout)
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
		return NewElement(horizontalSeq(true, rows[0]), n)
	} else {
		seqs := make([]*Element, 0, len(rows))
		for _, row := range rows {
			seq := horizontalSeq(true, row)
			seqs = append(seqs, NewElement(seq, n))
		}
		return NewElement(verticalSeq(seqs), n)
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

		return &duit.Box{
			Padding: duit.SpaceXY(6, 4),
			Margin:  image.Pt(6, 4),
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

func grepBody(n *html.Node) *html.Node {
	var body *html.Node

	if n.Type == html.ElementNode {
		if n.Data == "body" {
			return n
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		res := grepBody(c)
		if res != nil {
			body = res
		}
	}

	return body
}

func NodeToBox(r int, b *Browser, n *nodes.Node) *Element {
	if attr(*n.DomSubtree, "aria-hidden") == "true" || hasAttr(*n.DomSubtree, "hidden") {
		return nil
	}

	if n.IsDisplayNone() {
		return nil
	}

	if n.Type() == html.ElementNode {
		switch n.Data() {
		case "style", "script", "template":
			return nil
		case "input":
			t := attr(*n.DomSubtree, "type")
			if isPw := t == "password"; t == "text" || t == "" || t == "search" || isPw {
				return NewInputField(n)
			} else if t == "submit" {
				return NewSubmitButton(b, n)
			} else {
				return nil
			}
		case "button":
			if t := attr(*n.DomSubtree, "type"); t == "" || t == "submit" {
				return NewSubmitButton(b, n)
			} else {
				btn := &duit.Button{
					Text: nodes.ContentFrom(*n),
					Font: n.Font(),
				}
				return NewElement(
					btn,
					n,
				)
			}
		case "table":
			return NewTable(n).Element(r+1, b, n)
		case "noscript":
			if *ExperimentalJsInsecure || !*EnableNoScriptTag {
				return nil
			}
			fallthrough
		case "body", "p", "h1", "center", "nav", "article", "header", "div", "td":
			var innerContent duit.UI
			if nodes.IsPureTextContent(*n) {
				t := strings.TrimSpace(nodes.ContentFrom(*n))
				innerContent = &ColoredLabel{
					Label: &duit.Label{
						Text: t,
						Font: n.Font(),
					},
					n: n,
				}
			} else {
				innerContent = InnerNodesToBox(r+1, b, n)
			}

			return NewBoxElement(
				innerContent,
				n,
			)
		case "img", "svg":
			return NewElement(
				NewImage(n),
				n,
			)
		case "pre":
			return NewElement(
				NewCodeView(nodes.ContentFrom(*n), n.Map),
				n,
			)
		case "li":
			var innerContent duit.UI
			if nodes.IsPureTextContent(*n) {
				t := nodes.ContentFrom(*n)

				if ul := n.Ancestor("ul"); ul != nil {
					if ul.Css("list-style") != "none" && n.Css("list-style-type") != "none" {
						t = "• " + t
					}
				}
				innerContent = &ColoredLabel{
					Label: &duit.Label{
						Text: t,
						Font: n.Font(),
					},
					n: n,
				}
			} else {
				innerContent = InnerNodesToBox(r+1, b, n)
			}

			return NewElement(
				innerContent,
				n,
			)
		case "a":
			var href = n.Attr("href")
			var innerContent duit.UI

			if nodes.IsPureTextContent(*n) {
				innerContent = &ColoredLabel{
					Label: &duit.Label{
						Text:  nodes.ContentFrom(*n),
						Font:  n.Font(),
					},
					n: n,
				}
			} else {
				innerContent = InnerNodesToBox(r+1, b, n)
			}

			if innerContent == nil {
				return nil
			}

			el := NewElement(
				innerContent,
				n,
			)
			el.makeLink(href)
			return el
		default:
			// Internal node object
			return InnerNodesToBox(r+1, b, n)
		}
	} else if n.Type() == html.TextNode {
		// Leaf text object

		if text := strings.TrimSpace(nodes.ContentFrom(*n)); text != "" {
			text = strings.ReplaceAll(text, "\n", "")
			text = strings.ReplaceAll(text, "\t", "")
			l := strings.Split(text, " ")
			nn := make([]string, 0, len(l))

			for _, w := range l {
				if w != "" {
					nn = append(nn, w)
				}
			}

			text = strings.Join(nn, " ")
			ui := &duit.Label{
				Text: text,
				Font: n.Font(),
			}

			return NewElement(
				ui,
				n,
			)
		} else {
			return nil
		}
	} else {
		return nil
	}
}

func InnerNodesToBox(r int, b *Browser, n *nodes.Node) *Element {
	els := make([]*Element, 0, len(n.Children))

	for _, c := range n.Children {
		if el := NodeToBox(r+1, b, c); el != nil && !c.IsDisplayNone() {
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
		return
	}
	f(ui)
	switch v := ui.(type) {
	case nil:
		panic("null")
	case *duit.Scroll:
		traverseTree(r+1, v.Kid.UI, f)
	case *duit.Box:
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
	case *ColoredLabel:
		traverseTree(r+1, v.Label, f)
	case *duit.Button:
	case *Image:
		traverseTree(r+1, v.Image, f)
	case *duit.Field:
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
	case *duit.Box:
		fmt.Printf("duit.Box\n")
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
	case *ColoredLabel:
		t := v.Text
		if len(t) > 20 {
			t = t[:15] + "..."
		}
		fmt.Printf("ColoredLabel %v\n", t)
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
			if k == EnterKey {
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

	buf, _, err := b.Get(u)
	if err != nil {
		log.Fatalf("get: %v", err)
	}
	b.Website.html = string(buf)

	browser = b
	style.SetFetcher(b)
	dui = _dui
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
			log.Printf("A")
			addr = "/" + b.URL().Path + "/" + addr
		} else {
			log.Printf("B")
			m := strings.LastIndex(b.URL().Path, "/")
			if m > 0 {
				log.Printf("B.>")
				folder := b.URL().Path[0:m]
				addr = "/" + folder + "/" + addr
			} else {
				log.Printf("B.<=")
				addr = "/" + addr
			}
		}
		addr = strings.ReplaceAll(addr, "//", "/")
		addr = b.URL().Scheme + "://" + b.URL().Host + addr
	}
	return url.Parse(addr)
}

func (b *Browser) Back() (e duit.Event) {
	if len(b.History.urls) > 0 {
		b.History.Back()
		b.LocationField.Text = b.History.URL().String()
		b.LoadUrl(b.History.URL())
	}
	e.Consumed = true
	return
}

func (b *Browser) SetAndLoadUrl(u *url.URL) func() duit.Event {
	return func() duit.Event {
		b.LocationField.Text = u.String()
		b.LoadUrl(u)

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

func (b *Browser) LoadUrl(url *url.URL) (e duit.Event) {
	buf, contentType, err := b.get(url, true)
	if err != nil {
		log.Errorf("error loading %v: %v", url, err)
		if er := errors.Unwrap(err); er != nil {
			err = er
		}
		b.showBodyMessage(err.Error())
		return
	}
	if contentType.IsHTML() || contentType.IsPlain() || contentType.IsEmpty() {
		b.render(buf)
	} else {
		done := make(chan int)
		res := b.Download(done)

		log.Infof("Download unhandled content type: %v", contentType)

		go func() {
			fn := <-res

			if fn != "" {
				log.Infof("Download to %v", fn)
				f, _ := os.Create(fn)
				f.Write(buf)
				f.Close()
			}

			done <- 1
		}()
	}
	return duit.Event{
		Consumed:   true,
		NeedLayout: true,
		NeedDraw:   true,
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

	dui.MarkLayout(dui.Top.UI)
	dui.MarkDraw(dui.Top.UI)
	TraverseTree(b.Website.UI, func(ui duit.UI) {
		// just checking for nil elements. That would be a bug anyway and it's better
		// to notice it before it gets rendered

		if ui == nil {
			panic("nil")
		}
	})
	PrintTree(b.Website.UI)
	log.Printf("Render...")
	dui.Render()
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
	b.StatusBar.Text = msg
	if emptyBody {
		b.Website.UI = &duit.Label{}
	}
	dui.MarkLayout(dui.Top.UI)
	dui.MarkDraw(dui.Top.UI)
	dui.Render()
}

func (b *Browser) get(uri *url.URL, isNewOrigin bool) (buf []byte, contentType opossum.ContentType, err error) {
	msg := fmt.Sprintf("Get %v", uri.String())
	log.Printf(msg)
	b.statusBarMsg(msg, true)
	defer func() {
		b.statusBarMsg("", true)
	}()
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

