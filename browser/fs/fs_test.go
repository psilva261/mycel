package fs

import (
	"bufio"
	"io"
	"net"
	"testing"
)

func TestQuery(t *testing.T) {
	fs := New()
	c1, c2 := net.Pipe()
	go fs.query(c2)
	w := bufio.NewWriter(c1)
	w.WriteString("echo\n")
	w.Flush()
	bs, err := io.ReadAll(c1)
	if string(bs) != "" || err != nil {
		t.Fail()
	}
}
