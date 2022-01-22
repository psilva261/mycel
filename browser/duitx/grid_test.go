package duitx

import (
	"image"
	"testing"

	"github.com/mjl-/duit"
)

func TestInitPos(t *testing.T) {
	g := &Grid{
		Kids:     make([]*duit.Kid, 2*2),
		Columns:  2,
		Rows:     2,
		RowSpans: []int{1, 1, 1, 1},
		ColSpans: []int{1, 1, 1, 1},
	}
	g.initPos()
	if len(g.pos) != 2 || len(g.pos[0]) != 2 || len(g.pos[1]) != 2 || g.pos[0][0] != 0 || g.pos[0][1] != 1 || g.pos[1][0] != 2 || g.pos[1][1] != 3 {
		t.Fatalf("%+v", g.pos)
	}

	g = &Grid{
		Kids:     make([]*duit.Kid, 1*2),
		Columns:  2,
		Rows:     2,
		RowSpans: []int{1, 1},
		ColSpans: []int{2, 2},
	}
	g.initPos()
	if len(g.pos) != 2 || len(g.pos[0]) != 2 || len(g.pos[1]) != 2 || g.pos[0][0] != 0 || g.pos[0][1] != 1 || g.pos[1][0] != 0 || g.pos[1][1] != 1 {
		t.Fatalf("..%+v", g.pos)
	}

	g = &Grid{
		Kids:     make([]*duit.Kid, 2*1),
		Columns:  2,
		Rows:     2,
		RowSpans: []int{2, 2},
		ColSpans: []int{1, 1},
	}
	g.initPos()
	if len(g.pos) != 2 || len(g.pos[0]) != 2 || len(g.pos[1]) != 2 || g.pos[0][0] != 0 || g.pos[0][1] != 0 || g.pos[1][0] != 1 || g.pos[1][1] != 1 {
		t.Fatalf("%+v", g.pos)
	}
}

func TestMaxWidths(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	opts := &duit.DUIOpts{
		Dimensions: "400x300",
	}
	dui, err := duit.NewDUI("scroll_test", opts)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	g := Grid{
		Kids: duit.NewKids(
			&duit.Button{Text: "upper"},
			&duit.Button{Text: "LL"},
			&duit.Button{Text: "LR"},
		),
		Columns:  2,
		Rows:     2,
		RowSpans: []int{1, 1, 1},
		ColSpans: []int{2, 1, 1},
	}
	g.initPos()
	maxW, w, xs := g.maxWidths(dui, image.Point{X: 400, Y: 300})
	if len(maxW) != 2 || maxW[0]+maxW[1] != w || len(xs) != 2 || xs[0] != 0 || xs[1] != maxW[0] {
		t.Fatalf("%+v, %v, %+v", maxW, w, xs)
	}
}
