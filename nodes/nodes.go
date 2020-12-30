package nodes

import (
	"bytes"
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
	Attrs []html.Attribute
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
		Attrs:           doc.Attr,
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
	if n.DomSubtree == nil {
		return nil
	}
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

func (n *Node) Find(tag string) (c *Node) {
	for _, cc := range n.Children {
		if cc.Data() == tag {
			return cc
		} else if f := cc.Find(tag); f != nil {
			return f
		}
	}

	return
}

func (n *Node) Attr(k string) string {
	for _, a := range n.Attrs {
		if a.Key == k {
			return a.Val
		}
	}
	return ""
}

func (n *Node) QueryRef() string {
	nRef, ok := n.queryRef()
	if ok && strings.Contains(nRef, "#") {
		return nRef
	}

	path := make([]string, 0, 5)
	if n.Type() != html.TextNode {
		if ok {
			path = append(path, nRef)
		}
	}
	for p := n.Parent; p != nil; p = p.Parent {
		if part := p.Data(); part != "html" && part != "body" {
			if pRef, ok := p.queryRef(); ok {
				path = append([]string{pRef}, path...)
				if strings.Contains(pRef, "#") {
					break
				}
			}
		}
	}
	return strings.TrimSpace(strings.Join(path, " "))
}

func (n *Node) queryRef() (ref string, ok bool) {
	if n.DomSubtree == nil || n.Type() != html.ElementNode {
		return
	}

	if id := n.Attr("id"); id != "" {
		return "#" + id, true
	}

	ref = n.Data()

	var sl []string
	if c := strings.TrimSpace(n.Attr("class")); c != "" {
		l := strings.Split(c, " ")
		sl = make([]string, 0, len(l))

		for _, cl := range l {
			if cl == "" {
				continue
			}

			sl = append(sl, cl)
		}

		if len(sl) > 0 {
			ref += "." + strings.Join(sl, ".")
		}
	}

	return ref, true
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

func (n *Node) Serialized() (string, error) {
	var b bytes.Buffer

	err := html.Render(&b, n.DomSubtree)

	return b.String(), err
}

