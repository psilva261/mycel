package browser

import (
	"fmt"
	"github.com/psilva261/opossum/js"
	"github.com/psilva261/opossum/logger"
)

func processJS2() (resHtm string, changed bool, err error) {
	resHtm, changed, err = js.Start()
	if err != nil {
		return "", false, fmt.Errorf("start: %w", err)
	}
	log.Printf("processJS: changed = %v", changed)
	return
}
