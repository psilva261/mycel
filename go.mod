module github.com/psilva261/opossum

go 1.16

replace 9fans.net/go v0.0.0-00010101000000-000000000000 => github.com/knusbaum/go v0.0.0-20200413212707-848f58a0ec6e
replace github.com/srwiley/oksvg v0.0.0-20200311192757-870daf9aa564 => github.com/psilva261/oksvg v0.0.0-20210212153200-941e54e245a3

exclude github.com/aymerick/douceur v0.1.0

exclude github.com/aymerick/douceur v0.2.0

require (
	9fans.net/go v0.0.0-00010101000000-000000000000
	github.com/andybalholm/cascadia v1.1.0
	github.com/chris-ramon/douceur v0.2.1-0.20160603235419-f3463056cd52
	github.com/dlclark/regexp2 v1.4.0 // indirect
	github.com/dop251/goja v0.0.0-20210126164150-f5884268f0c0
	github.com/dop251/goja_nodejs v0.0.0-20200811150831-9bc458b4bbeb
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/gorilla/css v1.0.0 // indirect
	github.com/jvatic/goja-babel v0.0.0-20200102152603-63c66b7c796a
	github.com/mjl-/duit v0.0.0-20200330125617-580cb0b2843f
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646
	github.com/srwiley/oksvg v0.0.0-20200311192757-870daf9aa564
	github.com/srwiley/rasterx v0.0.0-20200120212402-85cb7272f5e9
	golang.org/x/image v0.0.0-20200927104501-e162460cd6b5
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b
	golang.org/x/text v0.3.5
	gopkg.in/yaml.v2 v2.3.0 // indirect
)
