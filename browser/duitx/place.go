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
	"image"

	"9fans.net/go/draw"
	"github.com/mjl-/duit"
	"github.com/psilva261/opossum/logger"
)

// Place contains other UIs it can position absolute, possibly on top of each other.
type Place struct {
	// Place is called during layout. It must configure Kids, and set self.R, based on sizeAvail.
	Place      func(self *duit.Kid, sizeAvail image.Point) `json:"-"`
	Kids       []*duit.Kid                                 // Kids to draw, set by the Place function.
	Background *draw.Image                                 `json:"-"` // For background color.

	kidsReversed []*duit.Kid
	size         image.Point
	imgs         []*draw.Image
	force        bool
}

var _ duit.UI = &Place{}

func (ui *Place) ensure() {
	if len(ui.kidsReversed) == len(ui.Kids) {
		return
	}
	ui.kidsReversed = make([]*duit.Kid, len(ui.Kids))
	for i, k := range ui.Kids {
		ui.kidsReversed[len(ui.Kids)-1-i] = k
	}
}

func (ui *Place) Layout(dui *duit.DUI, self *duit.Kid, sizeAvail image.Point, force bool) {
	ui.ensure()
	debugLayout(dui, self)

	ui.Place(self, sizeAvail)
}

func (ui *Place) Draw(dui *duit.DUI, self *duit.Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	if self.Draw == duit.Clean || ui == nil || len(ui.Kids) == 0 {
		return
	}
	self.Draw = duit.Clean
	if ui.imgs == nil || ui.Kids[0].R.Size() != ui.imgs[0].R.Size() {
		var err error
		if ui.imgs != nil {
			for _, i := range ui.imgs {
				i.Free()
			}
			ui.imgs = nil
		}
		if ui.Kids[0].R.Dx() == 0 || ui.Kids[0].R.Dy() == 0 {
			return
		}
		ui.imgs = make([]*draw.Image, len(ui.Kids))
		for i, k := range ui.Kids {
			ui.imgs[i], err = dui.Display.AllocImage(k.R, draw.ARGB32, false, 0x00000000)
			if err != nil {
				log.Errorf("allocimage: %v", err)
				return
			}
			k.Draw = duit.Dirty
		}
		self.Draw = duit.DirtyKid
	}
	if self.Draw == duit.DirtyKid || ui.force {
		kidsDraw(dui, self, ui.Kids, ui.size, ui.Background, ui.imgs, image.ZP, m, false)
		self.Draw = duit.Clean
		ui.force = false
	}
	for _, i := range ui.imgs {
		if i != nil {
			img.Draw(i.R.Add(orig), i, nil, image.ZP)
		}
	}
}

func kidsDraw(dui *duit.DUI, self *duit.Kid, kids []*duit.Kid, uiSize image.Point, bg *draw.Image, imgs []*draw.Image, orig image.Point, m draw.Mouse, force bool) {
	debugDraw(dui, self)

	force = force || self.Draw == duit.Dirty
	if force {
		self.Draw = duit.Dirty
	}

	if bg == nil {
		bg = dui.Background
	}
	if force {
		// TODO: consider resetting other backgrounds also
		imgs[0].Draw(rect(uiSize).Add(orig), bg, nil, image.ZP)
	}
	for i, k := range kids {
		if !force && k.Draw == duit.Clean {
			continue
		}
		if !force && k.Draw == duit.Dirty {
			imgs[i].Draw(k.R.Add(orig), bg, nil, image.ZP)
		}

		mm := m
		mm.Point = mm.Point.Sub(k.R.Min)
		if force {
			k.Draw = duit.Dirty
		}
		k.UI.Draw(dui, k, imgs[i], orig.Add(k.R.Min), mm, force)
		k.Draw = duit.Clean
	}
	self.Draw = duit.Clean
}

func (ui *Place) result(dui *duit.DUI, self *duit.Kid, r *duit.Result) {
	relayout := false
	redraw := false

	for _, k := range ui.Kids {
		if k.Layout != duit.Clean {
			relayout = true
		} else if k.Draw != duit.Clean {
			redraw = true
		}
	}
	if relayout {
		self.Layout = duit.DirtyKid
		self.Draw = duit.DirtyKid
		ui.force = ui.force || r.Consumed
	} else if redraw {
		self.Draw = duit.DirtyKid
		ui.force = ui.force || r.Consumed
	}
}

func (ui *Place) Mouse(dui *duit.DUI, self *duit.Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r duit.Result) {
	r = duit.KidsMouse(dui, self, ui.kidsReversed, m, origM, orig)
	ui.result(dui, self, &r)
	return
}

func (ui *Place) Key(dui *duit.DUI, self *duit.Kid, k rune, m draw.Mouse, orig image.Point) (r duit.Result) {
	r = duit.KidsKey(dui, self, ui.kidsReversed, k, m, orig)
	ui.result(dui, self, &r)
	return
}

func (ui *Place) FirstFocus(dui *duit.DUI, self *duit.Kid) (warp *image.Point) {
	return duit.KidsFirstFocus(dui, self, ui.Kids)
}

func (ui *Place) Focus(dui *duit.DUI, self *duit.Kid, o duit.UI) (warp *image.Point) {
	return duit.KidsFocus(dui, self, ui.Kids, o)
}

func (ui *Place) Mark(self *duit.Kid, o duit.UI, forLayout bool) (marked bool) {
	return duit.KidsMark(self, ui.Kids, o, forLayout)
}

func (ui *Place) Print(self *duit.Kid, indent int) {
	duit.PrintUI("Place", self, indent)
	duit.KidsPrint(ui.Kids, indent+1)
}
