package duitx

import (
	"testing"

	"github.com/mjl-/duit"
)

func TestInitPos(t *testing.T) {
	g := &Grid{
		Kids: make([]*duit.Kid, 2*2),
		Columns: 2,
		Rows: 2,
		RowSpans: []int{1, 1, 1, 1},
		ColSpans: []int{1, 1, 1, 1},
	}
	g.initPos()
	if len(g.pos) != 2 || len(g.pos[0]) != 2 || len(g.pos[1]) != 2 || g.pos[0][0] != 0 || g.pos[0][1] != 1 || g.pos[1][0] != 2 || g.pos[1][1] != 3 {
		t.Fatalf("%+v", g.pos)
	}

	g = &Grid{
		Kids: make([]*duit.Kid, 1*2),
		Columns: 2,
		Rows: 2,
		RowSpans: []int{1, 1},
		ColSpans: []int{2, 2},
	}
	g.initPos()
	if len(g.pos) != 2 || len(g.pos[0]) != 2 || len(g.pos[1]) != 2 || g.pos[0][0] != 0 || g.pos[0][1] != 1 || g.pos[1][0] != 0 || g.pos[1][1] != 1 {
		t.Fatalf("..%+v", g.pos)
	}

	g = &Grid{
		Kids: make([]*duit.Kid, 2*1),
		Columns: 2,
		Rows: 2,
		RowSpans: []int{2, 2},
		ColSpans: []int{1, 1},
	}
	g.initPos()
	if len(g.pos) != 2 || len(g.pos[0]) != 2 || len(g.pos[1]) != 2 || g.pos[0][0] != 0 || g.pos[0][1] != 0 || g.pos[1][0] != 1 || g.pos[1][1] != 1 {
		t.Fatalf("%+v", g.pos)
	}
}
