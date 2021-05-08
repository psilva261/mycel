package jsfcall

const (
	_ = iota
	Rerror
	Tinit
	Rinit
	Tclick
	Rclick
	Tquery
	Rquery
)

type Msg struct {
	Type    uint8
	Error   string
	Changed bool
	Html  string
}
