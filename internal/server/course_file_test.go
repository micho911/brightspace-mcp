package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func zipOf(t *testing.T, parts map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range parts {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type served struct {
	name, contentType string
	status            int
	body              []byte
}

func fakeFiles(t *testing.T, files map[string]served) Connect {
	return fakeBrightspace(t, func(w http.ResponseWriter, r *http.Request) {
		f, ok := files[r.URL.Path[strings.Index(r.URL.Path, "/topics/"):]]
		if !ok {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if f.contentType != "" {
			w.Header().Set("Content-Type", f.contentType)
		}
		if f.name != "" {
			w.Header().Set("Content-Disposition", `attachment; filename="`+f.name+`"`)
		}
		if f.status != 0 {
			w.WriteHeader(f.status)
		}
		_, _ = w.Write(f.body)
	})
}

func callReadCourseFile(t *testing.T, bs Connect, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := connect(t, bs).CallTool(context.Background(), &mcp.CallToolParams{Name: "read_course_file", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func fileText(t *testing.T, res *mcp.CallToolResult) CourseFileText {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res.Content)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	var f CourseFileText
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return f
}

func read(t *testing.T, bs Connect, id int, extra map[string]any) CourseFileText {
	args := map[string]any{"courseId": 1001, "topicId": id}
	for k, v := range extra {
		args[k] = v
	}
	return fileText(t, callReadCourseFile(t, bs, args))
}

func TestReadCourseFile(t *testing.T) {
	docx := zipOf(t, map[string]string{"word/document.xml": `<w:document xmlns:w="x"><w:body>
		<w:p><w:r><w:t>Hello</w:t></w:r><w:r><w:tab/><w:t>world</w:t></w:r></w:p><w:p><w:r><w:t>Second</w:t></w:r></w:p></w:body></w:document>`})
	pptx := zipOf(t, map[string]string{
		"ppt/slides/slide2.xml":  `<p:sld xmlns:a="x" xmlns:p="y"><a:p><a:r><a:t>Two</a:t></a:r></a:p></p:sld>`,
		"ppt/slides/slide10.xml": `<p:sld xmlns:a="x" xmlns:p="y"><a:p><a:r><a:t>Ten</a:t></a:r></a:p></p:sld>`,
		"ppt/slides/slide1.xml":  `<p:sld xmlns:a="x" xmlns:p="y"><a:p><a:r><a:t>One</a:t></a:r></a:p></p:sld>`})
	bs := fakeFiles(t, map[string]served{
		"/topics/1/file": {name: "notes.txt", contentType: "text/plain; charset=utf-8", body: []byte("plain notes")},
		"/topics/2/file": {name: "a.docx", contentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", body: docx},
		"/topics/3/file": {name: "s.pptx", contentType: "application/octet-stream", body: pptx},
		"/topics/4/file": {name: "page.html", contentType: "text/html", body: []byte("<h1>Title</h1><p>Body <a href=\"https://x.example\">link</a></p><script>x()</script>")},
		"/topics/5/file": {name: "slides.pdf", contentType: "application/pdf", body: []byte("%PDF-1.7")},
		"/topics/6/file": {name: "x.txt", contentType: "text/plain", body: []byte("0123456789")},
	})

	if f := read(t, bs, 1, nil); !f.Readable || f.Text != "plain notes" || f.Name != "notes.txt" || f.SizeBytes != 11 {
		t.Errorf("text = %+v", f)
	}
	if f := read(t, bs, 2, nil); f.Text != "Hello\tworld\nSecond" {
		t.Errorf("docx text = %q", f.Text)
	}
	if f := read(t, bs, 3, nil); f.Text != "--- Slide 1 ---\nOne\n\n--- Slide 2 ---\nTwo\n\n--- Slide 10 ---\nTen" {
		t.Errorf("pptx text = %q (slides must be in numeric order)", f.Text)
	}
	if f := read(t, bs, 4, nil); f.Text != "Title\nBody link (https://x.example)" {
		t.Errorf("html text = %q", f.Text)
	}
	if f := read(t, bs, 5, nil); f.Readable || f.Note == "" || f.Text != "" {
		t.Errorf("pdf = %+v, want not readable with a note", f)
	}
	if f := read(t, bs, 6, map[string]any{"maxChars": 4}); f.Text != "0123…" || !f.Truncated {
		t.Errorf("truncated = %+v", f)
	}
}

func TestReadCourseFileTooLarge(t *testing.T) {
	old := maxFileBytes
	maxFileBytes = 8
	t.Cleanup(func() { maxFileBytes = old })
	bs := fakeFiles(t, map[string]served{"/topics/1/file": {name: "big.txt", contentType: "text/plain", body: []byte("123456789")}})

	if f := read(t, bs, 1, nil); f.Readable || !strings.Contains(f.Note, "larger than") || f.Text != "" {
		t.Errorf("big file = %+v", f)
	}
}

func TestReadCourseFileSessionAndErrors(t *testing.T) {
	bs := fakeFiles(t, map[string]served{
		"/topics/1/file": {contentType: "text/html", body: []byte(`<script>location.replace("/d2l/login?sessionExpired=1")</script>`)},
		"/topics/2/file": {contentType: "text/plain", status: http.StatusForbidden, body: []byte("no")},
	})
	if msg := errorText(t, callReadCourseFile(t, bs, map[string]any{"courseId": 1001, "topicId": 1})); !strings.Contains(msg, "session has expired") {
		t.Errorf("login stub: error %q", msg)
	}
	if msg := errorText(t, callReadCourseFile(t, bs, map[string]any{"courseId": 1001, "topicId": 2})); !strings.Contains(msg, "does not let the user see files") {
		t.Errorf("forbidden: error %q", msg)
	}
	if msg := errorText(t, callReadCourseFile(t, bs, map[string]any{"courseId": 1001, "topicId": 9})); !strings.Contains(msg, "file 9 not found") {
		t.Errorf("unknown: error %q", msg)
	}
	if msg := errorText(t, callReadCourseFile(t, bs, map[string]any{"courseId": 1001, "topicId": 0})); !strings.Contains(msg, "topicId is required") {
		t.Errorf("missing topicId: error %q", msg)
	}
}
