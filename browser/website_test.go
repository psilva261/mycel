package browser

import (
	"golang.org/x/net/html"
	"strings"
	"testing"
)

func TestFormData(t *testing.T) {
	htm := `<form>
		<input name=a value=1>
		<textarea name=b>2</textarea>
	</form>`
	doc, err := html.Parse(
		strings.NewReader(string(htm)),
	)
	if err != nil {
		t.Fatalf(err.Error())
	}
	f := grep(doc, "form")
	data := formData(f, nil)
	if len(data) != 2 {
		t.Fatalf("%+v", f)
	}
}
