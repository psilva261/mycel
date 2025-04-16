package fs

import (
	"bytes"
	"fmt"
	"github.com/knusbaum/go9p/fs"
	"github.com/knusbaum/go9p/proto"
	"github.com/psilva261/mycel/logger"
	"github.com/psilva261/mycel/nodes"
	"github.com/psilva261/mycel/style"
	"golang.org/x/net/html"
)

// Node such that one obtains a file structure like
//
// /0
// /0/attrs
// /0/geom
// /0/html
// /0/style
// /0/tag
// /0/0
//   ...
// /0/1
//   ...
// ...
//
// (dir structure stolen from domfs)
type Node struct {
	fs   *FS
	name string
	nt   *nodes.Node
}

func (n Node) Stat() (s proto.Stat) {
	s = *n.fs.oFS.NewStat(n.name, n.fs.un, n.fs.gn, 0700)
	s.Mode |= proto.DMDIR
	// qtype bits should be consistent with Stat mode.
	s.Qid.Qtype = uint8(s.Mode >> 24)
	return
}

func (n Node) WriteStat(s *proto.Stat) error {
	return nil
}

func (n Node) SetParent(p fs.Dir) {
}

func (n Node) Parent() fs.Dir {
	return nil
}

func (n Node) Children() (cs map[string]fs.FSNode) {
	cs = make(map[string]fs.FSNode)
	if n.nt == nil {
		return
	}
	for i, c := range n.nt.Children {
		ddn := fmt.Sprintf("%v", i)
		cs[ddn] = &Node{
			fs: n.fs,
			name: ddn,
			nt:   c,
		}
	}
	if n.nt.Type() == html.ElementNode {
		cs["tag"] = n.tag()
		cs["attrs"] = Attrs{attrs: &n.nt.DomSubtree.Attr}
		cs["geom"] = n.geom()
		cs["html"] = n.html()
		cs["style"] = Style{cs: &n.nt.Map}
	}

	return
}

func (n Node) tag() fs.FSNode {
	return fs.NewDynamicFile(
		n.fs.oFS.NewStat("tag", n.fs.un, n.fs.gn, 0666),
		func() []byte {
			return []byte(n.nt.Data())
		},
	)
}

func (n Node) geom() fs.FSNode {
	return fs.NewDynamicFile(
		n.fs.oFS.NewStat("geom", n.fs.un, n.fs.gn, 0666),
		func() (bs []byte) {
			var dt style.DomTree
			if dt = n.nt.Map.DomTree; dt == nil {
				return
			}
			r := dt.Rect()
			return []byte(fmt.Sprintf("%v,%v,%v,%v", r.Min.X, r.Min.Y, r.Max.X, r.Max.Y))
		},
	)
}

func (n Node) html() fs.FSNode {
	return fs.NewDynamicFile(
		n.fs.oFS.NewStat("html", n.fs.un, n.fs.gn, 0666),
		func() []byte {
			buf := bytes.NewBufferString("")
			if err := html.Render(buf, n.nt.DomSubtree); err != nil {
				log.Errorf("render: %v", err)
				return []byte{}
			}
			return []byte(buf.String())
		},
	)
}

func (n Node) AddChild(fs.FSNode) error {
	return nil
}

func (n Node) DeleteChild(name string) error {
	return fmt.Errorf("no removal possible")
}

type Attrs struct {
	n     *Node
	attrs *[]html.Attribute
}

func (as Attrs) Stat() (s proto.Stat) {
	s = *as.n.fs.oFS.NewStat("attrs", as.n.fs.un, as.n.fs.gn, 0500)
	s.Mode |= proto.DMDIR
	// qtype bits should be consistent with Stat mode.
	s.Qid.Qtype = uint8(s.Mode >> 24)
	return
}

func (as Attrs) WriteStat(s *proto.Stat) error {
	return nil
}

func (as Attrs) SetParent(p fs.Dir) {
}

func (as Attrs) Parent() fs.Dir {
	return nil
}

func (as Attrs) Children() (cs map[string]fs.FSNode) {
	log.Printf("Attrs#Children()")
	cs = make(map[string]fs.FSNode)
	ff := func(k string) fs.FSNode {
		return fs.NewDynamicFile(
			as.n.fs.oFS.NewStat(k, as.n.fs.un, as.n.fs.gn, 0666),
			func() []byte {
				var v string
				for _, a := range *as.attrs {
					if a.Key == k {
						v = a.Val
					}
				}
				return []byte(v)
			},
		)
	}
	for _, attr := range *as.attrs {
		cs[attr.Key] = ff(attr.Key)
	}
	return
}

type Style struct {
	n  *Node
	cs *style.Map
}

func (st Style) Stat() (s proto.Stat) {
	s = *st.n.fs.oFS.NewStat("style", st.n.fs.un, st.n.fs.gn, 0500)
	s.Mode |= proto.DMDIR
	// qtype bits should be consistent with Stat mode.
	s.Qid.Qtype = uint8(s.Mode >> 24)
	return
}

func (st Style) WriteStat(s *proto.Stat) error {
	return nil
}

func (st Style) SetParent(p fs.Dir) {
}

func (st Style) Parent() fs.Dir {
	return nil
}

func (st Style) Children() (cs map[string]fs.FSNode) {
	log.Printf("Style#Children()")
	cs = make(map[string]fs.FSNode)
	ff := func(k string) fs.FSNode {
		return fs.NewDynamicFile(
			st.n.fs.oFS.NewStat(k, st.n.fs.un, st.n.fs.gn, 0666),
			func() []byte {
				var v string
				for p, d := range st.cs.Declarations {
					if p == k {
						v = d.Val
					}
				}
				return []byte(v)
			},
		)
	}
	for p := range st.cs.Declarations {
		cs[p] = ff(p)
	}
	return
}
