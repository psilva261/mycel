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

	"9fans.net/go/draw"
	"github.com/mjl-/duit"
)

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
	drawOffset int
}

var _ duit.UI = &Scroll{}

// NewScroll returns a full-height scroll bar containing ui.
func NewScroll(ui duit.UI) *Scroll {
	return &Scroll{Height: -1, Kid: duit.Kid{UI: ui}}
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
}

func (ui *Scroll) Draw(dui *duit.DUI, self *duit.Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	debugDraw(dui, self)

	if self.Draw == duit.Clean {
		return
	}
	self.Draw = duit.Clean

	if ui.r.Empty() {
		return
	}

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

	// draw child ui
	if ui.childR.Empty() {
		return
	}
	d := math.Abs(float64(ui.drawOffset - ui.Offset))
	if  d > float64(ui.r.Max.Y) {
		ui.Kid.Draw = duit.Dirty
	}
	if ui.img == nil || ui.drawRect().Size() != ui.img.R.Size() || ui.Kid.Draw == duit.Dirty {
		var err error
		if ui.img != nil {
			ui.img.Free()
			ui.img = nil
		}
		ui.Kid.Draw = duit.Dirty
		if ui.Kid.R.Dx() == 0 || ui.Kid.R.Dy() == 0 {
			return
		}
		ui.img, err = dui.Display.AllocImage(ui.drawRect(), draw.ARGB32, false, dui.BackgroundColor)
		if duitError(dui, err, "allocimage") {
			return
		}
		ui.drawOffset = ui.Offset
	} else if ui.Kid.Draw == duit.Dirty {
		ui.img.Draw(ui.img.R, dui.Background, nil, image.ZP)
	}
	m.Point = m.Point.Add(image.Pt(-ui.childR.Min.X, ui.Offset))
	if ui.Kid.Draw != duit.Clean {
		if force {
			ui.Kid.Draw = duit.Dirty
		}
		ui.Kid.UI.Draw(dui, &ui.Kid, ui.img, image.ZP, m, ui.Kid.Draw == duit.Dirty)
		ui.Kid.Draw = duit.Clean
	}
	img.Draw(ui.childR.Add(orig), ui.img, nil, image.Pt(0, ui.Offset))
}

// Allocate only an image buffer of view size ui.r
// - which is translated by scroll offset ui.Offset - instead
// of whole Kid view size ui.Kid.R which leads to much
// faster render times for large pages. Add same size rectangles
// above/below to decrease flickering.
func (ui *Scroll) drawRect() image.Rectangle {
	if 2*ui.r.Dy() > ui.Kid.R.Dy() {
		return ui.Kid.R
	}
	r := image.Rectangle{
		Min: ui.r.Min,
		Max: image.Point{
			ui.r.Max.X,
			3*ui.r.Max.Y,
		},
	}
	r = r.Add(image.Point{X:0, Y:ui.Offset-ui.r.Max.Y})
	if r.Min.Y > ui.Offset {
		r.Min.Y -= ui.Offset
	}
	return r
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
	} else if ui.Kid.Draw != duit.Clean || scrolled {
		self.Draw = duit.Dirty
	}
}

func (ui *Scroll) Mouse(dui *duit.DUI, self *duit.Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r duit.Result) {
	nOrigM := origM
	nOrigM.Point = nOrigM.Point.Add(image.Pt(-ui.scrollbarSize, ui.Offset))
	nm := m
	nm.Point = nm.Point.Add(image.Pt(-ui.scrollbarSize, ui.Offset))

	if m.Buttons == 0 {
		ui.Kid.UI.Mouse(dui, &ui.Kid, nm, nOrigM, image.ZP)
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
		m.Point = m.Point.Add(image.Pt(-ui.scrollbarSize, ui.Offset))
		r = ui.Kid.UI.Key(dui, &ui.Kid, k, m, image.ZP)
		ui.warpScroll(dui, self, r.Warp, orig)
		scrolled := false
		if !r.Consumed {
			scrolled = ui.scrollKey(k)
			r.Consumed = scrolled
		}
		ui.result(dui, self, &r, scrolled)
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
