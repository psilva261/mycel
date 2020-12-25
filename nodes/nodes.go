package nodes

import (
	"golang.org/x/net/html"
	"opossum/logger"
	"opossum/style"
	"strings"
)

var log *logger.Logger
func SetLogger(l *logger.Logger) {
	log = l
}

// Node represents a node at the render stage. It
// represents a subTree or just a single html node.
type Node struct {
	DomSubtree *html.Node
	Text string
	Wrappable bool
	Attr []html.Attribute
	style.Map
	Children []*Node
	Parent *Node
}

// NewNodeTree propagates the cascading styles to the leaves
//
// First applies the global style and at the end the local style attribute's style is attached.
func NewNodeTree(doc *html.Node, cs style.Map, nodeMap map[*html.Node]style.Map, parent *Node) (n *Node) {
	ncs := cs
	if m, ok := nodeMap[doc]; ok {
		ncs = ncs.ApplyChildStyle(m)
	}
	ncs = ncs.ApplyChildStyle(style.NewMap(doc))
	data := doc.Data
	if doc.Type == html.ElementNode {
		data = strings.ToLower(data)
	}
	n = &Node{
		//Data:           data,
		//Type:           doc.Type,
		DomSubtree:   doc,
		Attr:           doc.Attr,
		Map: ncs,
		Children:       make([]*Node, 0, 2),
		Parent: parent,
	}
	n.Wrappable = doc.Type == html.TextNode || doc.Data == "span" // TODO: probably this list needs to be extended
	if doc.Type == html.TextNode {

		n.Text = filterText(doc.Data)
	}
	i := 0
	for c := doc.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.CommentNode {
			n.Children = append(n.Children, NewNodeTree(c, ncs, nodeMap, n))
			i++
		}
	}

	return
}

// filterText removes line break runes (TODO: add this later but handle properly)
func filterText(t string) (text string) {
	return strings.ReplaceAll(t, "­", "")
}

func (n Node) Type() html.NodeType {
	return n.DomSubtree.Type
}

func (n Node) Data() string {
	if n.DomSubtree == nil {
		return ""
	}
	return n.DomSubtree.Data
}

// Ancestor of tag
func (n *Node) Ancestor(tag string) *Node {
	log.Printf("<%v>.ParentForm()", n.DomSubtree.Data)
	if n.DomSubtree.Data == tag {
		log.Printf("  I'm a %v :-)", tag)
		return n
	}
	if n.Parent != nil {
		log.Printf("  go to my parent")
		return n.Parent.Ancestor(tag)
	}
	return nil
}

func (n *Node) QueryRef() string {
	path := make([]string, 0, 5)
	path = append(path, n.Data())
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Data() != "html" && p.Data() != "body" {
			path = append([]string{p.Data()}, path...)
		} 
	}
	return strings.TrimSpace(strings.Join(path, " "))
}

func IsPureTextContent(n Node) bool {
	if n.Text != "" {
		return true
	}
	for _, c := range n.Children {
		if c.Text == "" {
			return false
		}
	}
	return true
}

func ContentFrom(n Node) string {
	var content string

	if n.Text != "" && n.Type() == html.TextNode && !n.Map.IsDisplayNone() {
		content += n.Text
	}

	for _, c := range n.Children {
		if !c.Map.IsDisplayNone() {
			content += ContentFrom(*c)
		}
	}

	return strings.TrimSpace(content)
}