package main

import (
	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
	"bytes"
	"fmt"
	"io"
	"bufio"
	"net"
	"os/user"
	"strings"
	"testing"
)

func connect() (fsys *client.Fsys, c io.Closer, err error) {
	u, err := user.Current()
	if err != nil {
		return
	}
	un := u.Username
	c1, c2 := net.Pipe()
	go func() {
		if err = Main(c1, c1); err != nil && err != io.EOF {
			panic(err.Error())
		}
	}()
	conn, err := client.NewConn(c2)
	if err != nil {
		return
	}
	fsys, err = conn.Attach(nil, un, "")
	if err != nil {
		return
	}
	return fsys, conn, nil
}

func call(fsys *client.Fsys, fn, cmd string, args... string) (resp string, err error) {
	fid, err := fsys.Open(fn, plan9.ORDWR)
	if err != nil {
		return
	}
	defer fid.Close()
	fid.Write([]byte(cmd+"\n"))
	for _, arg := range args {
		fid.Write([]byte(arg+"\n"))
	}
	r := bufio.NewReader(fid)
	b := bytes.NewBuffer([]byte{})
	_, err = io.Copy(b, r)
	if !strings.Contains(err.Error(), io.ErrClosedPipe.Error()) {
		return "", fmt.Errorf("unexpected error: %v", err)
	}
	return b.String(), nil
}

func TestMain(t *testing.T) {
	fsys, c, err := connect()
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer c.Close()
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
	resp, err := call(fsys, "ctl", "start")
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("%v", resp)
	if !strings.Contains(resp, `<h1 id="title">world</h1>`) {
		t.Fail()
	}
}


func TestClick(t *testing.T) {
	fsys, c, err := connect()
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer c.Close()
	htm = "<html><h1 id=title>hello</h1></html>"
	js = []string{
		`var c = 1;
		document.getElementById('title').addEventListener('click', function(event) {
			c = 3;
		});`,
	}
	_, err = call(fsys, "ctl", "start")
	if err != nil {
		t.Fatalf("%v", err)
	}
	_, err = call(fsys, "ctl", "click", "#title")
	if err != nil {
		t.Fatalf("%v", err)
	}
	resp, err := d.Exec("c", false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("%v", resp)
	if resp != "3" {
		t.Fail()
	}
}
