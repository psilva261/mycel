package img

import (
	"opossum/logger"
	"testing"
)

func init() {
	SetLogger(&logger.Logger{})
}

func TestParseDataUri(t *testing.T) {
	srcs := []string{"data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP//yH5BAEAAAAALAAAAAABAAEAAAIBRAA7",
		"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNgYAAAAAMAASsJTYQAAAAASUVORK5CYII=",
	}

	for _, src := range srcs {
		data, err := parseDataUri(src)
		if err != nil {
			t.Fatalf(err.Error())
		}
		t.Logf("%v", data)
	}
}

