package browser

import (
	"github.com/mjl-/duit"
	"golang.org/x/net/html"
	"io/ioutil"
	"net/url"
	"opossum"
	"opossum/domino"
	"opossum/nodes"
	"opossum/style"
	"strings"
)

type Website struct {
	duit.UI
	html      string
}

func (w *Website) layout(f opossum.Fetcher) {
	pass := func(htm string, csss ...string) (*html.Node, map[*html.Node]style.Map) {

		if debugPrintHtml {
			log.Printf("%v\n", htm)
		}

		var doc *html.Node
		var err error
		doc, err = html.ParseWithOptions(
			strings.NewReader(htm),
			html.ParseOptionEnableScripting(*ExperimentalJsInsecure),
		)
		if err != nil {
			panic(err.Error())
		}

		log.Printf("Retrieving CSS Rules...")
		var cssSize int
		nodeMap := make(map[*html.Node]style.Map)
		for i, css := range csss {

			log.Printf("CSS size %v kB", cssSize/1024)

			nm, err := style.FetchNodeMap(doc, css, 1280)
			if err == nil {
				log.Printf("[%v/%v] Fetch CSS Rules successful!", i+1, len(csss))
				if debugPrintHtml {
					log.Printf("%v", nm)
				}
				style.MergeNodeMaps(nodeMap, nm)
			} else {
				log.Errorf("Fetch CSS Rules failed: %v", err)
				if *DebugDumpCSS {
					ioutil.WriteFile("info.css", []byte(css), 0644)
				}
			}
		}

		return doc, nodeMap
	}

	log.Printf("1st pass")
	doc, _ := pass(w.html)

	log.Printf("2nd pass")
	log.Printf("Download style...")
	cssHrefs := style.Hrefs(doc)
	csss := make([]string, 0, len(cssHrefs))
	for _, href := range cssHrefs {
		url, err := f.LinkedUrl(href)
		if err != nil {
			log.Printf("error parsing %v", href)
			continue
		}
		log.Printf("Download %v", url)
		buf, contentType, err := f.Get(url)
		if err != nil {
			log.Printf("error downloading %v", url)
			continue
		}
		if contentType.IsCSS() {
			csss = append(csss, string(buf))
		} else {
			log.Printf("css: unexpected %v", contentType)
		}
	}
	csss = append([]string{ /*string(revertCSS), */ style.AddOnCSS}, csss...)
	doc, nodeMap := pass(w.html, csss...)

	if *ExperimentalJsInsecure {
		log.Printf("3rd pass")
		nt := nodes.NewNodeTree(doc, style.Map{}, nodeMap, nil)
		jsSrcs := domino.Srcs(nt)
		downloads := make(map[string]string)
		for _, src := range jsSrcs {
			url, err := f.LinkedUrl(src)
			if err != nil {
				log.Printf("error parsing %v", src)
				continue
			}
			log.Printf("Download %v", url)
			buf, _, err := f.Get(url)
			if err != nil {
				log.Printf("error downloading %v", url)
				continue
			}
			downloads[src] = string(buf)
		}
		codes := domino.Scripts(nt, downloads)
		log.Infof("JS pipeline start")
		jsProcessed, err := processJS2(w.html, nt, codes)
		if err == nil {
			if w.html != jsProcessed {
				log.Infof("html changed")
			}
			w.html = jsProcessed
			if debugPrintHtml {
				log.Printf("%v\n", jsProcessed)
			}
			doc, nodeMap = pass(w.html, csss...)
		} else {
			log.Errorf("JS error: %v", err)
		}
		log.Infof("JS pipeline end")
	}
	var countHtmlNodes func(*html.Node) int
	countHtmlNodes = func(n *html.Node) (num int) {
		num++
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			num += countHtmlNodes(c)
		}
		return
	}
	log.Printf("%v html nodes found...", countHtmlNodes(doc))

	body := grepBody(doc)

	log.Printf("Layout website...")
	numElements = 0
	scroller = duit.NewScroll(
		NodeToBox(0, browser, nodes.NewNodeTree(body, style.Map{}, nodeMap, &nodes.Node{})),
	)
	w.UI = scroller
	log.Printf("Layouting done (%v elements created)", numElements)
	if numElements < 10 {
		log.Errorf("Less than 10 elements layouted, seems css processing failed. Will layout without css")
		scroller = duit.NewScroll(
			NodeToBox(0, browser, nodes.NewNodeTree(body, style.Map{}, make(map[*html.Node]style.Map), nil)),
		)
		w.UI = scroller
	}
	log.Flush()
}

func formData(n html.Node) (data url.Values) {
	data = make(url.Values)
	if n.Data == "input" {
		if k := attr(n, "name"); k != "" {
			data.Set(k, attr(n, "value"))
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		for k, vs := range formData(*c) {
			data.Set(k, vs[0]) // TODO: what aboot the rest?
		}
	}
	return
}

func (b *Browser) submit(form *html.Node) {
	var err error
	method := "GET" // TODO
	if m := attr(*form, "method"); m != "" {
		method = strings.ToUpper(m)
	}
	uri := b.URL()
	log.Printf("form = %+v", form)
	if action := attr(*form, "action"); action != "" {
		uri, err = b.LinkedUrl(action)
		if err != nil {
			log.Printf("error parsing %v", action)
			return
		}
	}
	var buf []byte
	var contentType opossum.ContentType
	if method == "GET" {
		q := uri.Query()
		for k, vs := range formData(*form) {
			log.Printf("add query info %v => %v", k, vs[0])
			q.Set(k, vs[0]) // TODO: what is with the rest?
		}
		uri.RawQuery = q.Encode()
		log.Printf("uri raw query=%v", uri.RawQuery)
		buf, contentType, err = b.get(uri, true)
		log.Printf("uri=%v", uri.String())
	} else {
		buf, contentType, err = b.PostForm(uri, formData(*form))
	}
	if err == nil {
		if contentType.IsHTML() {
			b.render(buf)
		} else {
			log.Fatalf("post: unexpected %v", contentType)
		}
	} else {
		log.Printf("post form: %v", err)
	}
}
