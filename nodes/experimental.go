package nodes

import (
	"fmt"
	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
)

func (n *Node) Query(s string) (ns []*Node, err error) {
	cs, err := cascadia.Compile(s)
	if err != nil {
		return nil, fmt.Errorf("cssSel compile %v: %w", s, err)
	}
	var m func(doc *html.Node, nn *Node) *Node
	m = func(doc *html.Node, nn *Node) *Node {
		if nn.DomSubtree == doc {
			return nn
		}
		for _, c := range nn.Children {
			if res := m(doc, c); res != nil {
				return res
			}
		}
		return nil
	}
	if n == nil {
		return nil, fmt.Errorf("nil node tree")
	}
	for _, el := range cascadia.QueryAll(n.DomSubtree, cs) {
		if res := m(el, n); res != nil {
			ns = append(ns, res)
		}
	}
	return
}

func (n *Node) NumVClusters() (m int) {
	if n.IsFlex() {
		if n.IsFlexDirectionRow() {
			return 1
		} else {
			return len(n.Children)
		}
	} else {
		for i, c := range n.Children {
			if i == 0 || !c.IsInline() {
				m++
			}
		}
	}
	return
}

func (n *Node) VSlice(i, j int) (s []*Node) {
	s = make([]*Node, 0, j-i)
	m := 0
	for l, c := range n.Children {
		if l == 0 || !c.IsInline() {
			m++
		}
		if i <= l && l <= j {
			s = append(s, c)
		}
	}
	return
}

type Fan struct {
	from []int
	to   []int
}

func (f Fan) Slice(root *Node) *Node {
	return nil
}

/*func FanSlice(root *Node, depth, k, i, j int) *Node {
	newRoot := *root
	if depth == 0 {

	} else {
		for i, c := range root.Children {
			newRoot.Children[i] =
		}
	}
}*/
