package epub

import (
	"archive/zip"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
)

// EPUB represents a parsed EPUB book
type EPUB struct {
	RootDir      string
	OpfPath      string
	Title        string
	Author       string
	ContentFiles []string
}

// Parse opens and parses an EPUB file
func Parse(filename string) (*EPUB, error) {
	r, err := zip.OpenReader(filename)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	epub := &EPUB{}

	// Find container.xml
	var containerFile *zip.File
	for _, f := range r.File {
		if f.Name == "META-INF/container.xml" {
			containerFile = f
			break
		}
	}
	if containerFile == nil {
		return nil, fmt.Errorf("META-INF/container.xml not found")
	}

	// Parse container.xml
	rc, _ := containerFile.Open()
	containerData, _ := io.ReadAll(rc)
	rc.Close()

	// Extract OPF path
	opfMatch := regexp.MustCompile(`full-path="([^"]+)"`).FindSubmatch(containerData)
	if len(opfMatch) < 2 {
		return nil, fmt.Errorf("could not find OPF path")
	}
	epub.OpfPath = string(opfMatch[1])
	epub.RootDir = filepath.Dir(epub.OpfPath)
	if epub.RootDir == "." {
		epub.RootDir = ""
	}

	// Find and parse OPF file
	var opfFile *zip.File
	for _, f := range r.File {
		if f.Name == epub.OpfPath {
			opfFile = f
			break
		}
	}
	if opfFile == nil {
		return nil, fmt.Errorf("OPF file not found: %s", epub.OpfPath)
	}

	rc, _ = opfFile.Open()
	opfData, _ := io.ReadAll(rc)
	rc.Close()

	// Extract metadata
	epub.Title = extractXMLField(opfData, "dc:title")
	epub.Author = extractXMLField(opfData, "dc:creator")
	if epub.Title == "" {
		epub.Title = strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	}

	// Find content files
	for _, f := range r.File {
		if !f.FileInfo().IsDir() {
			ext := strings.ToLower(filepath.Ext(f.Name))
			if ext == ".xhtml" || ext == ".html" || ext == ".htm" {
				if !strings.Contains(strings.ToLower(f.Name), "nav.") {
					epub.ContentFiles = append(epub.ContentFiles, f.Name)
				}
			}
		}
	}

	return epub, nil
}

// ExtractText extracts text content from EPUB files
func (e *EPUB) ExtractText(filename string) (string, error) {
	r, err := zip.OpenReader(filename)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var textBuilder strings.Builder

	for _, filePath := range e.ContentFiles {
		var file *zip.File
		for _, f := range r.File {
			if f.Name == filePath {
				file = f
				break
			}
		}
		if file == nil {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			continue
		}

		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		text := htmlToText(string(data))
		if strings.TrimSpace(text) != "" {
			textBuilder.WriteString(text)
			textBuilder.WriteString("\n\n")
		}
	}

	return textBuilder.String(), nil
}

func extractXMLField(data []byte, tag string) string {
	pattern := fmt.Sprintf(`<%s[^>]*>([^<]+)</%s>`, tag, tag)
	re := regexp.MustCompile(pattern)
	matches := re.FindSubmatch(data)
	if len(matches) > 1 {
		return strings.TrimSpace(string(matches[1]))
	}
	return ""
}

func htmlToText(html string) string {
	// Remove script and style tags
	re := regexp.MustCompile(`(?s)<script[^>]*>.*?</script>`)
	html = re.ReplaceAllString(html, "")
	re = regexp.MustCompile(`(?s)<style[^>]*>.*?</style>`)
	html = re.ReplaceAllString(html, "")

	// Replace block elements with newlines
	re = regexp.MustCompile(`(?i)</(?:p|div|h[1-6]|li|tr|td|th|br|hr)[^>]*>`)
	html = re.ReplaceAllString(html, "\n")

	// Remove remaining HTML tags
	re = regexp.MustCompile(`<[^>]*>`)
	html = re.ReplaceAllString(html, "")

	// Decode HTML entities
	html = strings.ReplaceAll(html, "&nbsp;", " ")
	html = strings.ReplaceAll(html, "&amp;", "&")
	html = strings.ReplaceAll(html, "&lt;", "<")
	html = strings.ReplaceAll(html, "&gt;", ">")
	html = strings.ReplaceAll(html, "&quot;", "\"")
	html = strings.ReplaceAll(html, "&apos;", "'")
	html = strings.ReplaceAll(html, "&#39;", "'")

	// Clean whitespace
	re = regexp.MustCompile(`[ \t]+`)
	html = re.ReplaceAllString(html, " ")
	re = regexp.MustCompile(`\n\s*\n\s*\n+`)
	html = re.ReplaceAllString(html, "\n\n")

	return strings.TrimSpace(html)
}

// PrepareText cleans and prepares text for TTS
func PrepareText(text string) string {
	// Remove excessive newlines
	re := regexp.MustCompile(`\n\s*\n\s*\n+`)
	text = re.ReplaceAllString(text, "\n\n")

	// Remove short lines (headers/footers)
	lines := strings.Split(text, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 10 {
			cleaned = append(cleaned, line)
		}
	}

	text = strings.Join(cleaned, " ")

	// Clean up spaces
	re = regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}

// SplitText splits text into chunks
func SplitText(text string, maxSize int) []string {
	var chunks []string
	sentences := regexp.MustCompile(`[.!?]+\s+`).Split(text, -1)

	var current strings.Builder
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}
		if !strings.HasSuffix(sentence, ".") &&
			!strings.HasSuffix(sentence, "!") &&
			!strings.HasSuffix(sentence, "?") {
			sentence += "."
		}

		if current.Len()+len(sentence)+1 > maxSize {
			if current.Len() > 0 {
				chunks = append(chunks, current.String())
				current.Reset()
			}
		}
		if current.Len() > 0 {
			current.WriteString(" ")
		}
		current.WriteString(sentence)
	}
	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}
	return chunks
}
