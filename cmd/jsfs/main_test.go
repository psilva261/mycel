package main

import (
	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
	"bytes"
	"io"
	"bufio"
	"net"
	"os/user"
	"strings"
	"testing"
)

func TestMain(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Fatalf("%v", err)
	}
	un := u.Username
	c1, c2 := net.Pipe()
	err = Main(c1, c1)
	if err != nil {
		t.Fatalf("%v", err)
	}
	conn, err := client.NewConn(c2)
	if err != nil {
		t.Fatalf("%v", err)
	}
	fsys, err := conn.Attach(nil, un, "")
	if err != nil {
		t.Fatalf("%v", err)
	}
	d, err := fsys.Stat("ctl")
	if err != nil {
		t.Fatalf("%v", err)
	}
	if d.Name != "ctl" {
		t.Fail()
	}
	htm = "<html><h1 id=title>hello</h1></html>"
	js = []string{
		"document.getElementById('title').innerHTML='world'",
	}
	fid, err := fsys.Open("ctl", plan9.ORDWR)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer fid.Close()
	fid.Write([]byte("start\n"))
	r := bufio.NewReader(fid)
	b := bytes.NewBuffer([]byte{})
	_, err = io.Copy(b, r)
	if !strings.Contains(err.Error(), io.ErrClosedPipe.Error()) {
		t.Fatalf("%+v", err)
	}
	t.Logf("%v", b.String())
	if !strings.Contains(b.String(), `<h1 id="title">world</h1>`) {
		t.Fail()
	}
}
