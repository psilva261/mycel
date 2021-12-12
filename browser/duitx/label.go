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

var (
	selectedBg *draw.Image
)

// Label draws multiline text in a single font.:
//
// Keys:
//	cmd-c, copy text
//	\n, like button1 click, calls the Click function
type Label struct {
	Text     string                // Text to draw, wrapped at glyph boundary.
	Font     *draw.Font            `json:"-"` // For drawing text.
	Click    func() (e duit.Event) `json:"-"` // Called on button1 click.
	Selected bool

	lines []string
	orig  image.Point
	size  image.Point
	m     draw.Mouse
}

var _ duit.UI = &Label{}

func (ui *Label) font(dui *duit.DUI) *draw.Font {
	return dui.Font(ui.Font)
}

func (ui *Label) Layout(dui *duit.DUI, self *duit.Kid, sizeAvail image.Point, force bool) {
	debugLayout(dui, self)

	font := ui.font(dui)
	ui.lines = []string{}
	s := 0
	x := 0
	xmax := 0
	for i, c := range ui.Text {
		if c == '\n' {
			xmax = maximum(xmax, x)
			ui.lines = append(ui.lines, ui.Text[s:i])
			s = i + 1
			x = 0
			continue
		}
		dx := font.StringWidth(string(c))
		x += dx
		if i-s == 0 || x <= sizeAvail.X {
			continue
		}
		xmax = maximum(xmax, x-dx)
		ui.lines = append(ui.lines, ui.Text[s:i])
		s = i
		x = dx
	}
	if s < len(ui.Text) || s == 0 {
		ui.lines = append(ui.lines, ui.Text[s:])
		xmax = maximum(xmax, x)
	}
	ui.size = image.Pt(xmax, len(ui.lines)*ui.lineHeight(font))
	self.R = rect(ui.size)
}

func (ui *Label) lineHeight(font *draw.Font) int {
	return int(math.Ceil(float64(font.Height) * 1.2))
}

func (ui *Label) Draw(dui *duit.DUI, self *duit.Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	debugDraw(dui, self)

	if selectedBg == nil {
		var err error
		selectedBg, err = dui.Display.AllocImage(image.Rect(0, 0, 10, 10), draw.ARGB32, true, 0x9acd32ff)
		if err != nil {
			panic(fmt.Errorf("%v", err))
		}
	}

	p := orig
	font := ui.font(dui)
	for _, line := range ui.lines {
		if ui.Selected {
			img.StringBg(p, dui.Regular.Normal.Text, image.ZP, font, line, selectedBg, image.ZP)
		} else {
			img.String(p, dui.Regular.Normal.Text, image.ZP, font, line)
		}
		p.Y += ui.lineHeight(font)
	}
	ui.orig = orig
}

func (ui *Label) Mouse(dui *duit.DUI, self *duit.Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r duit.Result) {
	if m.In(rect(ui.size)) && ui.m.Buttons == 0 && m.Buttons == duit.Button1 && ui.Click != nil {
		e := ui.Click()
		propagateEvent(self, &r, e)
	}
	ui.m = m
	return
}

func (ui *Label) Key(dui *duit.DUI, self *duit.Kid, k rune, m draw.Mouse, orig image.Point) (r duit.Result) {
	switch k {
	case '\n':
		if ui.Click != nil {
			e := ui.Click()
			propagateEvent(self, &r, e)
		}
	case draw.KeyCmd + 'c':
		dui.WriteSnarf([]byte(ui.Text))
		r.Consumed = true
	}
	return
}

func (ui *Label) FirstFocus(dui *duit.DUI, self *duit.Kid) *image.Point {
	return nil
}

func (ui *Label) Focus(dui *duit.DUI, self *duit.Kid, o duit.UI) *image.Point {
	if ui != o {
		return nil
	}
	return &image.ZP
}

func (ui *Label) Mark(self *duit.Kid, o duit.UI, forLayout bool) (marked bool) {
	return self.Mark(o, forLayout)
}

func (ui *Label) Print(self *duit.Kid, indent int) {
	duit.PrintUI("Label", self, indent)
}

func propagateEvent(self *duit.Kid, r *duit.Result, e duit.Event) {
	if e.NeedLayout {
		self.Layout = duit.Dirty
	}
	if e.NeedDraw {
		self.Draw = duit.Dirty
	}
	r.Consumed = e.Consumed || r.Consumed
}

func (ui *Label) Rect() draw.Rectangle {
	if ui == nil {
		return draw.Rectangle{}
	}
	return draw.Rectangle{
		ui.orig,
		ui.orig.Add(ui.size),
	}
}
