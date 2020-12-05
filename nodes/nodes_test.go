package nodes

import (
	"testing"
)

func TestFilterText(t *testing.T) {
	in := "eben­falls"
	exp := "ebenfalls"
	if out := filterText(in); out != exp {
		t.Fatalf("%+v", out)
	}
}