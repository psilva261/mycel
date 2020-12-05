package nodes

import (
	//"golang.org/x/net/html"
	//"opossum/style"
	//"strings"
)
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
