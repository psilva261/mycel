package browser

import (
	"fmt"
	"github.com/psilva261/mycel"
	"github.com/psilva261/mycel/js"
	"github.com/psilva261/mycel/logger"
)

func processJS2(f  mycel.Fetcher) (s *js.JS, resHtm string, changed bool, err error) {
	s, resHtm, changed, err = js.Start(f)
	if err != nil {
		return nil, "", false, fmt.Errorf("start: %w", err)
	}
	log.Printf("processJS: changed = %v", changed)
	return
}
