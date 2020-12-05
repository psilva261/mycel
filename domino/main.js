
	    global = {};
	    //global.__domino_frozen__ = true; // Must precede any require('domino')
	    var domino = require('domino-lib/index');
	    var Element = domino.impl.Element; // etc

	    // JSDOM also knows the style tag
	    // https://github.com/jsdom/jsdom/issues/2485
		Object.assign(this, domino.createWindow(s.html, 'http://example.com'));
		window = this;
		window.parent = window;
		window.top = window;
		window.self = window;
		addEventListener = function() {};
		window.location.href = 'http://example.com';
		navigator = {};
		HTMLElement = domino.impl.HTMLElement;
	    // Fire DOMContentLoaded
	    // to trigger $(document)readfy!!!!!!!
	    document.close();
	
	console.log('Hello!!')
	