global = {};
//global.__domino_frozen__ = true; // Must precede any require('domino')
var domino = require('domino-lib/index');

Object.assign(this, domino.createWindow(opossum.html, 'http://example.com'));
window = this;
window.self = window;
window.parent = window;
window.top = window;
window.history = {
	replaceState: function() {}
};
window.location.href = 'http://example.com';
window.screen = {
	width: 1280,
	height: 1024
};
window.screenX = 0;
window.screenY = 25;
window.getComputedStyle = function(el, pseudo) {
	this.el = el;
	this.getPropertyValue = function(prop) {
		return opossum.style(___fq('', el), pseudo, prop, arguments[2]);
	};
	return this;
};
(function() {
	var utils = require('domino-lib/utils');
	utils.merge(window, domino.impl);
})()

location = window.location;
navigator = {
	platform: 'plan9(port)',
	userAgent: 'opossum'
};
Element.prototype.getClientRects = function() { /* I'm a stub */ return []; }

document._setMutationHandler(function(a) {
	// a provides attributes type, target and node or attr
	// (cf Object.keys(a))
	opossum.mutated(a.type, ___fq('yolo', a.target));
});

addEventListener = function() {};
removeEventListener = function() {};

function XMLHttpRequest() {
	var _method, _uri;
	var h = {};
	var ls = {};

	this.readyState = 0;

	var cb = function(data, err) {
		if (data !== '') {
			this.responseText = data;
			this.readyState = 4;
			this.state = 200;
			this.status = 200;
			if (ls['load']) ls['load'].bind(this)();
			if (this.onload) this.onload.bind(this)();

			if (this.onreadystatechange) this.onreadystatechange.bind(this)();
		}
	}.bind(this);

	this.addEventListener = function(k, fn) {
		ls[k] = fn;
	};
	this.open = function(method, uri) {
		_method = method;
		_uri = uri;
	};
	this.setRequestHeader = function(k, v) {
		h[k] = v;
	};
	this.send = function(data) {
		opossum.xhr(_method, _uri, h, data, cb);
		this.readyState = 2;
	};
	this.getAllResponseHeaders = function() {
		return '';
	};
}

var ___fq;
___fq = function(pre, el) {
	var i, p;

	if (!el) {
		return undefined;
	}
	p = el.parentElement;

	if (p) {
		for (i = 0; i < p.children.length; i++) {
			if (p.children[i] === el) {
				return ___fq('', p) + ' > :nth-child(' + (i+1) + ')';
			}
		}
	} else {
		return el.tagName;
	}
};
