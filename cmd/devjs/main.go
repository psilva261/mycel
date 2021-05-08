// js package as separate program (very wip)
package main

import (
	"encoding/json"
	"fmt"
	"github.com/psilva261/opossum/domino"
	"github.com/psilva261/opossum/domino/jsfcall"
	"github.com/psilva261/opossum/logger"
	"io"
	"os"
	"strings"
)

var (
	d *domino.Domino
	log *logger.Logger
)

func init() {
	f := false
	t := true
	domino.DebugDumpJS = &t
	logger.Quiet = &f
	logger.Init()
	log = &logger.Logger{Debug: true}
	domino.SetLogger(log)
}

func usage() {
	log.Printf("usage: devjs -h htmlfile jsfile1 [jsfile2 [..]]")
	os.Exit(1)
}

func Main(r io.Reader, w io.Writer, htm string, js []string) error {
	msg := jsfcall.Msg{}
	dec := json.NewDecoder(r)
	enc := json.NewEncoder(w)

	for {
		err := dec.Decode(&msg)
		if err == io.EOF {
			if d != nil {
				// TODO: probably that's not even necessary
				d.Stop()
			}
			break
		}
		if err != nil {
			return fmt.Errorf("decode: %v", err)
		}

		switch msg.Type {
		case jsfcall.Tinit:
			log.Printf("received Tinit")
			log.Printf("htm=%v", htm)
			d = domino.NewDomino(htm, nil, nil)
			d.Start()
			initialized := false
			for _, s := range js {
				if _, err := d.Exec/*6*/(s, !initialized); err != nil {
					if strings.Contains(err.Error(), "halt at") {
						return fmt.Errorf("execution halted: %w", err)
					}
					log.Printf("exec <script>: %v", err)
				}
				initialized = true
			}
			if err := d.CloseDoc(); err != nil {
				return fmt.Errorf("close doc: %w", err)
			}
			resHtm, changed, err := d.TrackChanges()
			if err == nil {
				log.Printf("processJS: changed = %v", changed)
			} else {
				return fmt.Errorf("track changes: %w", err)
			}
			msg := &jsfcall.Msg{
				Type: jsfcall.Rinit,
				Html: resHtm,
			}
			if err := enc.Encode(&msg); err != nil {
				return fmt.Errorf("encode: %v", err)
			}
		default:
			return fmt.Errorf("unhandled msg: %+v", msg)
		}
	}

	return nil
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
	}

	htmlfile := ""
	jsfiles := make([]string, 0, len(args))

	for len(args) > 0 {
		switch args[0] {
		case "-h":
			htmlfile, args = args[1], args[2:]
		default:
			var jsfile string
			jsfile, args = args[0], args[1:]
			jsfiles = append(jsfiles, jsfile)
		}
	}

	htm := ""
	js := make([]string, 0, len(jsfiles))
	if htmlfile != "" {
		b, err := os.ReadFile(htmlfile)
		if err != nil {
			log.Fatalf(err.Error())
		}
		htm = string(b)
	}
	for _, jsfile := range jsfiles {
		b, err := os.ReadFile(jsfile)
		if err != nil {
			log.Fatalf(err.Error())
		}
		js = append(js, string(b))
	}

	if err := Main(os.Stdin, os.Stdout, htm, js); err != nil {
		log.Fatalf("Main: %+v", err)
	}
}
