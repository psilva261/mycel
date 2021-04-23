// +build plan9

package style

import (
	"fmt"
)

func initFontserver() {}

func (cs Map) FontFilename() string {
	fontSize := matchClosestFontSize(cs.FontSize(), []int{6,7,8,10,13})
	return fmt.Sprintf("/lib/font/bit/lucidasans/unicode.%v.font", fontSize)
}
