// +build darwin freebsd netbsd openbsd linux

package style

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var availableFontSizes = make(map[string][]int)

func initFontserver() {
	buf, err := exec.Command("fontsrv", "-p", ".").Output()
	if err == nil {
		availableFontNames = strings.Split(string(buf), "\n")
	} else {
		log.Printf("exec fontsrv: %v", err)
	}
}

func fontSizes(fontName string) (fss []int, err error) {
	re := regexp.MustCompile(`^(\d+)$`)
	fss = make([]int, 0, 20)

	buf, err := exec.Command("fontsrv", "-p", fontName).Output()
	if err != nil {
		return
	}
	for _, s := range strings.Split(string(buf), "\n") {
		s = strings.TrimSpace(s)
		s = strings.TrimSuffix(s, "/")
		if !re.MatchString(s) {
			continue
		}
		fs, err := strconv.Atoi(s)
		if err != nil {
			log.Errorf("%v: %v", fs, err)
		}
		fss = append(fss, fs)
	}

	return
}

func (cs Map) FontFilename() (string, bool) {
	f := cs.preferedFontName([]string{"HelveticaNeue", "Helvetica"})
	if _, ok := availableFontSizes[f]; !ok {
		fss, err := fontSizes(f)
		if err != nil {
			log.Errorf("font sizes %v: %v", f, err)
		}
		availableFontSizes[f] = fss
	}
	s := matchClosestFontSize(2*cs.FontSize(), availableFontSizes[f])

	return fmt.Sprintf("/mnt/font/"+f+"%va/font", s), true
}
