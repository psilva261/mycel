package domino

import (
	"github.com/psilva261/opossum/logger"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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

func handler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "<html><body>Hello World!</body></html>")
}

func xhr(req *http.Request) (resp *http.Response, err error) {
	w := httptest.NewRecorder()
	handler(w, req)
	resp = w.Result()
	return resp, nil
}

func TestXMLHttpRequest(t *testing.T) {
	d := NewDomino(simpleHTML, xhr, nil)
	d.Start()
	script := `
		var oReq = new XMLHttpRequest();
		var loaded = false;
		oReq.addEventListener("load", function() {
			loaded = true;
		});
		oReq.open("GET", "http://www.example.org/example.txt");
		oReq.send();
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
	buf, err := ioutil.ReadFile("../../../js/jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	d := NewDomino(simpleHTML, xhr, nil)
	d.Start()
	script := `
	var res;
	$.ajax({
		url: '/',
		success: function() {
			res = 'success';
		},
		error: function() {
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
	buf, err := ioutil.ReadFile("../../../js/godoc/jquery-1.8.2.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	d := NewDomino(simpleHTML, xhr, nil)
	d.Start()
	script := `
	var res;
	$.ajax({
		url: '/',
		success: function() {
			res = 'success';
		},
		error: function() {
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

func TestNoJsCompatComment(t *testing.T) {
	d := NewDomino(simpleHTML, nil, nil)
	d.Start()
	script := `

<!-- This is an actual comment

	''.replace(/^\s*<!--/g, '');
	const a = 1;
	a + 7;
-->
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

func TestJQuery(t *testing.T) {
	buf, err := ioutil.ReadFile("../../../js/jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	d := NewDomino(simpleHTML, nil, nil)
	d.Start()
	script := `
	$(document).ready(function() {
		undefinedExpr
	});
	setTimeout(function() {
		console.log("ok");
	}, 1000);
	var a = 1;
	`
	_, err = d.Exec(string(buf) + ";" + script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	res, err := d.Exec("a+1", false)
	t.Logf("res=%v", res)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if res != "2" {
		t.Fatal()
	}
	d.Stop()
}

func TestJQueryCss(t *testing.T) {
	buf, err := ioutil.ReadFile("../../../js/jquery-3.5.1.js")
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
	q := func(sel, prop string) (val string, err error) {
		if sel != "HTML > :nth-child(2) > :nth-child(1)" {
			panic(sel)
		}
		if prop != "display" {
			panic(prop)
		}
		return "inline-block", nil
	}
	d := NewDomino(h, nil, q)
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
	buf, err := ioutil.ReadFile("../../../js/godoc/pkg.html")
	if err != nil {
		t.Fatalf("%v", err)
	}
	d := NewDomino(string(buf), nil, nil)
	d.Start()
	for i, fn := range []string{"initfuncs.js", "jquery-1.8.2.js", "goversion.js", "godocs.js"} {
		buf, err := ioutil.ReadFile("../../../js/godoc/"+fn)
		if err != nil {
			t.Fatalf("%v", err)
		}
		_, err = d.Exec(string(buf), i == 0)
		if err != nil {
			t.Fatalf("%v", err)
		}
	}
	d.Stop()
}

func TestGoplayground(t *testing.T) {
	buf, err := ioutil.ReadFile("../../../js/godoc/golang.html")
	if err != nil {
		t.Fatalf("%v", err)
	}
	d := NewDomino(string(buf), nil, nil)
	d.Start()
	for i, fn := range []string{"initfuncs.js", "jquery-1.8.2.js", "playground.js", "goversion.js", "godocs.js", "golang.js"} {
		buf, err := ioutil.ReadFile("../../../js/godoc/"+fn)
		if err != nil {
			t.Fatalf("%v", err)
		}
		_, err = d.Exec(string(buf), i == 0)
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

	d.Stop()
}

func TestJqueryUI(t *testing.T) {
	buf, err := ioutil.ReadFile("../../../js/jqueryui/tabs.html")
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
		buf, err := ioutil.ReadFile("../../../js/jqueryui/"+fn)
		if err != nil {
			t.Fatalf("%v", err)
		}
		_, err = d.Exec(string(buf), i == 0)
		if err != nil {
			t.Fatalf("%v", err)
		}
	}
	d.Stop()
}

func TestRun(t *testing.T) {
	jQuery, err := ioutil.ReadFile("../../../js/jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	SCRIPT := string(jQuery) + `
	setTimeout(function() {
		var h = document.querySelector('html');
    	console.log(h.innerHTML);
	}, 1000);
	Object.assign(this, window);
    $(document).ready(function() {
    	console.log('READDDYYYYY!!!!!!!');
    });
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
	jQuery, err := ioutil.ReadFile("../../../js/jquery-3.5.1.js")
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
	d.CloseDoc()

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
	jQuery, err := ioutil.ReadFile("../../../js/jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	//t.Parallel()
	SCRIPT := string(jQuery) + `
	setTimeout(function() {
		var h = document.querySelector('html');
    	console.log(h.innerHTML);
	}, 1000);
	Object.assign(this, window);
    $(document).ready(function() {
    	console.log('READDDYYYYY!!!!!!!');
    });
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
	_=res
	res, err = d.Exec("$('h1').html('minor updates :-)'); $('h1').html();", false)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("new res=%v", res)
	d.Stop()
}

func TestMutationEvents(t *testing.T) {
	buf, err := ioutil.ReadFile("../../../js/jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	q := func(sel, prop string) (val string, err error) {
		if sel != "HTML > :nth-child(2) > :nth-child(1)" {
			panic(sel)
		}
		if prop != "display" {
			panic(prop)
		}
		return "inline-block", nil
	}
	d := NewDomino(simpleHTML, nil, q)
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
