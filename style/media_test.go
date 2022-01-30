package style

import (
	"testing"
)

func TestParseQuery(t *testing.T) {
	mqs, err := parseQuery(`only screen and (max-width: 600px)`)
	if err != nil {
		t.Fail()
	}
	if len(mqs) != 1 {
		t.Fail()
	}
	mq := mqs[0]
	if mq.inverse || mq.typ != "screen" || len(mq.exprs) != 1 {
		t.Fail()
	}
	expr := mq.exprs[0]
	if expr.modifier != "max" || expr.feature != "width" || expr.value != "600px" {
		t.Fail()
	}
}

func TestMatchQuery(t *testing.T) {
	matching := map[string]string{
		"type": "screen",
		"width": "500",
	}
	notMatching := map[string]string{
		"type": "screen",
		"width": "700",
	}
	yes, err := MatchQuery(`only screen and (max-width: 600px)`, matching)
	if err != nil {
		t.Fail()
	}
	t.Logf("%v", yes)
	if !yes {
		t.Fail()
	}
	yes, err = MatchQuery(`only screen and (max-width: 600px)`, notMatching)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("%v", yes)
	if yes {
		t.Fail()
	}
}
