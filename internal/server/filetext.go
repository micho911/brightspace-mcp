package server

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// maxOfficeXML caps how much of one XML part of an Office file is read, so a
// zip bomb cannot fill memory.
const maxOfficeXML = 20 << 20

var reSlide = regexp.MustCompile(`^ppt/slides/slide(\d+)\.xml$`)

// textExtensions are file types read as plain text.
var textExtensions = map[string]bool{
	".txt": true, ".md": true, ".csv": true, ".tsv": true, ".json": true, ".xml": true,
	".yaml": true, ".yml": true, ".log": true, ".tex": true,
	".py": true, ".go": true, ".java": true, ".c": true, ".h": true, ".cpp": true,
	".js": true, ".ts": true, ".sql": true, ".r": true, ".sh": true,
}

// extractText returns the readable text of a file, or ok=false when the type
// is not supported.
func extractText(name, contentType string, data []byte) (text string, ok bool) {
	ext := strings.ToLower(path.Ext(name))
	switch {
	case ext == ".html" || ext == ".htm" || (ext == "" && contentType == "text/html"):
		return htmlToText(strings.ToValidUTF8(string(data), "")), true
	case ext == ".docx":
		return docxText(data)
	case ext == ".pptx":
		return pptxText(data)
	case textExtensions[ext] || (!isKnownBinaryExt(ext) && strings.HasPrefix(contentType, "text/")):
		return strings.ToValidUTF8(string(data), ""), true
	}
	return "", false
}

func isKnownBinaryExt(ext string) bool {
	switch ext {
	case ".pdf", ".docx", ".pptx", ".xlsx", ".doc", ".ppt", ".xls", ".zip", ".png", ".jpg", ".jpeg", ".gif", ".mp4", ".mov", ".mp3":
		return true
	}
	return false
}

func openZip(data []byte) (*zip.Reader, bool) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	return zr, err == nil
}

func zipPart(zr *zip.Reader, name string) (*zip.File, bool) {
	for _, f := range zr.File {
		if f.Name == name {
			return f, true
		}
	}
	return nil, false
}

func docxText(data []byte) (string, bool) {
	zr, ok := openZip(data)
	if !ok {
		return "", false
	}
	part, ok := zipPart(zr, "word/document.xml")
	if !ok {
		return "", false
	}
	text, ok := officeXMLText(part)
	return strings.TrimSpace(text), ok
}

func pptxText(data []byte) (string, bool) {
	zr, ok := openZip(data)
	if !ok {
		return "", false
	}
	type slide struct {
		n    int
		part *zip.File
	}
	var slides []slide
	for _, f := range zr.File {
		if m := reSlide.FindStringSubmatch(f.Name); m != nil {
			n, _ := strconv.Atoi(m[1])
			slides = append(slides, slide{n, f})
		}
	}
	if len(slides) == 0 {
		return "", false
	}
	sort.Slice(slides, func(i, j int) bool { return slides[i].n < slides[j].n })
	var b strings.Builder
	for _, s := range slides {
		text, ok := officeXMLText(s.part)
		if !ok {
			continue
		}
		b.WriteString("--- Slide " + strconv.Itoa(s.n) + " ---\n" + strings.TrimSpace(text) + "\n\n")
	}
	return strings.TrimSpace(b.String()), true
}

// officeXMLText reads the text runs (w:t in Word, a:t in PowerPoint) of an
// Office XML part, with a line break after each paragraph.
func officeXMLText(part *zip.File) (string, bool) {
	if part.UncompressedSize64 > maxOfficeXML {
		return "", false
	}
	rc, err := part.Open()
	if err != nil {
		return "", false
	}
	defer rc.Close()

	dec := xml.NewDecoder(io.LimitReader(rc, maxOfficeXML))
	var b strings.Builder
	inText := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return b.String(), b.Len() > 0
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "t":
				inText = true
			case "tab":
				b.WriteByte('\t')
			case "br", "cr":
				b.WriteByte('\n')
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				inText = false
			case "p":
				b.WriteByte('\n')
			}
		case xml.CharData:
			if inText {
				b.Write(t)
			}
		}
	}
	return b.String(), true
}
