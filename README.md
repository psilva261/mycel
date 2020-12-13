# Opossum Web Browser

Basic portable Web browser; only needs a Go compiler to compile, no C dependencies

Supported features:

- rudimentary CSS/HTML5 support, large parts like float/flex layout are just stub implementations
- Server-side rendered websites
- Images (pre-loaded all at once though)
- TLS
- experimental JS/DOM without AJAX can be activated (basically script tags are evaluated)

# Install

## Plan 9

You can download a tarball with the binary at http://psilva.sdf.org/opossum-plan9-amd64.tgz

To compile the source Go 1.15 is needed. Probably `$GOPROXY` should be set to `https://proxy.golang.org`

```
cd cmd/browse
go run .
```

## macOS

Requirements:

- Go
- Plan9Port

```
cd cmd/browse
go run .
```

## TODO

- load images on the fly
- implement more parts of HTML5 and CSS
- create a widget for div/span
- clean up code, support webfs, snarf, file downloads
