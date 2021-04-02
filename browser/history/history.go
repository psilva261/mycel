package history

import (
	"net/url"
	"strings"
)

type History struct {
	items []Item
}

func (h History) URL() *url.URL {
	return h.items[len(h.items)-1].URL
}

func (h *History) Push(u *url.URL, oldScroll int) {
	if len(h.items) > 0 {
		if h.items[len(h.items)-1].URL.String() == u.String() {
			return
		}
		h.setScroll(oldScroll)
	}
	it := Item{u, 0}
	h.items = append(h.items, it)
}

func (h *History) Back() {
	if len(h.items) > 1 {
		h.items = h.items[:len(h.items)-1]
	}
}

func (h *History) String() string {
	addrs := make([]string, len(h.items))
	for i, it := range h.items {
		addrs[i] = it.URL.String()
	}
	return strings.Join(addrs, ", ")
}

func (h *History) Scroll() int {
	return h.items[len(h.items)-1].Scroll
}

func (h *History) setScroll(s int) {
	h.items[len(h.items)-1].Scroll = s
}

type Item struct {
	*url.URL
	Scroll int
}
