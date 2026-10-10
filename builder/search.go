package builder

import (
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SearchIndex is the book-wide full-text index written next to the preview assets.
type SearchIndex struct {
	Pages []SearchPage `json:"pages"`
}

// SearchPage is one generated page. Sections keep heading anchors so a hit can scroll to the match.
type SearchPage struct {
	Title    string          `json:"title"`
	URL      string          `json:"url"`
	Sections []SearchSection `json:"sections"`
}

// SearchSection is a heading and the plain text that follows it.
type SearchSection struct {
	Heading string `json:"heading,omitempty"`
	Anchor  string `json:"anchor,omitempty"`
	Text    string `json:"text,omitempty"`
}

type searchDoc struct {
	Title  string
	URL    string
	HTML   string
	Source string
}

var (
	headingPattern = regexp.MustCompile(`(?is)<h([1-6])([^>]*)>(.*?)</h[1-6]>`)
	idAttrPattern  = regexp.MustCompile(`(?i)\bid\s*=\s*("([^"]*)"|'([^']*)')`)
	tagPattern     = regexp.MustCompile(`<[^>]+>`)
	spacePattern   = regexp.MustCompile(`\s+`)
)

func htmlPageURL(relPath string) string {
	ext := filepath.Ext(relPath)
	htmlPath := strings.TrimSuffix(relPath, ext) + ".html"
	return "/" + filepath.ToSlash(htmlPath)
}

func (b *Builder) readmeRelPath() string {
	rel := "README.md"
	if b.Book.Config != nil && b.Book.Config.Structure != nil && b.Book.Config.Structure.Readme != "" {
		rel = b.Book.Config.Structure.Readme
	}
	return filepath.ToSlash(rel)
}

func (b *Builder) addSearchDoc(title, url, html, source string) {
	source = filepath.ToSlash(source)
	for _, doc := range b.searchDocs {
		if doc.URL == url || (source != "" && doc.Source == source) {
			return
		}
	}
	b.searchDocs = append(b.searchDocs, searchDoc{
		Title:  title,
		URL:    url,
		HTML:   html,
		Source: source,
	})
}

func (b *Builder) hasSearchSource(source string) bool {
	source = filepath.ToSlash(source)
	for _, doc := range b.searchDocs {
		if doc.Source == source {
			return true
		}
	}
	return false
}

func (b *Builder) writeSearchIndex() error {
	index := SearchIndex{Pages: make([]SearchPage, 0, len(b.searchDocs))}
	for _, doc := range b.searchDocs {
		sections := sectionsFromHTML(doc.Title, doc.HTML)
		if len(sections) == 0 {
			sections = []SearchSection{{Heading: doc.Title}}
		}
		index.Pages = append(index.Pages, SearchPage{
			Title:    doc.Title,
			URL:      doc.URL,
			Sections: sections,
		})
	}

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	staticDir := filepath.Join(b.OutputDir, "static")
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(staticDir, "search-index.json"), data, 0644)
}

func sectionsFromHTML(pageTitle, fragment string) []SearchSection {
	matches := headingPattern.FindAllStringSubmatchIndex(fragment, -1)
	if len(matches) == 0 {
		text := plainText(fragment)
		if text == "" && pageTitle == "" {
			return nil
		}
		return []SearchSection{{Heading: pageTitle, Text: text}}
	}

	var sections []SearchSection
	if matches[0][0] > 0 {
		if text := plainText(fragment[:matches[0][0]]); text != "" {
			sections = append(sections, SearchSection{Heading: pageTitle, Text: text})
		}
	}

	for i, match := range matches {
		attrs := fragment[match[4]:match[5]]
		inner := fragment[match[6]:match[7]]
		end := len(fragment)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		heading := plainText(inner)
		body := plainText(fragment[match[1]:end])
		if heading == "" && body == "" {
			continue
		}
		sections = append(sections, SearchSection{
			Heading: heading,
			Anchor:  headingID(attrs),
			Text:    body,
		})
	}

	if len(sections) == 0 && (pageTitle != "" || plainText(fragment) != "") {
		return []SearchSection{{Heading: pageTitle, Text: plainText(fragment)}}
	}
	return sections
}

func firstHeading(fragment string) string {
	for _, section := range sectionsFromHTML("", fragment) {
		if section.Heading != "" {
			return section.Heading
		}
	}
	return ""
}

func headingID(attrs string) string {
	match := idAttrPattern.FindStringSubmatch(attrs)
	if match == nil {
		return ""
	}
	if match[2] != "" {
		return match[2]
	}
	return match[3]
}

func plainText(fragment string) string {
	text := tagPattern.ReplaceAllString(fragment, " ")
	text = html.UnescapeString(text)
	text = spacePattern.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}
