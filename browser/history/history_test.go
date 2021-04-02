package history

import (
	"net/url"
	"testing"
)

func TestNoDup(t *testing.T) {
	h := History{}
	uris := []string{
		"https://example.com",
		"https://example.com/a",
		"https://example.com/a",
		"https://example.com/b",
		"https://example.com/b",
	}
	for _, uri := range uris {
		u, err := url.Parse(uri)
		if err != nil {
			t.Error()
		}
		h.Push(u, 0)
	}
	if len(h.items) != 3 {
		t.Error()
	}
}
