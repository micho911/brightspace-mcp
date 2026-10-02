package brightspace

import (
	"encoding/json"
	"testing"
)

func TestFlexText(t *testing.T) {
	cases := map[string]string{
		`"x"`:                            "x",
		`{"Text":"a","Html":"<b>a</b>"}`: "<b>a</b>",
		`{"Text":"a","Html":null}`:       "a",
		`{"Text":"","Html":""}`:          "",
		`42`:                             "42",
		`null`:                           "",
	}
	for in, want := range cases {
		var got FlexText
		if err := json.Unmarshal([]byte(in), &got); err != nil {
			t.Errorf("%s: %v", in, err)
		} else if string(got) != want {
			t.Errorf("%s = %q, want %q", in, got, want)
		}
	}
}
