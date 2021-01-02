package main

import (
	"flag"
	"fmt"

	"os"
	"opossum"
	"opossum/browser"
	"opossum/domino"
	"opossum/img"
	"opossum/logger"
	"opossum/style"
	"opossum/nodes"
	"runtime/pprof"
	"time"

	"github.com/mjl-/duit"
)

const debugPrintHtml = false

var dui *duit.DUI
var log *logger.Logger

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var cssFonts = flag.Bool("cssFonts", true, "toggle css fonts (default true)")
var experimentalUseBoxBackgrounds = flag.Bool("experimentalUseBoxBackgrounds", true, "show box BGs (default true)")
var startPage = flag.String("startPage", "http://9p.io", "")
var dbg = flag.Bool("debug", false, "show debug logs")

func init() {
	browser.DebugDumpCSS = flag.Bool("debugDumpCSS", false, "write css to info.css")
	domino.DebugDumpJS = flag.Bool("debugDumpJS", false, "write js to main.js")
	browser.ExperimentalJsInsecure = flag.Bool("experimentalJsInsecure", false, "DO NOT ACTIVATE UNLESS INSTRUCTED OTHERWISE")
	browser.EnableNoScriptTag = flag.Bool("enableNoScriptTag", false, "enable noscript tag")
	logger.Quiet = flag.Bool("quiet", defaultQuietActive, "don't print info messages and non-fatal errors")
}

func render(b *browser.Browser) {
	dui.Top.UI = &duit.Box{
		Kids: duit.NewKids(
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
		),
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
	
	f := &duit.Field{
		Text: value,
	}

	dui.Top.UI = &duit.Box{
		Kids: duit.NewKids(
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
		),
	}
	log.Printf("Render.....")
	dui.MarkLayout(dui.Top.UI)
	dui.MarkDraw(dui.Top.UI)
	dui.Render()
	log.Printf("Rendering done")

	return res
}

func Main() (err error) {
	dui, err = duit.NewDUI("opossum", nil) // TODO: rm global var
	if err != nil {
		return fmt.Errorf("new dui: %w", err)
	}

	style.Init(dui, log)
	browser.SetLogger(log)
	domino.SetLogger(log)
	img.SetLogger(log)
	opossum.SetLogger(log)
	nodes.SetLogger(log)

	b := browser.NewBrowser(dui, *startPage)
	b.Download = func(done chan int) chan string {
		go func() {
			<-done
			render(b)
		}()
		return confirm(b, fmt.Sprintf("Download %v", b.URL()), "/download.file")
	}
	render(b)

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
	style.CssFonts = *cssFonts
	style.ExperimentalUseBoxBackgrounds = *experimentalUseBoxBackgrounds

	if err := Main(); err != nil {
		log.Fatalf("Main: %v", err)
	}
}
