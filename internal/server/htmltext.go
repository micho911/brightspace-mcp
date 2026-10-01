package server

import (
	"html"
	"regexp"
	"strings"
)

var (
	reHTMLHidden = regexp.MustCompile(`(?is)<(script|style)\b.*?</(script|style)>`)
	reHTMLAnchor = regexp.MustCompile(`(?is)<a\s[^>]*?href\s*=\s*["']([^"']*)["'][^>]*>(.*?)</a>`)
	reHTMLBreak  = regexp.MustCompile(`(?i)<br\s*/?>|<(ul|ol)\b[^>]*>|</(p|div|li|h[1-6]|tr|ul|ol|blockquote)>`)
	reHTMLItem   = regexp.MustCompile(`(?i)<li\b[^>]*>`)
	reHTMLTag    = regexp.MustCompile(`(?s)<[^>]*>`)
	reSpaces     = regexp.MustCompile(`[ \t\f\v]+`)
	reBlankLines = regexp.MustCompile(`\n{3,}`)
)

// htmlToText turns the HTML of a teacher's post into plain text for an
// assistant: tags are dropped, paragraphs and list items become lines, and a
// link keeps its address, as "text (https://...)", because the address is
// often the point of the post.
func htmlToText(s string) string {
	s = reHTMLHidden.ReplaceAllString(s, "")
	s = reHTMLAnchor.ReplaceAllStringFunc(s, func(a string) string {
		m := reHTMLAnchor.FindStringSubmatch(a)
		href := html.UnescapeString(strings.TrimSpace(m[1]))
		label := strings.TrimSpace(reHTMLTag.ReplaceAllString(m[2], ""))
		lower := strings.ToLower(href)
		if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
			return label
		}
		if label == "" || label == href {
			return href
		}
		return label + " (" + href + ")"
	})
	s = reHTMLItem.ReplaceAllString(s, "- ")
	s = reHTMLBreak.ReplaceAllString(s, "\n")
	s = reHTMLTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, " ", " ")
	s = reSpaces.ReplaceAllString(s, " ")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}
	s = reBlankLines.ReplaceAllString(strings.Join(lines, "\n"), "\n\n")
	return strings.TrimSpace(s)
}
