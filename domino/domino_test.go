package domino

import (
	"io/ioutil"
	"github.com/psilva261/opossum"
	"github.com/psilva261/opossum/logger"
	"github.com/psilva261/opossum/nodes"
	"github.com/psilva261/opossum/style"
	"golang.org/x/net/html"
	"net/url"
	"strings"
	"testing"
	"time"
)

const simpleHTML = `
<html>
<body>
<h1 id="title">Hello</h1>
</body>
</html>
`

func init() {
	f := false
	t := true
	DebugDumpJS = &t
	logger.Quiet = &f
	logger.Init()
	log = &logger.Logger{Debug: true}
}

func TestSimple(t *testing.T) {
	d := NewDomino(simpleHTML, nil, nil)
	d.Start()
	s := `
	var state = 'empty';
	var a = 1;
	b = 2;
	`
	_, err := d.Exec(s, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	s2 := `
	(function() {
		if (state !== 'empty') throw new Exception(state);

		state = a + b;
	})()
	var a = 1;
	b = 2;
	`
	_, err = d.Exec(s2, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	d.Stop()
}

func TestGlobals(t *testing.T) {
	d := NewDomino(simpleHTML, nil, nil)
	d.Start()
}

func TestJQuery(t *testing.T) {
	buf, err := ioutil.ReadFile("jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	d := NewDomino(simpleHTML, nil, nil)
	d.Start()
	script := `
	$(document).ready(function() {
		gfgf
		console.log('yolo');
	});
	setTimeout(function() {
		console.log("ok");
	}, 1000);
	var numberOne = 1;
	`
	_=buf
	_, err = d.Exec(string(buf) + ";" + script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	res, err := d.Exec("numberOne+1", false)
	t.Logf("res=%v", res)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if res != "2" {
		t.Fatal()
	}
	d.Stop()
}

func TestJQueryHide(t *testing.T) {
	buf, err := ioutil.ReadFile("jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	d := NewDomino(simpleHTML, nil, nil)
	d.Start()
	script := `
	$(document).ready(function() {
		$('h1').hide();
	});
	`
	_, err = d.Exec(string(buf) + ";" + script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if err = d.CloseDoc(); err != nil {
		t.Fatalf("%v", err)
	}
	res, err := d.Exec("$('h1').attr('style')", false)
	t.Logf("res=%v", res)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if res != "display: none;" {
		t.Fatal()
	}
	d.Stop()
}

func TestJQueryCss(t *testing.T) {
	buf, err := ioutil.ReadFile("jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	h := `
	<html>
	<body>
	<h1 id="title" style="display: inline-block;">Hello</h1>
	</body>
	</html>
	`
	d := NewDomino(h, nil, nil)
	r := strings.NewReader(h)
	doc, err := html.Parse(r)
	if err != nil { t.Fatalf(err.Error()) }
	d.nt = nodes.NewNodeTree(doc, style.Map{}, make(map[*html.Node]style.Map), nil)
	d.Start()
	_, err = d.Exec(string(buf), true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	res, err := d.Exec("$('h1').css('display')", false)
	t.Logf("res=%v", res)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if res != "inline-block" {
		t.Fatal()
	}
	d.Stop()
}

func TestGodoc(t *testing.T) {
	buf, err := ioutil.ReadFile("godoc/pkg.html")
	if err != nil {
		t.Fatalf("%v", err)
	}
	d := NewDomino(string(buf), nil, nil)
	d.Start()
	for i, fn := range []string{"initfuncs.js", "jquery-1.8.2.js", "goversion.js", "godocs.js"} {
		buf, err := ioutil.ReadFile("godoc/"+fn)
		if err != nil {
			t.Fatalf("%v", err)
		}
		_, err = d.Exec(string(buf) /*+ ";" + script*/, i == 0)
		if err != nil {
			t.Fatalf("%v", err)
		}
	}
	d.Stop()
}

func TestGoplayground(t *testing.T) {
	buf, err := ioutil.ReadFile("godoc/golang.html")
	if err != nil {
		t.Fatalf("%v", err)
	}
	d := NewDomino(string(buf), nil, nil)
	d.Start()
	for i, fn := range []string{"initfuncs.js", "jquery-1.8.2.js", "playground.js", "goversion.js", "godocs.js", "golang.js"} {
		buf, err := ioutil.ReadFile("godoc/"+fn)
		if err != nil {
			t.Fatalf("%v", err)
		}
		_, err = d.Exec(string(buf) /*+ ";" + script*/, i == 0)
		if err != nil {
			t.Fatalf("%v", err)
		}
	}
	res, err := d.Exec("window.playground", false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if !strings.Contains(res, "function playground(opts) {")  {
		t.Fatalf("%v", res)
	}
	if err = d.CloseDoc(); err != nil {
		t.Fatalf("%v", err)
	}

	d.Stop()
}

func TestJqueryUI(t *testing.T) {
	buf, err := ioutil.ReadFile("jqueryui/tabs.html")
	if err != nil {
		t.Fatalf("%v", err)
	}
	d := NewDomino(string(buf), nil, nil)
	d.Start()
	script := `
	Object.assign(this, window);
	`
	_ = script
	for i, fn := range []string{"jquery-1.12.4.js", "jquery-ui.js", "tabs.js"} {
		buf, err := ioutil.ReadFile("jqueryui/"+fn)
		if err != nil {
			t.Fatalf("%v", err)
		}
		_, err = d.Exec(string(buf) /*+ ";" + script*/, i == 0)
		if err != nil {
			t.Fatalf("%v", err)
		}
	}
	d.Stop()
}

func TestRun(t *testing.T) {
	jQuery, err := ioutil.ReadFile("jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	//t.Parallel()
	SCRIPT := string(jQuery) + `
    ;;;
	setTimeout(function() {
		console.log("ok :-)");
		console.log(s.buf);
		var h = document.querySelector('html');
    	console.log(h.innerHTML);
	}, 1000);
	console.log("Started");
	Object.assign(this, window);
    $(document).ready(function() {
    	console.log('READDDYYYYY!!!!!!!');
    });
    console.log('$:');
    console.log($);
    console.log('$h1:');
    console.log($('h1').html());

    //elem.dispatchEvent(event);
    console.log(window.location.href);
	`
	d := NewDomino(simpleHTML, nil, nil)
	d.Start()
	_, err = d.Exec(SCRIPT, true)
	if err != nil {
		t.Fatalf(err.Error())
	}

	res, err := d.Exec("$('h1').html()", false)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if res != "Hello" {
		t.Fatalf(res)
	}
	d.Stop()
}

func TestTriggerClick(t *testing.T) {
	jQuery, err := ioutil.ReadFile("jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	SCRIPT := string(jQuery) + `
	var clicked = false;
    $(document).ready(function() {
    	$('h1').click(function() {
    		clicked = true;
    	});
    });
	`
	d := NewDomino(simpleHTML, nil,  nil)
	d.Start()
	_, err = d.Exec(SCRIPT, true)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if err = d.CloseDoc(); err != nil {
		t.Fatalf("%v", err)
	}

	res, err := d.Exec("$('h1').html()", false)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if res != "Hello" {
		t.Fatalf(res)
	}

	if _, _, err = d.TrackChanges(); err != nil {
		t.Fatalf(err.Error())
	}
	_, changed, err := d.TriggerClick("h1")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if changed {
		t.Fatal()
	}
	res, err = d.Exec("clicked", false)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if res != "true" {
		t.Fatalf(res)
	}
	d.Stop()
}

func TestDomChanged(t *testing.T) {
	jQuery, err := ioutil.ReadFile("jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	//t.Parallel()
	SCRIPT := string(jQuery) + `
    ;;;
	setTimeout(function() {
		console.log("ok :-)");
		console.log(s.buf);
		var h = document.querySelector('html');
    	console.log(h.innerHTML);
	}, 1000);
	console.log("Started");
	Object.assign(this, window);
    $(document).ready(function() {
    	console.log('READDDYYYYY!!!!!!!');
    });
    console.log('$:');
    console.log($);
    console.log('$h1:');
    console.log($('h1').html());

    //elem.dispatchEvent(event);
    console.log(window.location.href);
	`
	d := NewDomino(simpleHTML, nil, nil)
	d.Start()
	_, err = d.Exec(SCRIPT, true)
	if err != nil {
		t.Fatalf(err.Error())
	}

	res, err := d.Exec("$('h1').html()", false)
	if err != nil {
		t.Fatalf(err.Error())
	}
	/*if res != "Hello" {
		t.Fatalf(res)
	}*/
	_=res
	res, err = d.Exec("$('h1').html('minor updates :-)'); $('h1').html();", false)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("new res=%v", res)
	d.Stop()
}

func TestTrackChanges(t *testing.T) {
	d := NewDomino(simpleHTML, nil, nil)
	d.Start()
	_, err := d.Exec(``, true)
	if err != nil {
		t.Fatalf(err.Error())
	}
	// 0th time: init
	if _, _, err = d.TrackChanges(); err != nil {
		t.Fatalf(err.Error())
	}
	// 1st time: no change
	html, changed, err := d.TrackChanges()
	if err != nil {
		t.Fatalf(err.Error())
	}
	if changed == true {
		t.Fatal()
	}
	// 2nd time: no change
	html, changed, err = d.TrackChanges()
	if err != nil {
		t.Fatalf(err.Error())
	}
	if changed == true {
		t.Fatal()
	}
	_, err = d.Exec("document.getElementById('title').innerHTML='new title'; true;", false)
	if err != nil {
		t.Fatalf(err.Error())
	}
	// 3rd time: yes change
	html, changed, err = d.TrackChanges()
	if err != nil {
		t.Fatalf(err.Error())
	}
	if html == "" {
		t.Fatalf(err.Error())
	}
	if changed == false {
		t.Fatal()
	}
	if !strings.Contains(html, "new title") {
		t.Fatalf(html)
	}
	d.Stop()
}

/*func TestExecInlinedScripts(t *testing.T) {
	const h = `
	<html>
	<body>
	<h1 id="title">Hello</h1>
	<script>
	document.getElementById('title').innerHTML = 'Good day';
	</script>
	</body>
	</html>
	`
	d := NewDomino(h)
	d.Start()
	err := d.ExecInlinedScripts()
	if err != nil {
		t.Fatalf(err.Error())
	}
	res, err := d.Export("document.getElementById('title').innerHTML")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if !strings.Contains(res, "Good day") {
		t.Fatalf(res)
	}
	d.Stop()
}*/

/*func TestWindowEqualsGlobal(t *testing.T) {
	const h = `
	<html>
	<body>
	<script>
	a = 2;
	window.b = 5;
	</script>
	<script>
	console.log('window.a=', window.a);
	console.log('wot');
	console.log('window.b=', window.b);
	console.log('wit');
	window.a++;
	b++;
	</script>
	</body>
	</html>
	`
	d := NewDomino(h)
	d.Start()
	err := d.ExecInlinedScripts()
	if err != nil {
		t.Fatalf(err.Error())
	}
	res, err := d.Export("window.a")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if !strings.Contains(res, "3") {
		t.Fatalf(res)
	}
	res, err = d.Export("window.b")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if !strings.Contains(res, "6") {
		t.Fatalf(res)
	}
	d.Stop()
}*/

func TestES6(t *testing.T) {
	d := NewDomino(simpleHTML, nil,  nil)
	d.Start()
	script := `
	var foo = function(data={}) {}
	var h = {
		a: 1,
		b: 11
	};
	var {a, b} = h;
	`
	_, err := d.Exec6(script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	res, err := d.Exec("a+b", false)
	t.Logf("res=%v", res)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if res != "12" {
		t.Fatal()
	}
	d.Stop()
}

func TestWindowParent(t *testing.T) {
	d := NewDomino(simpleHTML, nil, nil)
	d.Start()
	script := `
	console.log('Hello!!')
	`
	_, err := d.Exec(script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	res, err := d.Exec("window === window.parent", false)
	t.Logf("res=%v", res)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if res != "true" {
		t.Fatal()
	}
	d.Stop()
}

func TestReferrer(t *testing.T) {
	d := NewDomino(simpleHTML, nil, nil)
	d.Start()
	script := `
	document.referrer;
	`
	res, err := d.Exec(script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("res=%v", res)
	if res != "https://example.com" {
		t.Fatal()
	}
	d.Stop()
}

type MockBrowser struct {
	origin *url.URL
	linkedUrl *url.URL
}

func (mb *MockBrowser) LinkedUrl(string) (*url.URL, error) {
	return mb.linkedUrl, nil
}

func (mb *MockBrowser) Origin() (*url.URL) {
	return mb.origin
}

func (mb *MockBrowser) Get(*url.URL) (bs []byte, ct opossum.ContentType, err error) {
	return
}

func TestXMLHttpRequest(t *testing.T) {
	mb := &MockBrowser{}
	mb.origin, _ = url.Parse("https://example.com")
	mb.linkedUrl, _ = url.Parse("https://example.com")
	d := NewDomino(simpleHTML, mb, nil)
	d.Start()
	script := `
		var oReq = new XMLHttpRequest();
		var loaded = false;
		oReq.addEventListener("load", function() {
			console.log('loaded!!!!! !!! 11!!!1!!elf!!!1!');
			loaded = true;
		});
		console.log(oReq.open);
		console.log('open:');
		oReq.open("GET", "http://www.example.org/example.txt");
		console.log('send:');
		oReq.send();
		console.log('return:');
	`
	_, err := d.Exec(script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	<-time.After(time.Second)
	res, err := d.Exec("oReq.responseText;", false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("res=%v", res)
	if !strings.Contains(res, "<html") {
		t.Fatal()
	}
	d.Stop()
}

func TestJQueryAjax(t *testing.T) {
	mb := &MockBrowser{}
	mb.origin, _ = url.Parse("https://example.com")
	mb.linkedUrl, _ = url.Parse("https://example.com")
	buf, err := ioutil.ReadFile("jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	d := NewDomino(simpleHTML, mb, nil)
	d.Start()
	script := `
	var res;
	$.ajax({
		url: '/',
		success: function() {
			console.log('success!!!');
			res = 'success';
		},
		error: function() {
			console.log('error!!!');
			res = 'err';
		}
	});
	`
	_, err = d.Exec(string(buf) + ";" + script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if err = d.CloseDoc(); err != nil {
		t.Fatalf("%v", err)
	}
	<-time.After(time.Second)
	res, err := d.Exec("res;", false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("res=%v", res)
	if res != "success" {
		t.Fatalf(res)
	}
	d.Stop()
}

func TestJQueryAjax182(t *testing.T) {
	mb := &MockBrowser{}
	mb.origin, _ = url.Parse("https://example.com")
	mb.linkedUrl, _ = url.Parse("https://example.com")
	buf, err := ioutil.ReadFile("godoc/jquery-1.8.2.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	d := NewDomino(simpleHTML, mb, nil)
	d.Start()
	script := `
	var res;
	$.ajax({
		url: '/',
		success: function() {
			console.log('success!!!');
			res = 'success';
		},
		error: function() {
			console.log('error!!!');
			res = 'err';
		}
	});
	`
	_, err = d.Exec(string(buf) + ";" + script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if err = d.CloseDoc(); err != nil {
		t.Fatalf("%v", err)
	}
	<-time.After(5*time.Second)
	res, err := d.Exec("res;", false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("res=%v", res)
	if res != "success" {
		t.Fatalf(res)
	}
	d.Stop()
}

func TestMutationEvents(t *testing.T) {
	buf, err := ioutil.ReadFile("jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	d := NewDomino(simpleHTML, nil, nil)
	d.Start()
	script := `
	$('h1').hide();
	$('h1').show();
	`
	_, err = d.Exec(string(buf) + ";" + script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if err = d.CloseDoc(); err != nil {
		t.Fatalf("%v", err)
	}
	res, err := d.Exec("$('h1').attr('style')", false)
	t.Logf("res=%v", res)
	if err != nil {
		t.Fatalf("%v", err)
	}
	d.Stop()
}

func TestNoJsCompatComment(t *testing.T) {
	d := NewDomino(simpleHTML, nil, nil)
	d.Start()
	script := `
<!--
	const a = 1;
	a + 7;
// -->
	`
	res, err := d.Exec(script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("res=%v", res)
	if res != "8" {
		t.Fatal()
	}
	d.Stop()
}
