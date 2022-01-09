module github.com/psilva261/opossum

go 1.17

replace 9fans.net/go v0.0.0-00010101000000-000000000000 => github.com/psilva261/go v0.0.0-20210805155101-6b9925e0d807

replace 9fans.net/go v0.0.2 => github.com/psilva261/go v0.0.0-20210805155101-6b9925e0d807

replace github.com/mjl-/duit v0.0.0-20200330125617-580cb0b2843f => github.com/psilva261/duit v0.0.0-20210802155600-7e8fedefa7ba

exclude github.com/hanwen/go-fuse v1.0.0

exclude github.com/hanwen/go-fuse/v2 v2.0.3

require (
	9fans.net/go v0.0.2
	github.com/andybalholm/cascadia v1.3.1
	github.com/knusbaum/go9p v1.18.0
	github.com/mjl-/duit v0.0.0-20200330125617-580cb0b2843f
	github.com/srwiley/oksvg v0.0.0-20211120171407-1837d6608d8c
	github.com/srwiley/rasterx v0.0.0-20210519020934-456a8d69b780
	github.com/tdewolff/parse/v2 v2.5.26
	golang.org/x/image v0.0.0-20211028202545-6944b10bf410
	golang.org/x/net v0.0.0-20210916014120-12bc252f5db8
	golang.org/x/text v0.3.7
)

require (
	github.com/Plan9-Archive/libauth v0.0.0-20180917063427-d1ca9e94969d // indirect
	github.com/emersion/go-sasl v0.0.0-20200509203442-7bfe0ed36a21 // indirect
	github.com/fhs/mux9p v0.3.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
