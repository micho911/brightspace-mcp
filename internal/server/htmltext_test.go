package server

import "testing"

func TestHTMLToText(t *testing.T) {
	for name, tc := range map[string]struct{ in, want string }{
		"paragraphs": {
			in:   "<p>First</p>\n<p>Second&nbsp;line</p>",
			want: "First\n\nSecond line",
		},
		"link keeps its address": {
			in:   `<p>See <a href="https://example.org/q?a=1&amp;b=2" target="_blank" rel="noopener">here</a>.</p>`,
			want: "See here (https://example.org/q?a=1&b=2).",
		},
		"link whose text is the address": {
			in:   `<a href="https://example.org/x">https://example.org/x</a>`,
			want: "https://example.org/x",
		},
		"unsafe or relative links lose the address": {
			in:   `<a href="javascript:alert(1)">click</a> <a href="/d2l/x">local</a>`,
			want: "click local",
		},
		"list": {
			in:   "<ul><li>One</li><li>Two</li></ul>",
			want: "- One\n- Two",
		},
		"line break and entities": {
			in:   "a<br/>b &lt;c&gt; &amp; d",
			want: "a\nb <c> & d",
		},
		"scripts and styles are dropped": {
			in:   "<style>p{color:red}</style><p>Hi</p><script>alert('x')</script>",
			want: "Hi",
		},
		"image only link": {
			in:   `<a href="https://example.org/pic"><img src="a.png"></a>`,
			want: "https://example.org/pic",
		},
		"empty": {in: "", want: ""},
	} {
		t.Run(name, func(t *testing.T) {
			if got := htmlToText(tc.in); got != tc.want {
				t.Errorf("htmlToText(%q)\n got %q\nwant %q", tc.in, got, tc.want)
			}
		})
	}
}
