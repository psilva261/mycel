module github.com/psilva261/opossum

go 1.16

replace 9fans.net/go v0.0.0-00010101000000-000000000000 => github.com/psilva261/go v0.0.0-20210409105224-1f229ca22d72

replace 9fans.net/go v0.0.2 => github.com/psilva261/go v0.0.0-20210409105224-1f229ca22d72

replace github.com/knusbaum/go9p v1.17.0 => github.com/psilva261/go9p-1 v1.17.1-0.20210620075710-0428cf31f72f

exclude github.com/aymerick/douceur v0.1.0

exclude github.com/aymerick/douceur v0.2.0

exclude github.com/hanwen/go-fuse v1.0.0

exclude github.com/hanwen/go-fuse/v2 v2.0.3

require (
	9fans.net/go v0.0.2
	github.com/andybalholm/cascadia v1.1.0
	github.com/chris-ramon/douceur v0.2.1-0.20160603235419-f3463056cd52
	github.com/dlclark/regexp2 v1.4.0 // indirect
	github.com/dop251/goja v0.0.0-20210317175251-bb14c2267b76
	github.com/dop251/goja_nodejs v0.0.0-20210225215109-d91c329300e7
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/gorilla/css v1.0.0 // indirect
	github.com/knusbaum/go9p v1.17.0
	github.com/mjl-/duit v0.0.0-20200330125617-580cb0b2843f
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646
	github.com/srwiley/oksvg v0.0.0-20210320200257-875f767ac39a
	github.com/srwiley/rasterx v0.0.0-20200120212402-85cb7272f5e9
	golang.org/x/image v0.0.0-20210220032944-ac19c3e999fb
	golang.org/x/net v0.0.0-20210316092652-d523dce5a7f4
	golang.org/x/text v0.3.5
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
