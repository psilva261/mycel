package duitx

import (
	"image"
	"testing"
)

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
