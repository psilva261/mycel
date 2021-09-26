package opossum

import (
	"testing"

	"github.com/chris-ramon/douceur/parser"
)

// aymerick douceur issues #6
func TestInfiniteLoop(t *testing.T) {
	parser.Parse(`
@media ( __desktop ) {
  background-color: red;
}
`)
}
