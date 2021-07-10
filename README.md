# Opossum Web Browser

Basic portable Web browser; only needs a Go compiler to compile.

The UI is built with https://github.com/mjl-/duit

Still experimental and a lot of features are missing.

Supported features:

- rudimentary HTML5 and CSS support, large parts like float/flex layout are just stub implementations
- Server-side rendered websites
- Images (pre-loaded all at once though)
- TLS
- experimental JS/DOM can be activated (very basic jQuery examples work)
- file downloads

# Install

## Plan 9

Setup TLS:

```
hget https://curl.haxx.se/ca/cacert.pem > /sys/lib/tls/ca.pem
```

### Binary

A recent binary for amd64 and 386 can be downloaded from http://psilva.sdf.org/opossum.html

### Compile from Source

```
go install ./cmd/opossum
```

There are various command line options, visible with `-h`, most importantly to see errors:

```
opossum '-quiet=false'
```

(`-quiet=false` produces a lot of output, consider turning on scroll since processing waits for that...)

or all messages:

```
opossum '-quiet=false' '-debug=true'
```

`$font` is used to select the font.

## macOS

Requirements:

- Go
- Plan9Port

```
go install ./cmd/opossum
```

# JS support

It's more like a demo and it's not really clear right now how much sandboxing
is really needed. A rudimentary AJAX implementation is there though.

Use on your own Risk!

Mostly based on goja (ECMAScript 5.1) and https://github.com/fgnass/domino
(fork of DOM implementation from Mozilla in JS). Some sort of DOM diffing
is needed, also AJAX functions, `getComputedStyle` etc. are either missing or stubs.
Very simple jQuery based code works though, e.g. jQuery UI Tab view
https://jqueryui.com/resources/demos/tabs/default.html or the toggle buttons on
https://golang.org/pkg There is also highly experimental ES6 support with Babel.
(Needs also https://github.com/psilva261/6to5)

Build the js engine:

```
go install ./cmd/gojafs
```

On 9legacy also the folder `/mnt/opossum` needs to exist.

```

Then try on Plan 9 with e.g.:

```
opossum '-experimentalJsInsecure=true' -startPage https://jqueryui.com/resources/demos/tabs/default.html
```

or macOS etc.:

```
opossum -experimentalJsInsecure=true -startPage https://jqueryui.com/resources/demos/tabs/default.html
```


# TODO

- load images on the fly
- implement more parts of HTML5 and CSS
- create a widget for div/span
- clean up code, support webfs, snarf
