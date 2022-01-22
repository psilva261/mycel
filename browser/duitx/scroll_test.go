package duitx

import (
	"9fans.net/go/draw"
	"github.com/mjl-/duit"
	"image"
	"testing"
	"time"
)

func TestFreeCur(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	ui := &Scroll{
		r: image.Rectangle{
			Min: image.Point{0, 0},
			Max: image.Point{100, 1000},
		},
		Offset: 1,
		tiles:  make(map[int]*draw.Image),
		last:   make(map[int]time.Time),
	}
	dui, err := duit.NewDUI("scroll_test", nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	r := rect(draw.Point{100, 100})
	for i := 0; i < 10; i++ {
		ui.tiles[i], err = dui.Display.AllocImage(r, draw.ARGB32, false, 0xff00ff00)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	}
	ui.freeCur()
	if len(ui.tiles) != 8 {
		t.Fail()
	}
}

func TestPos(t *testing.T) {
	s := &Scroll{
		r: image.Rectangle{
			Min: image.Point{0, 0},
			Max: image.Point{1600, 1200},
		},
		Offset: 0,
	}
	if i, of := s.pos(); i != 0 || of != 0 {
		t.Fatalf("%v %v", i, of)
	}
	s.Offset = 3400
	if i, of := s.pos(); i != 2 || of != 1000 {
		t.Fatalf("%v %v", i, of)
	}
}

func TestTlR(t *testing.T) {
	s := &Scroll{
		r: image.Rectangle{
			Min: image.Point{0, 0},
			Max: image.Point{1600, 1200},
		},
	}
	if r := s.tlR(0); r.Min.X != 0 || r.Min.Y != 0 || r.Dx() != 1600 || r.Dy() != 1200 {
		t.Fatalf("%v", r)
	}
	if r := s.tlR(2); r.Min.X != 0 || r.Min.Y != 2400 || r.Dx() != 1600 || r.Dy() != 1200 {
		t.Fatalf("%v", r)
	}
}
