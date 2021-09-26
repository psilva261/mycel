package cache

import (
	"github.com/psilva261/opossum"
	"sort"
	"time"
)

var c = make(Items, 0, 100)

type Items []*Item

func (is Items) Len() int {
	return len(is)
}

func (is Items) Swap(i, j int) {
	is[i], is[j] = is[j], is[i]
}

func (is Items) Less(i, j int) bool {
	return is[i].Used.After(is[j].Used)
}

type Item struct {
	Addr string
	opossum.ContentType
	Buf  []byte
	Used time.Time
}

func Get(addr string) (i Item, ok bool) {
	for _, it := range c {
		if it.Addr == addr {
			it.Used = time.Now()
			return *it, true
		}
	}
	return
}

func Set(i Item) {
	i.Used = time.Now()
	c = append(c, &i)
}

func Tidy() {
	if len(c) < 100 {
		return
	}
	sort.Stable(c)
	c = c[0:50]
}
