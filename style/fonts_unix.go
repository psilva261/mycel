// +build darwin freebsd netbsd openbsd linux

package style

import (
	"fmt"
	"math"
)

func (cs Map) FontFilename() string {
	pref := cs.preferedFontName([]string{"HelveticaNeue", "Helvetica"})
	fontSize := 2 * /*dui.Scale(*/int(math.RoundToEven(cs.FontSize()))/*)*/

	return fmt.Sprintf("/mnt/font/"+pref+"%va/font", fontSize)
}