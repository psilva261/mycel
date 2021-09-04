package main

import (
	"9fans.net/go/draw"
	"fmt"
	"image"
	"os"
	"github.com/knusbaum/go9p"
	"github.com/psilva261/opossum/browser"
	"github.com/psilva261/opossum/js"
	"github.com/psilva261/opossum/logger"
	"github.com/psilva261/opossum/style"
	"os/signal"
	"runtime/pprof"
	"time"
	"github.com/mjl-/duit"
)


var (
	dui *duit.DUI
	b *browser.Browser
	cpuprofile string
	startPage string = "http://9p.io"
	dbg bool
	v View
)

func init() {
	browser.EnableNoScriptTag = true
}

type View interface {
	Render() []*duit.Kid
}

type Nav struct {}

func (n *Nav) Render() []*duit.Kid {
	return duit.NewKids(
		&duit.Grid{
			Columns: 2,
			Padding: duit.NSpace(2, duit.SpaceXY(5, 3)),
			Halign:  []duit.Halign{duit.HalignLeft, duit.HalignRight},
			Valign:  []duit.Valign{duit.ValignMiddle, duit.ValignMiddle},
			Kids: duit.NewKids(
				&duit.Button{
					Text:  "Back",
					Font:  browser.Style.Font(),
					Click: b.Back,
				},
				&duit.Box{
					Kids: duit.NewKids(
						b.LocationField,
					),
				},
			),
		},
		b.StatusBar,
		b.Website,
	)
}

type Confirm struct {
	text string
	value string
	res chan *string
	done bool
}

func (c *Confirm) Render() []*duit.Kid {
	f := &duit.Field{
		Text: c.value,
	}
	return duit.NewKids(
		&duit.Grid{
			Columns: 3,
			Padding: duit.NSpace(3, duit.SpaceXY(5, 3)),
			Halign:  []duit.Halign{duit.HalignLeft, duit.HalignLeft, duit.HalignRight},
			Valign:  []duit.Valign{duit.ValignMiddle, duit.ValignMiddle, duit.ValignMiddle},
			Kids: duit.NewKids(
				&duit.Button{
					Text:  "Ok",
					Font:  browser.Style.Font(),
					Click: func() (e duit.Event) {
						if c.done { return }
						s := f.Text
						c.res <- &s
						c.done = true
						e.Consumed = true
						v = &Nav{}
						render()
						return
					},
				},
				&duit.Button{
					Text:  "Abort",
					Font:  browser.Style.Font(),
					Click: func() (e duit.Event) {
						if c.done { return }
						close(c.res)
						c.done = true
						e.Consumed = true
						v = &Nav{}
						render()
						return
					},
				},
				f,
			),
		},
		&duit.Label{
			Text: c.text,
		},
	)
}

type Loading struct {}

func (l *Loading) Render() []*duit.Kid {
	return nil
}

func render() {
	white, err := dui.Display.AllocImage(image.Rect(0, 0, 10, 10), draw.ARGB32, true, 0xffffffff)
	if err != nil {
		log.Errorf("%v", err)
	}
	dui.Top.UI = &duit.Box{
		Kids: v.Render(),
		Background: white,
	}
	browser.PrintTree(b.Website.UI)
	log.Printf("Render.....")
	dui.MarkLayout(dui.Top.UI)
	dui.MarkDraw(dui.Top.UI)
	dui.Render()
	log.Printf("Rendering done")
}

func Main() (err error) {
	dui, err = duit.NewDUI("opossum", nil) // TODO: rm global var
	if err != nil {
		return fmt.Errorf("new dui: %w", err)
	}
	dui.Debug = dbg

	style.Init(dui)

	b = browser.NewBrowser(dui, startPage)
	b.Download = func(res chan *string) {
		v = &Confirm{
			text: fmt.Sprintf("Download %v", b.URL()),
			value: "/download.file",
			res: res,
		}
		render()
		return
	}
	v = &Nav{}
	render()

	for {
		select {
		case e := <-dui.Inputs:
			dui.Input(e)

		case err, ok := <-dui.Error:
			if !ok {
				return nil
			}
			log.Printf("main: duit: %s\n", err)
		}
	}
}

func usage() {
	fmt.Printf("usage: opossum [-v|-vv] [-h] [-jsinsecure] [-cpuprofile fn] [startPage]\n")
	os.Exit(1)
}

func main() {
	quiet := true
	args := os.Args[1:]
	for len(args) > 0 {
		switch args[0] {
		case "-vv":
			quiet = false
			dbg = true
			args = args[1:]
		case "-v":
			quiet = false
			args = args[1:]
		case "-h":
			usage()
			args = args[1:]
		case "-jsinsecure":
			browser.ExperimentalJsInsecure = true
			args = args[1:]
		case "-cpuprofile":
			cpuprofile, args = args[0], args[2:]
		default:
			if len(args) > 1 {
				usage()
			}
			startPage, args = args[0], args[1:]
		}
	}

	if quiet {
		log.SetQuiet()
	}

	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		go func() {
			<-time.After(time.Minute)
			pprof.StopCPUProfile()
			os.Exit(2)
		}()
	}

	log.Debug = dbg
	go9p.Verbose = log.Debug

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, os.Kill)

	go func() {
		<-done
		js.Stop()
		os.Exit(1)
	}()

	if err := Main(); err != nil {
		log.Fatalf("Main: %v", err)
	}
}
