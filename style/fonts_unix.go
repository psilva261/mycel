// +build darwin freebsd netbsd openbsd linux

package style

import (
	"fmt"
	"math"
	"os/exec"
	"strings"
)

func initFontserver() {
	buf, err := exec.Command("fontsrv", "-p", ".").Output()
	if err == nil {
		availableFontNames = strings.Split(string(buf), "\n")
	} else {
		log.Printf("exec fontsrv: %v", err)
	}
}

func (cs Map) FontFilename() string {
	pref := cs.preferedFontName([]string{"HelveticaNeue", "Helvetica"})
	fontSize := 2 * /*dui.Scale(*/int(math.RoundToEven(cs.FontSize()))/*)*/

	return fmt.Sprintf("/mnt/font/"+pref+"%va/font", fontSize)
}