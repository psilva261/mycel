package main

import (
	"9fans.net/go/draw"
	"fmt"
	"github.com/mjl-/duit"
	"github.com/psilva261/opossum/browser"
	"github.com/psilva261/opossum/js"
	"github.com/psilva261/opossum/logger"
	"github.com/psilva261/opossum/style"
	"image"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"
)

var (
	dui        *duit.DUI
	b          *browser.Browser
	cpuprofile string
	memprofile string
	loc        string = "http://9p.io"
	dbg        bool
	v          View
	Style      = style.Map{}
)

func init() {
	browser.EnableNoScriptTag = true
}

type View interface {
	Render() []*duit.Kid
}

type Nav struct {
	LocationField *duit.Field
	StatusBar     *duit.Label
}

func NewNav() (n *Nav) {
	n = &Nav{
		StatusBar: &duit.Label{
			Text: "",
		},
	}
	n.LocationField = &duit.Field{
		Text: loc,
		Font: Style.Font(),
		Keys: n.keys,
	}
	return
}

func (n *Nav) keys(k rune, m draw.Mouse) (e duit.Event) {
	if k == browser.EnterKey && !b.Loading() {
		a := n.LocationField.Text
		if !strings.HasPrefix(strings.ToLower(a), "http") {
			a = "http://" + a
		}
		u, err := url.Parse(a)
		if err != nil {
			log.Errorf("parse url: %v", err)
			return
		}
		return b.LoadUrl(u)
	}
	return
}

func (n *Nav) Render() []*duit.Kid {
	uis := []duit.UI{
		&duit.Grid{
			Columns: 3,
			Halign:  []duit.Halign{duit.HalignLeft, duit.HalignLeft, duit.HalignRight},
			Valign:  []duit.Valign{duit.ValignMiddle, duit.ValignMiddle, duit.ValignMiddle},
			Kids: duit.NewKids(
				&duit.Button{
					Text:  "Back",
					Font:  browser.Style.Font(),
					Click: b.Back,
				},
				&duit.Button{
					Text:  "Stop",
					Font:  browser.Style.Font(),
					Click: func() duit.Event {
						b.Cancel()
						return duit.Event{
							Consumed: true,
						}
					},
				},
				&duit.Box{
					Kids: duit.NewKids(
						n.LocationField,
					),
				},
			),
		},
		n.StatusBar,
	}
	if b != nil {
		uis = append(uis, b.Website)
	}
	return duit.NewKids(uis...)
}

type Confirm struct {
	text  string
	value string
	res   chan *string
	done  bool
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
					Text: "Ok",
					Font: browser.Style.Font(),
					Click: func() (e duit.Event) {
						if c.done {
							return
						}
						s := f.Text
						c.res <- &s
						c.done = true
						e.Consumed = true
						v = NewNav()
						render()
						return
					},
				},
				&duit.Button{
					Text: "Abort",
					Font: browser.Style.Font(),
					Click: func() (e duit.Event) {
						if c.done {
							return
						}
						close(c.res)
						c.done = true
						e.Consumed = true
						v = NewNav()
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

func render() {
	white, err := dui.Display.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, 0xffffffff)
	if err != nil {
		log.Errorf("%v", err)
	}
	dui.Top.UI = &duit.Box{
		Kids:       v.Render(),
		Background: white,
	}
	if b != nil {
		browser.PrintTree(b.Website.UI)
	}
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
	resize()

	style.Init(dui)
	v = NewNav()
	render()

	b = &browser.Browser{
		LocCh: make(chan string, 10),
	}
	b = browser.NewBrowser(dui, loc)
	b.Download = func(res chan *string) {
		v = &Confirm{
			text:  fmt.Sprintf("Download %v", b.URL()),
			value: "/download.file",
			res:   res,
		}
		render()
		return
	}
	v = NewNav()
	render()

	for {
		select {
		case e := <-dui.Inputs:
			//log.Infof("e=%v", e)
			dui.Input(e)
			if e.Type == duit.InputResize {
				resize()
			}

		case loc = <-b.LocCh:
			log.Infof("loc=%v", loc)
			ue, err := url.QueryUnescape(loc)
			if err == nil {
				loc = ue
			} else {
				log.Errorf("unescape %v: %v", loc, err)
			}
			if nav, ok := v.(*Nav); ok {
				nav.LocationField.Text = loc
			}

		case msg := <-b.StatusCh:
			if nav, ok := v.(*Nav); ok {
				if msg == "" {
					nav.StatusBar.Text = ""
				} else {
					nav.StatusBar.Text += msg + "\n"
				}
				dui.MarkLayout(nav.StatusBar)
				dui.MarkDraw(nav.StatusBar)
				dui.Render()
			}

		case err, ok := <-dui.Error:
			//log.Infof("err=%v", err)
			if !ok {
				finalize()
				return nil
			}
			log.Printf("main: duit: %s\n", err)
		}
	}
}

func usage() {
	fmt.Printf("usage: opossum [-v|-vv] [-h] [-jsinsecure] [-cpu|-mem fn] [startPage]\n")
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
		case "-cpu":
			cpuprofile, args = args[1], args[2:]
		case "-mem":
			memprofile, args = args[1], args[2:]
		default:
			if len(args) > 1 {
				usage()
			}
			loc, args = args[0], args[1:]
		}
	}

	if quiet {
		log.SetQuiet()
	}

	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatalf("%v", err)
		}
		pprof.StartCPUProfile(f)
		go func() {
			<-time.After(time.Minute)
			pprof.StopCPUProfile()
			f.Close()
			os.Exit(2)
		}()
	}

	if memprofile != "" {
		f, err := os.Create(memprofile)
		if err != nil {
			log.Fatalf("%v", err)
		}
		go func() {
			<-time.After(time.Minute)
			runtime.GC()
			pprof.WriteHeapProfile(f)
			f.Close()
			os.Exit(2)
		}()
	}

	log.Debug = dbg

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, os.Kill)

	go func() {
		<-done
		finalize()
	}()

	if err := Main(); err != nil {
		log.Fatalf("Main: %v", err)
	}
}

func finalize() {
	js.Stop()
	os.Exit(1)
}

func resize() {
	size := dui.Display.ScreenImage.R.Size()
	style.WindowWidth = size.X/dui.Scale(1)
	style.WindowHeight = size.Y/dui.Scale(1)
}
