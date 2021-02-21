package browser

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
	"image"

	"9fans.net/go/draw"
	"github.com/mjl-/duit"
)

// NewBox returns a box containing all uis in its Kids field.
func NewBox(uis ...duit.UI) *Box {
	kids := make([]*duit.Kid, len(uis))
	for i, ui := range uis {
		kids[i] = &duit.Kid{UI: ui}
	}
	return &Box{Kids: kids}
}

// NewReverseBox returns a box containing all uis in original order in its Kids field, with the Reverse field set.
func NewReverseBox(uis ...duit.UI) *Box {
	kids := make([]*duit.Kid, len(uis))
	for i, ui := range uis {
		kids[i] = &duit.Kid{UI: ui}
	}
	return &Box{Kids: kids, Reverse: true}
}

// Box keeps elements on a line as long as they fit, then moves on to the next line.
type Box struct {
	Kids       []*duit.Kid      // Kids and UIs in this box.
	Reverse    bool        // Lay out children from bottom to top. First kid will be at the bottom.
	Margin     duit.Space // In lowDPI pixels, will be adjusted for highDPI screens.
	Padding    duit.Space       // Padding inside box, so children don't touch the sides; in lowDPI pixels, also adjusted for highDPI screens.
	Valign     duit.Valign      // How to align children on a line.
	Width      int         // 0 means dynamic (as much as needed), -1 means full width, >0 means that exact amount of lowDPI pixels.
	Height     int         // 0 means dynamic (as much as needed), -1 means full height, >0 means that exact amount of lowDPI pixels.
	MaxWidth   int         // if >0, the max number of lowDPI pixels that will be used.
	ContentBox bool        // Use ContentBox (BorderBox by default)
	Background *draw.Image `json:"-"` // Background for this box, instead of default duit background.

	size image.Point // of entire box, including padding but excluding margin
}

var _ duit.UI = &Box{}

func (ui *Box) Layout(dui *duit.DUI, self *duit.Kid, sizeAvail image.Point, force bool) {
	debugLayout(dui, self)
	if duit.KidsLayout(dui, self, ui.Kids, force) {
		return
	}

	if ui.Width < 0 && ui.MaxWidth > 0 {
		panic("combination ui.Width < 0 and ui.MaxWidth > 0 invalid")
	}

	padding := dui.ScaleSpace(ui.Padding)
	margin := dui.ScaleSpace(ui.Margin)

	// widths and heights
	bbw := dui.Scale(ui.Width)
	bbmaxw := dui.Scale(ui.MaxWidth)
	bbh := dui.Scale(ui.Height)

	if ui.ContentBox {
		bbw += margin.Dx()+padding.Dx()
		bbmaxw += margin.Dx()+padding.Dx()
		bbh += margin.Dy()+padding.Dy()
	}

	osize := sizeAvail
	if ui.Width > 0 && bbw < sizeAvail.X {
		sizeAvail.X = bbw
	} else if ui.MaxWidth > 0 && bbmaxw < sizeAvail.X {
		// note: ui.Width is currently the same as MaxWidth, but that might change when we don't mind extending beyong given X, eg with horizontal scroll
		sizeAvail.X = bbmaxw
	}
	if ui.Height > 0 {
		sizeAvail.Y = bbh
	}
	sizeAvail = sizeAvail.Sub(padding.Size()).Sub(margin.Size())
	nx := 0 // number on current line

	// variables below are about box contents excluding offsets for padding and margin
	cur := image.ZP
	xmax := 0  // max x seen so far
	lineY := 0 // max y of current line

	fixValign := func(kids []*duit.Kid) {
		if len(kids) < 2 {
			return
		}
		for _, k := range kids {
			switch ui.Valign {
			case duit.ValignTop:
			case duit.ValignMiddle:
				k.R = k.R.Add(image.Pt(0, (lineY-k.R.Dy())/2))
			case duit.ValignBottom:
				k.R = k.R.Add(image.Pt(0, lineY-k.R.Dy()))
			}
		}
	}

	for i, k := range ui.Kids {
		k.UI.Layout(dui, k, sizeAvail.Sub(image.Pt(0, cur.Y+lineY)), true)
		childSize := k.R.Size()
		var kr image.Rectangle
		if nx == 0 || cur.X+childSize.X <= sizeAvail.X {
			kr = rect(childSize).Add(cur).Add(padding.Topleft())
			cur.X += childSize.X
			lineY = maximum(lineY, childSize.Y)
			nx += 1
		} else {
			if nx > 0 {
				fixValign(ui.Kids[i-nx : i])
				cur.X = 0
				cur.Y += lineY + margin.Topleft().Y
			}
			// Add padding translation, so the child UI can be drawn right there
			kr = rect(childSize).Add(cur).Add(padding.Topleft())
			nx = 1
			cur.X = childSize.X
			lineY = childSize.Y
		}
		k.R = kr
		if xmax < cur.X {
			xmax = cur.X
		}
	}
	fixValign(ui.Kids[len(ui.Kids)-nx : len(ui.Kids)])
	cur.Y += lineY

	if ui.Reverse {
		bottomY := cur.Y + padding.Dy()
		for _, k := range ui.Kids {
			y1 := bottomY - k.R.Min.Y
			y0 := y1 - k.R.Dy()
			k.R = image.Rect(k.R.Min.X, y0, k.R.Max.X, y1)
		}
	}

	ui.size = image.Pt(xmax, cur.Y).Add(padding.Size())
	if ui.Width < 0 {
		ui.size.X = osize.X
	}
	if ui.Height < 0 && ui.size.Y < osize.Y {
		ui.size.Y = osize.Y
	}
	self.R = rect(ui.size.Add(margin.Size()))
}

func (ui *Box) Draw(dui *duit.DUI, self *duit.Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	margin := dui.ScaleSpace(ui.Margin)
	orig = orig.Add(margin.Topleft())
	duit.KidsDraw(dui, self, ui.Kids, ui.size, ui.Background, img, orig, m, force)
}

func (ui *Box) Mouse(dui *duit.DUI, self *duit.Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r duit.Result) {
	margin := dui.ScaleSpace(ui.Margin)
	origM.Point = origM.Point.Sub(margin.Topleft())
	m.Point = m.Point.Sub(margin.Topleft())
	return duit.KidsMouse(dui, self, ui.Kids, m, origM, orig)
}

func (ui *Box) Key(dui *duit.DUI, self *duit.Kid, k rune, m draw.Mouse, orig image.Point) (r duit.Result) {
	// nil check for tests
	if dui != nil {
		margin := dui.ScaleSpace(ui.Margin)
		m.Point = m.Point.Sub(margin.Topleft())
	}
	return duit.KidsKey(dui, self, ui.orderedKids(), k, m, orig)
}

func (ui *Box) orderedKids() []*duit.Kid {
	if !ui.Reverse {
		return ui.Kids
	}
	n := len(ui.Kids)
	kids := make([]*duit.Kid, n)
	for i := range ui.Kids {
		kids[i] = ui.Kids[n-1-i]
	}
	return kids
}

func (ui *Box) FirstFocus(dui *duit.DUI, self *duit.Kid) *image.Point {
	return duit.KidsFirstFocus(dui, self, ui.orderedKids())
}

func (ui *Box) Focus(dui *duit.DUI, self *duit.Kid, o duit.UI) *image.Point {
	return duit.KidsFocus(dui, self, ui.Kids, o)
}

func (ui *Box) Mark(self *duit.Kid, o duit.UI, forLayout bool) (marked bool) {
	return duit.KidsMark(self, ui.Kids, o, forLayout)
}

func (ui *Box) Print(self *duit.Kid, indent int) {
	duit.PrintUI("Box", self, indent)
	duit.KidsPrint(ui.Kids, indent+1)
}
