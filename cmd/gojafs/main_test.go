package main

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"strings"
	"testing"
)

func call(fn, cmd string, args ...string) (resp string, err error) {
	conn, rwc := net.Pipe()
	go ctl(conn)
	defer rwc.Close()
	rwc.Write([]byte(cmd + "\n"))
	for _, arg := range args {
		rwc.Write([]byte(arg + "\n"))
	}
	r := bufio.NewReader(rwc)
	b := bytes.NewBuffer([]byte{})
	_, err = io.Copy(b, r)
	/*if !strings.Contains(err.Error(), io.ErrClosedPipe.Error()) {
		return "", fmt.Errorf("unexpected error: %v", err)
	}*/
	return b.String(), nil
}

func TestMain(t *testing.T) {
	htm = "<html><h1 id=title>hello</h1></html>"
	js = []string{
		"document.getElementById('title').innerHTML='world'",
	}
	t.Logf("call start...")
	resp, err := call("ctl", "start")
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("%v", resp)
	if !strings.Contains(resp, `<h1 id="title">world</h1>`) {
		t.Fail()
	}
}

func TestClick(t *testing.T) {
	htm = "<html><h1 id=title>hello</h1></html>"
	js = []string{
		`var c = 1;
		document.getElementById('title').addEventListener('click', function(event) {
			c = 3;
		});`,
	}
	_, err := call("ctl", "start")
	if err != nil {
		t.Fatalf("%v", err)
	}
	_, err = call("ctl", "click", "#title")
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
