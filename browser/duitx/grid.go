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

	"9fans.net/go/draw"
	"github.com/mjl-/duit"
)

// Grid lays out other duit.UIs in a table-like grid.
type Grid struct {
	Kids       []*duit.Kid      // Holds UIs in the grid, per row.
	Columns    int         // Number of clumns.
	Valign     []duit.Valign    // Vertical alignment per column.
	Halign     []duit.Halign    // Horizontal alignment per column.
	Padding    []duit.Space     // Padding in lowDPI pixels per column.
	Width      int         // -1 means full width, 0 means automatic width, >0 means exactly that many lowDPI pixels.
	Background *draw.Image `json:"-"` // Background color.

	widths  []int
	heights []int
	size    image.Point
}

var _ duit.UI = &Grid{}

func (ui *Grid) Layout(dui *duit.DUI, self *duit.Kid, sizeAvail image.Point, force bool) {
	debugLayout(dui, self)
	if duit.KidsLayout(dui, self, ui.Kids, force) {
		return
	}

	if ui.Valign != nil && len(ui.Valign) != ui.Columns {
		panic(fmt.Sprintf("len(valign) = %d, should be ui.Columns = %d", len(ui.Valign), ui.Columns))
	}
	if ui.Halign != nil && len(ui.Halign) != ui.Columns {
		panic(fmt.Sprintf("len(halign) = %d, should be ui.Columns = %d", len(ui.Halign), ui.Columns))
	}
	if ui.Padding != nil && len(ui.Padding) != ui.Columns {
		panic(fmt.Sprintf("len(padding) = %d, should be ui.Columns = %d", len(ui.Padding), ui.Columns))
	}
	if len(ui.Kids)%ui.Columns != 0 {
		panic(fmt.Sprintf("len(kids) = %d, should be multiple of ui.Columns = %d", len(ui.Kids), ui.Columns))
	}

	scaledWidth := dui.Scale(ui.Width)
	if scaledWidth > 0 && scaledWidth < sizeAvail.X {
		ui.size.X = scaledWidth
	}

	ui.widths = make([]int, ui.Columns) // widths include padding
	spaces := make([]duit.Space, ui.Columns)
	if ui.Padding != nil {
		for i, pad := range ui.Padding {
			spaces[i] = dui.ScaleSpace(pad)
		}
	}
	width := 0                       // total width so far
	x := make([]int, len(ui.widths)) // x offsets per column
	x[0] = 0

	// first determine the column widths
	for col := 0; col < ui.Columns; col++ {
		if col > 0 {
			x[col] = x[col-1] + ui.widths[col-1]
		}
		ui.widths[col] = 0
		newDx := 0
		space := spaces[col]
		for i := col; i < len(ui.Kids); i += ui.Columns {
			k := ui.Kids[i]
			k.UI.Layout(dui, k, image.Pt(sizeAvail.X-space.Dx(), sizeAvail.Y-space.Dy()), true)
			newDx = maximum(newDx, k.R.Dx()+space.Dx())
		}
		ui.widths[col] = newDx
		width += ui.widths[col]
	}

	// Reduce used widths if too large
	if width > sizeAvail.X {
		r := float64(sizeAvail.X) / float64(width)
		width = sizeAvail.X
		for i := range ui.widths {
			ui.widths[i] = int(float64(ui.widths[i])*r)
		}
	}

	// Enable full width if activated
	if scaledWidth < 0 && width < sizeAvail.X {
		leftover := sizeAvail.X - width
		given := 0
		for i := range ui.widths {
			x[i] += given
			var dx int
			if i == len(ui.widths)-1 {
				dx = leftover - given
			} else {
				dx = leftover / len(ui.widths)
			}
			ui.widths[i] += dx
			given += dx
		}
	}

	// now determine row heights
	ui.heights = make([]int, (len(ui.Kids)+ui.Columns-1)/ui.Columns)
	height := 0                       // total height so far
	y := make([]int, len(ui.heights)) // including padding
	y[0] = 0
	for i := 0; i < len(ui.Kids); i += ui.Columns {
		row := i / ui.Columns
		if row > 0 {
			y[row] = y[row-1] + ui.heights[row-1]
		}
		rowDy := 0
		for col := 0; col < ui.Columns; col++ {
			space := spaces[col]
			k := ui.Kids[i+col]
			k.UI.Layout(dui, k, image.Pt(ui.widths[col]-space.Dx(), sizeAvail.Y-y[row]-space.Dy()), true)
			offset := image.Pt(x[col], y[row]).Add(space.Topleft())
			k.R = k.R.Add(offset) // aligned in top left, fixed for halign/valign later on
			rowDy = maximum(rowDy, k.R.Dy()+space.Dy())
		}
		ui.heights[row] = rowDy
		height += ui.heights[row]
	}

	// now shift the kids for right valign/halign
	for i, k := range ui.Kids {
		row := i / ui.Columns
		col := i % ui.Columns
		space := spaces[col]

		valign := duit.ValignTop
		halign := duit.HalignLeft
		if ui.Valign != nil {
			valign = ui.Valign[col]
		}
		if ui.Halign != nil {
			halign = ui.Halign[col]
		}
		cellSize := image.Pt(ui.widths[col], ui.heights[row]).Sub(space.Size())
		spaceX := 0
		switch halign {
		case duit.HalignLeft:
		case duit.HalignMiddle:
			spaceX = (cellSize.X - k.R.Dx()) / 2
		case duit.HalignRight:
			spaceX = cellSize.X - k.R.Dx()
		}
		spaceY := 0
		switch valign {
		case duit.ValignTop:
		case duit.ValignMiddle:
			spaceY = (cellSize.Y - k.R.Dy()) / 2
		case duit.ValignBottom:
			spaceY = cellSize.Y - k.R.Dy()
		}
		k.R = k.R.Add(image.Pt(spaceX, spaceY))
	}

	ui.size = image.Pt(width, height)
	if ui.Width < 0 && ui.size.X < sizeAvail.X {
		ui.size.X = sizeAvail.X
	}
	self.R = rect(ui.size)
}

func (ui *Grid) Draw(dui *duit.DUI, self *duit.Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	duit.KidsDraw(dui, self, ui.Kids, ui.size, ui.Background, img, orig, m, force)
}

func (ui *Grid) Mouse(dui *duit.DUI, self *duit.Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r duit.Result) {
	return duit.KidsMouse(dui, self, ui.Kids, m, origM, orig)
}

func (ui *Grid) Key(dui *duit.DUI, self *duit.Kid, k rune, m draw.Mouse, orig image.Point) (r duit.Result) {
	return duit.KidsKey(dui, self, ui.Kids, k, m, orig)
}

func (ui *Grid) FirstFocus(dui *duit.DUI, self *duit.Kid) *image.Point {
	return duit.KidsFirstFocus(dui, self, ui.Kids)
}

func (ui *Grid) Focus(dui *duit.DUI, self *duit.Kid, o duit.UI) *image.Point {
	return duit.KidsFocus(dui, self, ui.Kids, o)
}

func (ui *Grid) Mark(self *duit.Kid, o duit.UI, forLayout bool) (marked bool) {
	return duit.KidsMark(self, ui.Kids, o, forLayout)
}

func (ui *Grid) Print(self *duit.Kid, indent int) {
	duit.PrintUI(fmt.Sprintf("Grid columns=%d padding=%v", ui.Columns, ui.Padding), self, indent)
	duit.KidsPrint(ui.Kids, indent+1)
}
