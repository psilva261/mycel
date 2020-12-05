// +build plan9

package style

import (
	"fmt"
	"math"
)

func matchClosestFontSize(desired float64, available []int) (closest int) {
	for _, a := range available {
		if closest == 0 || math.Abs(float64(a)-desired) < math.Abs(float64(closest)-desired) {
			closest = a
		}
	}
	return
}

func (cs Map) FontFilename() string {
	fontSize := matchClosestFontSize(cs.FontSize(), []int{5,6,7,8,9,10,12,14,16,18,20,24,28,32})
	return fmt.Sprintf("/lib/font/bit/lucida/unicode.%v.font", fontSize)
}
