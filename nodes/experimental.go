package nodes

import (
	"fmt"
	"github.com/andybalholm/cascadia"
	"github.com/psilva261/opossum"
	"golang.org/x/net/html"
)

// Path relative to body
func (n *Node) Path() (p string, ok bool) {
	p, ok = n.path()
	if ok {
		p = opossum.PathPrefix+p
	}
	return
}

func (n *Node) path() (p string, ok bool) {
	var i int
	var c *Node

	if n.DomSubtree == nil || n.Type() != html.ElementNode {
		return
	}
	if n.parent == nil {
		return "/", true
	}
	for i, c = range n.parent.Children {
		if c == n {
			break
		}
		if c.Type() == html.ElementNode {
			i++
		}
	}
	p += fmt.Sprintf("/%v", i)
	q, ok := n.parent.path()
	if ok {
		p = q + p
	}
	return p, true
}

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
