package domino

import (
	"io/ioutil"
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
	t := true
	DebugDumpJS = &t
}

func TestSimple(t *testing.T) {
	d := NewDomino(simpleHTML)
	d.Start()
	script := `
	console.log('Hello!!');
	var numberOne = 1;
	`
	err := d.Exec(script)
	if err != nil {
		t.Fatalf("%v", err)
	}
	res, err := d.Export("numberOne+1")
	t.Logf("res=%v", res)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if res != "2" {
		t.Fatal()
	}
	d.Stop()
}

func TestJQuery(t *testing.T) {
	buf, err := ioutil.ReadFile("jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	d := NewDomino(simpleHTML)
	d.Start()
	script := `
	console.log('Hello!!');
	//console.log(window.jQuery);
	console.log(this);
	Object.assign(this, window);
	//console.log(this.jQuery);
	console.log($);
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
	err = d.Exec(string(buf) + ";" + script)
	if err != nil {
		t.Fatalf("%v", err)
	}
	res, err := d.Export("numberOne+1")
	t.Logf("res=%v", res)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if res != "2" {
		t.Fatal()
	}
	time.Sleep(2 * time.Second)
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
	d := NewDomino(simpleHTML)
	d.Start()
	err = d.Exec(SCRIPT)
	if err != nil {
		t.Fatalf(err.Error())
	}

	time.Sleep(2 * time.Second)
	res, err := d.Export("$('h1').html()")
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
	//t.Parallel()
	SCRIPT := string(jQuery) + `
    ;;;
    Object.assign(this, window);
	console.log("Started");
	var clicked = false;
    $(document).ready(function() {
    	console.log('READDDYYYYY!!!!!!!');
    	$('h1').click(function() {
    		console.log('CLICKED!!!!');
    		clicked = true;
    	});
    });
	`
	d := NewDomino(simpleHTML)
	d.Start()
	err = d.Exec(SCRIPT)
	if err != nil {
		t.Fatalf(err.Error())
	}

	//time.Sleep(2 * time.Second)
	res, err := d.Export("$('h1').html()")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if res != "Hello" {
		t.Fatalf(res)
	}
	_, ok, err := d.TriggerClick("h1")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if !ok {
		t.Fatal()
	}
	res, err = d.Export("clicked")
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
	d := NewDomino(simpleHTML)
	d.Start()
	err = d.Exec(SCRIPT)
	if err != nil {
		t.Fatalf(err.Error())
	}

	time.Sleep(2 * time.Second)
	res, err := d.Export("$('h1').html()")
	if err != nil {
		t.Fatalf(err.Error())
	}
	/*if res != "Hello" {
		t.Fatalf(res)
	}*/
	_=res
	res, err = d.Export("$('h1').html('minor updates :-)'); $('h1').html();")
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("new res=%v", res)
	<-time.After(2*time.Second)
	d.Stop()
}

func TestTrackChanges(t *testing.T) {
	d := NewDomino(simpleHTML)
	d.Start()
	err := d.Exec(``)
	if err != nil {
		t.Fatalf(err.Error())
	}
	// 1st time: no change
	html, changed, err := d.TrackChanges()
	if err != nil {
		t.Fatalf(err.Error())
	}
	if html == "" {
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
	if html == "" {
		t.Fatalf(err.Error())
	}
	if changed == true {
		t.Fatal()
	}
	_, err = d.Export("document.getElementById('title').innerHTML='new title'; true;")
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

func TestExecInlinedScripts(t *testing.T) {
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
}

func TestWindowEqualsGlobal(t *testing.T) {
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
}

func TestES6(t *testing.T) {
	d := NewDomino(simpleHTML)
	d.Start()
	script := `
	console.log('Hello!!');
	const numberOne = 1;
	`
	err := d.Exec6(script)
	if err != nil {
		t.Fatalf("%v", err)
	}
	res, err := d.Export("numberOne+1")
	t.Logf("res=%v", res)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if res != "2" {
		t.Fatal()
	}
	d.Stop()
}

func TestWindowParent(t *testing.T) {
	d := NewDomino(simpleHTML)
	d.Start()
	script := `
	console.log('Hello!!')
	`
	err := d.Exec(script)
	if err != nil {
		t.Fatalf("%v", err)
	}
	res, err := d.Export("window === window.parent")
	t.Logf("res=%v", res)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if res != "true" {
		t.Fatal()
	}
	d.Stop()
}
