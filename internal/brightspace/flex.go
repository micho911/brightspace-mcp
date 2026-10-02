package brightspace

import (
	"bytes"
	"encoding/json"
)

// FlexText decodes a text field that Brightspace sends in different shapes:
// a plain string, a number, or a RichText block ({"Text":…,"Html":…}). It
// keeps the HTML form when there is one, else the text. Fields whose shape
// the documentation does not pin down use it, so one surprise cannot fail a
// whole listing.
type FlexText string

func (f *FlexText) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	switch {
	case len(b) == 0 || string(b) == "null":
		return nil
	case b[0] == '"':
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = FlexText(s)
	case b[0] == '{':
		var rt RichText
		if err := json.Unmarshal(b, &rt); err != nil {
			return err
		}
		*f = FlexText(rt.String())
	default:
		*f = FlexText(b) // a number or boolean: keep its text
	}
	return nil
}

// RichText is a Brightspace RichText block. Either form may be missing.
type RichText struct {
	Text string `json:"Text"`
	HTML string `json:"Html"`
}

// String returns the HTML form when there is one (it keeps links), else the
// plain text. Callers turn HTML into text themselves.
func (r RichText) String() string {
	if r.HTML != "" {
		return r.HTML
	}
	return r.Text
}
