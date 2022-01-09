package style

import (
	"bytes"
	"testing"
)

func TestParseInline(t *testing.T) {
	b := bytes.NewBufferString("color: red;")
	s, err := Parse(b, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("s: %+v", s)
	if len(s.Rules) != 1 {
		t.Fail()
	}
	r := s.Rules[0]
	if len(r.Declarations) != 1 {
		t.Fail()
	}
	d := r.Declarations[0]
	if d.Prop != "color" || d.Val != "red" {
		t.Fail()
	}
}

func TestParseMin(t *testing.T) {
	b := bytes.NewBufferString(`
		h1 {
			font-weight: bold;
			font-size: 100px;
		}
		p, quote, a < b, div {
			color: grey !important;
		}
		:root {
			--emph: red;
			--h: 10px;
		}

		b {
			color: var(--emph);
		}
	`)
	s, err := Parse(b, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("s: %+v", s)
	if len(s.Rules) != 4 {
		t.Fatalf("%+v", s)
	}
	r := s.Rules[0]
	if len(r.Declarations) != 2 || len(r.Selectors) != 1 || r.Selectors[0].Val != "h1" {
		t.Fatalf("%+v", r)
	}
	d := r.Declarations[0]
	if d.Prop != "font-weight" || d.Val != "bold" || d.Important {
		t.Fatalf("%+v", d)
	}
	r = s.Rules[1]
	if len(r.Declarations) != 1 || len(r.Selectors) != 3 {
		t.Fatalf("%+v", r)
	}
	d = r.Declarations[0]
	if d.Prop != "color" || d.Val != "grey" || !d.Important {
		t.Fatalf("%+v", d)
	}
	r = s.Rules[2]
	if len(r.Declarations) != 2 || len(r.Selectors) != 1 || r.Selectors[0].Val != ":root" {
		t.Fatalf("%+v", r)
	}
	d = r.Declarations[0]
	if d.Prop != "--emph" || d.Val != "red" {
		t.Fatalf("%+v %+v", r, d)
	}
}

func TestParseMedia(t *testing.T) {
	b := bytes.NewBufferString(`
		@media only screen and (max-width: 600px) {
		  body {
		    background-color: lightblue;
		  }
		}
	`)
	s, err := Parse(b, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("s: %+v", s)
	t.Logf("s.Rules[0].Prelude: %+v", s.Rules[0].Prelude)
	//t.Logf("s.Rules[0].Prelude: %+v", s.Rules[0].Rules[0].Prelude)
	if len(s.Rules) != 1 {
		t.Fatalf("%+v", s)
	}
	r := s.Rules[0]
	if len(r.Declarations) != 0 || len(r.Selectors) > 0 {
		t.Fatalf("%+v", r)
	}
	d := r.Rules[0].Declarations[0]
	if d.Prop != "background-color" || d.Val != "lightblue" {
		t.Fatalf("%+v", d)
	}
}

func TestParseComment(t *testing.T) {
	b := bytes.NewBufferString(`
		h1 {
			font-weight: bold;
			font-size: 100px;
		}
		/* grey text */
		p {
			color: grey !important;
		}
	`)
	s, err := Parse(b, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("s: %+v", s)
	if len(s.Rules) != 2 {
		t.Fatalf("%v", s)
	}
	r := s.Rules[0]
	if len(r.Declarations) != 2 || r.Selectors[0].Val != "h1" {
		t.Fatalf("%v", r)
	}
	d := r.Declarations[0]
	if d.Prop != "font-weight" || d.Val != "bold" {
		t.Fatalf("%v", d)
	}
	r = s.Rules[1]
	if len(r.Declarations) != 1 || r.Selectors[0].Val != "p" {
		t.Fatalf("%v", r)
	}
	d = r.Declarations[0]
	if d.Prop != "color" || d.Val != "grey" || !d.Important {
		t.Fatalf("%v", d)
	}
}

func TestParseQual(t *testing.T) {
	b := bytes.NewBufferString(`
		h1 {
			font-weight: bold;
			font-size: 100px;
		}
		p {
			color: grey !important;
		}
		a[href] {
		  color: blue;
		  margin-right: 2px;
		}
	`)
	s, err := Parse(b, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("s: %+v", s)
	if len(s.Rules) != 3 {
		t.Fail()
	}
	r := s.Rules[0]
	if len(r.Declarations) != 2 || r.Selectors[0].Val != "h1" {
		t.Fail()
	}
	d := r.Declarations[0]
	if d.Prop != "font-weight" || d.Val != "bold" {
		t.Fail()
	}
	r = s.Rules[2]
	if len(r.Declarations) != 2 || r.Selectors[0].Val != "a[href]" {
		t.Fatalf("%+v", r)
	}
	d = r.Declarations[0]
	if d.Prop != "color" || d.Val != "blue" {
		t.Fail()
	}
}
