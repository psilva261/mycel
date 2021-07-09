package main

import (
	"9fans.net/go/draw"
	"flag"
	"fmt"
	"image"
	"os"
	"github.com/knusbaum/go9p"
	"github.com/psilva261/opossum"
	"github.com/psilva261/opossum/browser"
	"github.com/psilva261/opossum/browser/fs"
	"github.com/psilva261/opossum/img"
	"github.com/psilva261/opossum/js"
	"github.com/psilva261/opossum/logger"
	"github.com/psilva261/opossum/style"
	"github.com/psilva261/opossum/nodes"
	"os/signal"
	"runtime/pprof"
	"time"
	"github.com/mjl-/duit"
)

const debugPrintHtml = false

var dui *duit.DUI
var log *logger.Logger

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var startPage = flag.String("startPage", "http://9p.io", "")
var dbg = flag.Bool("debug", false, "show debug logs")

func init() {
	browser.DebugDumpCSS = flag.Bool("debugDumpCSS", false, "write css to info.css")
	js.DebugDumpJS = flag.Bool("debugDumpJS", false, "write js to main.js")
	browser.ExperimentalJsInsecure = flag.Bool("experimentalJsInsecure", false, "DO NOT ACTIVATE UNLESS INSTRUCTED OTHERWISE")
	browser.EnableNoScriptTag = flag.Bool("enableNoScriptTag", false, "enable noscript tag")
	logger.Quiet = flag.Bool("quiet", defaultQuietActive, "don't print info messages and non-fatal errors")
}

func mainView(b *browser.Browser) []*duit.Kid {
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

func render(b *browser.Browser, kids []*duit.Kid) {
	white, err := dui.Display.AllocImage(image.Rect(0, 0, 10, 10), draw.ARGB32, true, 0xffffffff)
	if err != nil {
		log.Errorf("%v", err)
	}
	dui.Top.UI = &duit.Box{
		Kids: kids,
		Background: white,
	}
	browser.PrintTree(b.Website.UI)
	log.Printf("Render.....")
	dui.MarkLayout(dui.Top.UI)
	dui.MarkDraw(dui.Top.UI)
	dui.Render()
	log.Printf("Rendering done")
}

func confirm(b *browser.Browser, text, value string) chan string {
	res := make(chan string)

	dui.Call <- func() {
		f := &duit.Field{
			Text: value,
		}

		kids := duit.NewKids(
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
							res <- f.Text
							e.Consumed = true
							return
						},
					},
					&duit.Button{
						Text:  "Abort",
						Font:  browser.Style.Font(),
						Click: func() (e duit.Event) {
							res <- ""
							e.Consumed = true
							return
						},
					},
					f,
				),
			},
			&duit.Label{
				Text: text,
			},
		)

		render(b, kids)
	}

	return res
}

func Main() (err error) {
	dui, err = duit.NewDUI("opossum", nil) // TODO: rm global var
	if err != nil {
		return fmt.Errorf("new dui: %w", err)
	}

	style.Init(dui, log)
	browser.SetLogger(log)
	fs.SetLogger(log)
	img.SetLogger(log)
	js.SetLogger(log)
	opossum.SetLogger(log)
	nodes.SetLogger(log)

	b := browser.NewBrowser(dui, *startPage)
	b.Download = func(done chan int) chan string {
		go func() {
			<-done
			dui.Call <- func() {
				render(b, mainView(b))
			}
		}()
		return confirm(b, fmt.Sprintf("Download %v", b.URL()), "/download.file")
	}
	render(b, mainView(b))

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

func main() {
	flag.Parse()
	logger.Init()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
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

	log = logger.Log
	log.Debug = *dbg
	go9p.Verbose = log.Debug

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, os.Kill)

	go func() {
		<-done
		js.Stop()
	}()

	if err := Main(); err != nil {
		log.Fatalf("Main: %v", err)
	}
}
