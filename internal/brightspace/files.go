package brightspace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
)

// ErrTooLarge means a file is bigger than the size the caller allows.
var ErrTooLarge = errors.New("file is larger than the size limit")

// File is a downloaded file. It lives in memory only.
type File struct {
	Name        string
	ContentType string
	Data        []byte
}

// TopicFile downloads the file of a content item. It never reads more than
// maxBytes: a bigger file gives ErrTooLarge.
func (c *Client) TopicFile(ctx context.Context, orgUnitID, topicID, maxBytes int64) (File, error) {
	path := "/d2l/api/le/" + leVersion + "/" + strconv.FormatInt(orgUnitID, 10) + "/content/topics/" + strconv.FormatInt(topicID, 10) + "/file"
	resp, err := c.send(ctx, http.MethodGet, path, http.Header{"Accept": {"*/*"}}, nil)
	if err != nil {
		return File{}, err
	}
	defer resp.Body.Close()

	mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return File{}, fmt.Errorf("GET %s: %w", path, ErrNotFound)
	case resp.StatusCode == http.StatusUnauthorized || (resp.StatusCode >= 300 && resp.StatusCode < 400):
		return File{}, ErrSessionExpired
	case resp.StatusCode == http.StatusForbidden && mediaType == "text/html":
		return File{}, ErrSessionExpired
	case resp.StatusCode == http.StatusForbidden:
		return File{}, fmt.Errorf("GET %s: %w", path, ErrForbidden)
	case resp.StatusCode != http.StatusOK:
		return File{}, fmt.Errorf("GET %s: unexpected status %s", path, resp.Status)
	case resp.ContentLength > maxBytes:
		return File{}, ErrTooLarge
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return File{}, fmt.Errorf("GET %s: read file: %w", path, err)
	}
	if int64(len(data)) > maxBytes {
		return File{}, ErrTooLarge
	}
	// A lost session can also come back as a 200 login page.
	if mediaType == "text/html" && looksLikeLogin(data) {
		return File{}, ErrSessionExpired
	}

	f := File{ContentType: mediaType, Data: data}
	if _, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition")); err == nil {
		f.Name = params["filename"]
	}
	return f, nil
}

// looksLikeLogin reports whether an HTML page is Brightspace's login page or
// its session-expired stub.
func looksLikeLogin(html []byte) bool {
	head := bytes.ToLower(html[:min(len(html), 4096)])
	return bytes.Contains(head, []byte("/d2l/login")) || bytes.Contains(head, []byte("sessionexpired"))
}
