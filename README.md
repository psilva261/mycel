# Opossum Web Browser

Basic portable Web browser; only needs a Go compiler to compile, no C dependencies.

The UI is built with https://github.com/mjl-/duit

Still very experimental and most features are missing, here's a screenshot: http://psilva.sdf.org/scr.png

Supported features:

- rudimentary CSS/HTML5 support, large parts like float/flex layout are just stub implementations
- Server-side rendered websites
- Images (pre-loaded all at once though)
- TLS
- experimental JS/DOM without AJAX can be activated (basically script tags are evaluated)
- file downloads

# Install

## Plan 9

You can download a tarball with the binary at http://psilva.sdf.org/opossum-plan9-amd64.tgz

```
./opossum-plan9-amd64.bin
```

Also `/sys/lib/tls/ca.pem` needs to be present for TLS to work. ca certs can be downloaded from the curl homepage:

```
hget https://curl.haxx.se/ca/cacert.pem > /sys/lib/tls/ca.pem
```

To compile the source Go 1.15 is needed. Probably `$GOPROXY` should be set to `https://proxy.golang.org`

```
cd cmd/browse
go run .
```

There are various command line options, visible with `-h`, most importantly to see errors:

```
go run . '-quiet=false'
```

(`-quiet=false` produces a lot of output, consider turning on scroll since processing waits for that...)

or all messages:

```
go run . '-quiet=false' '-debug=true'
```

## macOS

Requirements:

- Go
- Plan9Port

```
cd cmd/browse
go run .
```

# JS support

Very experimental support for that. Mostly based on goja (ECMAScript 5.1) and github.com/fgnass/domino (DOM implementation in JS). Some sort of DOM diffing is needed, also AJAX functions, `getComputedStyle` etc. are either missing or stubs. Very simple jQuery based code works though, e.g. jQuery UI Tab view https://jqueryui.com/resources/demos/tabs/default.html or the toggle buttons on https://golang.org/pkg There is also highly experimental ES6 support with Babel.

Try on Plan 9 with e.g.:

```
go run . '-experimentalJsInsecure=true' -startPage https://jqueryui.com/resources/demos/tabs/default.html
```

or macOS etc.:

```
go run . -experimentalJsInsecure=true -startPage https://jqueryui.com/resources/demos/tabs/default.html
```


# TODO

- load images on the fly
- implement more parts of HTML5 and CSS
- create a widget for div/span
- clean up code, support webfs, snarf
