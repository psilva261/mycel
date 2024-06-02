package duitx

// Original code from github.com/mjl-/duit
//
// Copyright 2018 Mechiel Lukkien mechiel@ueber.net
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of this
// software and associated documentation files (the "Software"), to deal in the Software
// without restriction, including without limitation the rights to use, copy, modify, merge,
// publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons
// to whom the Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all copies or
// substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED,
// INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A
// PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
// OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
// SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

import (
	"fmt"
	"image"
	"math"
	"time"

	"9fans.net/go/draw"
	"github.com/mjl-/duit"
	"github.com/psilva261/mycel/logger"
)

const maxAge = time.Minute

// Scroll shows a part of its single child, typically a box, and lets you scroll the content.
type Scroll struct {
	Kid    duit.Kid
	Height int // < 0 means full height, 0 means as much as necessary, >0 means exactly that many lowdpi pixels

	r             image.Rectangle // entire ui
	barR          image.Rectangle
	barActiveR    image.Rectangle
	childR        image.Rectangle
	Offset        int         // current scroll offset in pixels
	img           *draw.Image // for child to draw on
	scrollbarSize int
	lastMouseUI   duit.UI
	drawOffset    int

	tiles        map[int]*draw.Image
	last         map[int]time.Time
	tilesChanged bool
}

var _ duit.UI = &Scroll{}

// NewScroll returns a full-height scroll bar containing ui.
func NewScroll(dui *duit.DUI, ui duit.UI) *Scroll {
	s := &Scroll{
		Height: -1,
		Kid:    duit.Kid{UI: ui},
		tiles:  make(map[int]*draw.Image),
		last:   make(map[int]time.Time),
	}
	return s
}

func (ui *Scroll) Free() {
	ui.tiles = make(map[int]*draw.Image)
	ui.last = make(map[int]time.Time)
}

func (ui *Scroll) freeCur() {
	i, of := ui.pos()
	tl, ok := ui.tiles[i]
	tl1, ok1 := ui.tiles[i+1]
	if !ui.tilesChanged && (!ok || ui.sizeOk(tl)) && (of == 0 || !ok1 || ui.sizeOk(tl1)) {
		return
	}
	if ui.tiles[i] != nil {
		ui.tiles[i].Free()
		delete(ui.tiles, i)
		delete(ui.last, i)
	}
	if of > 0 {
		if ui.tiles[i+1] != nil {
			ui.tiles[i+1].Free()
			delete(ui.tiles, i+1)
			delete(ui.last, i+1)
		}
	}
	ui.tilesChanged = false
}

func (ui *Scroll) sizeOk(tl *draw.Image) bool {
	return tl != nil && tl.R.Dx() == ui.r.Dx() && tl.R.Dy() == ui.r.Dy()
}

func (ui *Scroll) ensure(dui *duit.DUI, i int) {
	log.Printf("ensure(dui, %v)", i)
	last, ok := ui.last[i]
	tl, _ := ui.tiles[i]
	if ok && time.Since(last) < maxAge && ui.sizeOk(tl) {
		return
	}

	log.Printf("ensure(dui, %v): draw", i)
	r := ui.r.Add(image.Point{X: 0, Y: i * ui.r.Dy()})
	img, err := dui.Display.AllocImage(r, draw.ARGB32, false, dui.BackgroundColor)
	if duitError(dui, err, "allocimage") {
		return
	}
	ui.Kid.UI.Draw(dui, &ui.Kid, img, image.ZP, draw.Mouse{}, true)

	if ui.tiles[i] != nil {
		ui.tiles[i].Free()
		ui.tiles[i] = nil
	}
	log.Printf("ensure: ui.tiles[%d] = img(R=%+v, ...)", i, img.R)
	ui.tiles[i] = img
	ui.last[i] = time.Now()

	for j, t := range ui.tiles {
		if math.Abs(float64(i-j)) > 5 {
			t.Free()
			delete(ui.tiles, j)
			delete(ui.last, j)
		}
	}
}

func (ui *Scroll) pos() (t, of int) {
	t = ui.Offset / ui.r.Dy()
	of = ui.Offset % ui.r.Dy()
	return
}

func (ui *Scroll) tlR(i int) (r image.Rectangle) {
	r.Min.X = ui.r.Min.X
	r.Max.X = ui.r.Max.X
	r.Min.Y = ui.r.Min.Y + i*ui.r.Dy()
	r.Max.Y = r.Min.Y + ui.r.Dy()
	return
}

func (ui *Scroll) Layout(dui *duit.DUI, self *duit.Kid, sizeAvail image.Point, force bool) {
	debugLayout(dui, self)

	if self.Layout == duit.Clean && !force {
		return
	}
	self.Layout = duit.Clean
	self.Draw = duit.Dirty
	// todo: be smarter about DirtyKid

	ui.scrollbarSize = dui.Scale(duit.ScrollbarSize)
	scaledHeight := dui.Scale(ui.Height)
	if scaledHeight > 0 && scaledHeight < sizeAvail.Y {
		sizeAvail.Y = scaledHeight
	}
	ui.r = rect(sizeAvail)
	ui.barR = ui.r
	ui.barR.Max.X = ui.barR.Min.X + ui.scrollbarSize
	ui.childR = ui.r
	ui.childR.Min.X = ui.barR.Max.X

	// todo: only force when sizeAvail or childR changed?
	ui.Kid.UI.Layout(dui, &ui.Kid, image.Pt(ui.r.Dx()-ui.barR.Dx(), ui.r.Dy()), force)
	ui.Kid.Layout = duit.Clean
	ui.Kid.Draw = duit.Dirty

	kY := ui.Kid.R.Dy()
	if ui.r.Dy() > kY && ui.Height == 0 {
		ui.barR.Max.Y = kY
		ui.r.Max.Y = kY
		ui.childR.Max.Y = kY
	}
	self.R = rect(ui.r.Size())
	ui.Free()
}

func (ui *Scroll) Draw(dui *duit.DUI, self *duit.Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	debugDraw(dui, self)

	if self.Draw == duit.Clean {
		return
	} else {
		log.Printf("Draw: self.Draw=%v is not clean, force=%v", self.Draw, force)
	}

	if ui.r.Empty() {
		self.Draw = duit.Clean
		return
	}

	ui.drawBar(dui, self, img, orig, m, force)
	ui.drawChild(dui, self, img, orig, m, force)
	self.Draw = duit.Clean
}

func (ui *Scroll) drawBar(dui *duit.DUI, self *duit.Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	// ui.scroll(0)
	barHover := m.In(ui.barR)

	bg := dui.ScrollBGNormal
	vis := dui.ScrollVisibleNormal
	if barHover {
		bg = dui.ScrollBGHover
		vis = dui.ScrollVisibleHover
	}

	h := ui.r.Dy()
	uih := ui.Kid.R.Dy()
	if uih > h {
		barR := ui.barR.Add(orig)
		img.Draw(barR, bg, nil, image.ZP)
		barH := h * h / uih
		barY := ui.Offset * h / uih
		ui.barActiveR = ui.barR
		ui.barActiveR.Min.Y += barY
		ui.barActiveR.Max.Y = ui.barActiveR.Min.Y + barH
		barActiveR := ui.barActiveR.Add(orig)
		barActiveR.Max.X -= 1 // unscaled
		img.Draw(barActiveR, vis, nil, image.ZP)
	}
}

func (ui *Scroll) drawChild(dui *duit.DUI, self *duit.Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	// draw child ui
	if ui.childR.Empty() {
		return
	}

	var i, of int
	var tl, tl1 *draw.Image
	var ok, ok1, ok2, ok3, okm1, okm2 bool
	p := draw.Point{X: 0, Y: ui.Offset}
	n := draw.Point{X: 0, Y: -ui.Offset}

	predrawCur := func() {
		// tile draw
		i, of = ui.pos()
		tl, ok = ui.tiles[i]
		tl1, ok1 = ui.tiles[i+1]
		if !ok {
			ui.ensure(dui, i)
		}
		if !ok1 {
			ui.ensure(dui, i+1)
		}
		if !ok {
			tl, _ = ui.tiles[i]
		}
		if !ok1 && of > 0 {
			tl1, _ = ui.tiles[i+1]
		}
	}

	predrawFut := func() {
		// tile draw
		i, of = ui.pos()
		tl1, ok1 = ui.tiles[i+1]
		_, ok2 = ui.tiles[i+2]
		_, ok3 = ui.tiles[i+3]
		if i > 0 {
			_, okm1 = ui.tiles[i-1]
		}
		if i > 1 {
			_, okm2 = ui.tiles[i-2]
		}
		if of == 0 && !ok1 {
			ui.ensure(dui, i+1)
		}
		if ok1 && !ok2 {
			ui.ensure(dui, i+2)
		}
		if ok2 && !ok3 {
			ui.ensure(dui, i+3)
		}
		if i > 0 && !okm1 {
			ui.ensure(dui, i-1)
		}
		if i > 1 && okm1 && !okm2 {
			ui.ensure(dui, i-2)
		}
	}
	defer predrawFut()

	if self.Draw == duit.DirtyKid {
		ui.freeCur()
		ui.Kid.Draw = duit.Clean
	} else if ui.Kid.Draw != duit.Clean || force {
		log.Printf("drawChild: refresh: ui.Kid.Draw=%v  force=%v", ui.Kid.Draw, force)
		ui.freeCur()
		tmp := img.Clipr
		img.ReplClipr(false, ui.childR.Add(orig))
		ui.Kid.UI.Draw(dui, &ui.Kid, img, orig.Add(ui.childR.Min).Add(n), draw.Mouse{}, true)
		img.ReplClipr(false, tmp)
		ui.Kid.Draw = duit.Clean
		return
	}

	predrawCur()

	rTop := draw.Rectangle{
		Min: ui.childR.Min,
		Max: draw.Point{
			X: ui.childR.Max.X,
			Y: ui.childR.Max.Y - of,
		},
	}
	rBtm := draw.Rectangle{
		Min: draw.Point{
			X: ui.childR.Min.X,
			Y: rTop.Max.Y,
		},
		Max: ui.childR.Max,
	}
	pOf := draw.Point{X: 0, Y: ui.Offset + rTop.Dy()}
	img.Draw(rTop.Add(orig), tl, nil, p)
	if of > 0 {
		img.Draw(rBtm.Add(orig), tl1, nil, pOf)
	}

	return
}

func (ui *Scroll) scroll(delta int) (changed bool) {
	o := ui.Offset
	ui.Offset += delta
	ui.Offset = maximum(0, ui.Offset)
	ui.Offset = minimum(ui.Offset, maximum(0, ui.Kid.R.Dy()-ui.childR.Dy()))
	return o != ui.Offset
}

func (ui *Scroll) scrollKey(k rune) (consumed bool) {
	switch k {
	case draw.KeyUp:
		return ui.scroll(-50)
	case draw.KeyDown:
		return ui.scroll(50)
	case draw.KeyPageUp:
		return ui.scroll(-200)
	case draw.KeyPageDown:
		return ui.scroll(200)
	}
	return false
}

func (ui *Scroll) scrollMouse(m draw.Mouse, scrollOnly bool) (consumed bool) {
	switch m.Buttons {
	case duit.Button4:
		return ui.scroll(-m.Y / 4)
	case duit.Button5:
		return ui.scroll(m.Y / 4)
	}

	if scrollOnly {
		return false
	}
	switch m.Buttons {
	case duit.Button1:
		return ui.scroll(-m.Y)
	case duit.Button2:
		Offset := m.Y * ui.Kid.R.Dy() / ui.barR.Dy()
		OffsetMax := ui.Kid.R.Dy() - ui.childR.Dy()
		Offset = maximum(0, minimum(Offset, OffsetMax))
		o := ui.Offset
		ui.Offset = Offset
		return o != ui.Offset
	case duit.Button3:
		return ui.scroll(m.Y)
	}
	return false
}

func (ui *Scroll) result(dui *duit.DUI, self *duit.Kid, r *duit.Result, scrolled bool) {
	if ui.Kid.Layout != duit.Clean {
		ui.Kid.UI.Layout(dui, &ui.Kid, ui.childR.Size(), false)
		ui.Kid.Layout = duit.Clean
		ui.Kid.Draw = duit.Dirty
		self.Draw = duit.Dirty
		if r.Consumed && !scrolled {
			ui.tilesChanged = true
		}
	} else if ui.Kid.Draw != duit.Clean || scrolled {
		self.Draw = duit.Dirty
		if r.Consumed && !scrolled {
			ui.tilesChanged = true
		}
	}
}

func (ui *Scroll) Mouse(dui *duit.DUI, self *duit.Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r duit.Result) {
	nOrigM := origM
	nOrigM.Point = nOrigM.Point.Add(image.Pt(-ui.scrollbarSize, ui.Offset))
	nm := m
	nm.Point = nm.Point.Add(image.Pt(-ui.scrollbarSize, ui.Offset))

	if m.Buttons == 0 {
		ui.Kid.UI.Mouse(dui, &ui.Kid, nm, nOrigM, image.ZP) // comment this to have no flicker after mouse move and then scroll
		return
	}
	if m.Point.In(ui.barR) {
		r.Hit = ui
		r.Consumed = ui.scrollMouse(m, false)
		self.Draw = duit.Dirty
		return
	} else if m.Point.In(ui.childR) {
		r.Consumed = ui.scrollMouse(m, true)
		if r.Consumed {
			self.Draw = duit.Dirty
			return
		}
		r = ui.Kid.UI.Mouse(dui, &ui.Kid, nm, nOrigM, image.ZP)
		if r.Consumed {
			self.Draw = duit.Dirty
			ui.tilesChanged = true
			log.Printf("Mouse: set ui.tilesChanged = true")
		}
	}
	return
}

func (ui *Scroll) Key(dui *duit.DUI, self *duit.Kid, k rune, m draw.Mouse, orig image.Point) (r duit.Result) {
	if m.Point.In(ui.barR) {
		r.Hit = ui
		r.Consumed = ui.scrollKey(k)
		if r.Consumed {
			self.Draw = duit.Dirty
		}
	}
	if m.Point.In(ui.childR) {
		log.Printf("Key: in ui.childR (self.Draw=%v)", self.Draw)
		m.Point = m.Point.Add(image.Pt(-ui.scrollbarSize, ui.Offset))
		scrolled := ui.scrollKey(k)
		if scrolled {
			self.Draw = duit.Dirty
			r.Consumed = scrolled
			return
		}
		r = ui.Kid.UI.Key(dui, &ui.Kid, k, m, image.ZP)
		ui.warpScroll(dui, self, r.Warp, orig)
		ui.result(dui, self, &r, scrolled)
		log.Printf("Key: in ui.childR (self.Draw'=%v)", self.Draw)
	}
	return
}

func (ui *Scroll) warpScroll(dui *duit.DUI, self *duit.Kid, warp *image.Point, orig image.Point) {
	if warp == nil {
		return
	}

	Offset := ui.Offset
	if warp.Y < ui.Offset {
		ui.Offset = maximum(0, warp.Y-dui.Scale(40))
	} else if warp.Y > ui.Offset+ui.r.Dy() {
		ui.Offset = minimum(ui.Kid.R.Dy()-ui.r.Dy(), warp.Y+dui.Scale(40)-ui.r.Dy())
	}
	if Offset != ui.Offset {
		if self != nil {
			self.Draw = duit.Dirty
		} else {
			dui.MarkDraw(ui)
		}
	}
	warp.Y -= ui.Offset
	warp.X += orig.X + ui.scrollbarSize
	warp.Y += orig.Y
}

func (ui *Scroll) _focus(dui *duit.DUI, p *image.Point) *image.Point {
	if p == nil {
		return nil
	}
	pp := p.Add(ui.childR.Min)
	p = &pp
	ui.warpScroll(dui, nil, p, image.ZP)
	return p
}

func (ui *Scroll) FirstFocus(dui *duit.DUI, self *duit.Kid) *image.Point {
	p := ui.Kid.UI.FirstFocus(dui, &ui.Kid)
	return ui._focus(dui, p)
}

func (ui *Scroll) Focus(dui *duit.DUI, self *duit.Kid, o duit.UI) *image.Point {
	if o == ui {
		p := image.Pt(minimum(ui.scrollbarSize/2, ui.r.Dx()), minimum(ui.scrollbarSize/2, ui.r.Dy()))
		return &p
	}
	p := ui.Kid.UI.Focus(dui, &ui.Kid, o)
	return ui._focus(dui, p)
}

func (ui *Scroll) Mark(self *duit.Kid, o duit.UI, forLayout bool) (marked bool) {
	if self.Mark(o, forLayout) {
		return true
	}
	marked = ui.Kid.UI.Mark(&ui.Kid, o, forLayout)
	if marked {
		if forLayout {
			if self.Layout == duit.Clean {
				self.Layout = duit.DirtyKid
			}
		} else {
			if self.Layout == duit.Clean {
				self.Draw = duit.DirtyKid
			}
		}
	}
	return
}

func (ui *Scroll) Print(self *duit.Kid, indent int) {
	what := fmt.Sprintf("Scroll Offset=%d childR=%v", ui.Offset, ui.childR)
	duit.PrintUI(what, self, indent)
	ui.Kid.UI.Print(&ui.Kid, indent+1)
}

//////////////////////
//                  //
// helper functions //
//                  //
//////////////////////

func pt(v int) image.Point {
	return image.Point{v, v}
}

func rect(p image.Point) image.Rectangle {
	return image.Rectangle{image.ZP, p}
}

func extendY(r image.Rectangle, dy int) image.Rectangle {
	r.Max.Y += dy
	return r
}

func insetPt(r image.Rectangle, pad image.Point) image.Rectangle {
	r.Min = r.Min.Add(pad)
	r.Max = r.Max.Sub(pad)
	return r
}

func outsetPt(r image.Rectangle, pad image.Point) image.Rectangle {
	r.Min = r.Min.Sub(pad)
	r.Max = r.Max.Add(pad)
	return r
}

func minimum64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maximum64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func minimum(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maximum(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func debugLayout(d *duit.DUI, self *duit.Kid) {
	if d.DebugLayout > 0 {
		log.Printf("duit: Layout %T %s layout=%d draw=%d\n", self.UI, self.R, self.Layout, self.Draw)
	}
}

func debugDraw(d *duit.DUI, self *duit.Kid) {
	if d.DebugDraw > 0 {
		log.Printf("duit: Draw %T %s layout=%d draw=%d\n", self.UI, self.R, self.Layout, self.Draw)
	}
}

func duitError(d *duit.DUI, err error, msg string) bool {
	if err == nil {
		return false
	}
	go func() {
		d.Error <- fmt.Errorf("%s: %s", msg, err)
	}()
	return true
}
