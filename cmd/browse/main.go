package main

import (
	"flag"
	"fmt"

	"os"
	"opossum"
	"opossum/browser"
	"opossum/domino"
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
	logger.Quiet = flag.Bool("quiet", defaultQuietActive, "don't print info messages and non-fatal errors")
}

func Main() (err error) {
	dui, err = duit.NewDUI("opossum", nil) // TODO: rm global var
	if err != nil {
		return fmt.Errorf("new dui: %w", err)
	}

	style.Init(dui, log)

	w := dui.Display.Windows.Bounds().Dx()
	log.Printf("w=%v", w)
	log.Printf("w'=%v", dui.Scale(w))
	log.Printf("kid=%v", dui.Top.R)
	browser.SetLogger(log)
	opossum.SetLogger(log)
	nodes.SetLogger(log)
	b := browser.NewBrowser(dui, *startPage)

	dui.Top.UI = &duit.Box{
		Kids: duit.NewKids(
			&duit.Grid{
				Columns: 2,
				Padding: duit.NSpace(2, duit.SpaceXY(5, 3)),
				Halign:  []duit.Halign{duit.HalignLeft, duit.HalignRight},
				Valign:  []duit.Valign{duit.ValignMiddle, duit.ValignMiddle},
				Kids: duit.NewKids(
					&duit.Button{
						Text:  "Load",
						Font:  browser.Style.Font(),
						Click: b.LoadUrl,
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
	dui.Render()
	log.Printf("Rendering done")

	for {
		select {
		case e := <-dui.Inputs:
			dui.Input(e)
			//log.Printf("e=%+v", e)

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
	os.Chdir("../..")
	log = logger.Log
	log.Debug = *dbg
	style.CssFonts = *cssFonts
	style.ExperimentalUseBoxBackgrounds = *experimentalUseBoxBackgrounds
	if err := Main(); err != nil {
		log.Fatalf("Main: %v", err)
	}
}
