module github.com/psilva261/mycel

go 1.18

replace 9fans.net/go v0.0.0-00010101000000-000000000000 => github.com/psilva261/go v0.0.0-20210805155101-6b9925e0d807

replace 9fans.net/go v0.0.2 => github.com/psilva261/go v0.0.0-20210805155101-6b9925e0d807

replace github.com/mjl-/duit v0.0.0-20200330125617-580cb0b2843f => github.com/psilva261/duit v0.0.0-20210802155600-7e8fedefa7ba

exclude github.com/hanwen/go-fuse v1.0.0

exclude github.com/hanwen/go-fuse/v2 v2.0.3

exclude github.com/hanwen/go-fuse/v2 v2.1.0

require (
	9fans.net/go v0.0.2
	github.com/andybalholm/cascadia v1.3.1
	github.com/knusbaum/go9p v1.18.0
	github.com/mjl-/duit v0.0.0-20200330125617-580cb0b2843f
	github.com/srwiley/oksvg v0.0.0-20220731023508-a61f04f16b76
	github.com/srwiley/rasterx v0.0.0-20220730225603-2ab79fcdd4ef
	github.com/tdewolff/parse/v2 v2.5.26
	golang.org/x/image v0.0.0-20220902085622-e7cb96979f69
	golang.org/x/net v0.0.0-20220826154423-83b083e8dc8b
	golang.org/x/text v0.3.7
)

require (
	github.com/Plan9-Archive/libauth v0.0.0-20180917063427-d1ca9e94969d // indirect
	github.com/emersion/go-sasl v0.0.0-20200509203442-7bfe0ed36a21 // indirect
	github.com/fhs/mux9p v0.3.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
