package builder

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hitzhangjie/gitbook/book"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// Builder builds a GitBook project
type Builder struct {
	Book      *book.Book
	OutputDir string
	template  *template.Template
	md        goldmark.Markdown
}

// NavItem represents a navigation item
type NavItem struct {
	Title    string
	Path     string
	URL      string
	Active   bool
	Level    int
	Children []NavItem
}

// TOCItem represents a table of contents item
type TOCItem struct {
	Title    string
	ID       string
	Level    int
	Children []TOCItem
}

// PageData represents data for page template
type PageData struct {
	Title       string
	BookTitle   string
	Content     template.HTML
	NavTree     []NavItem
	TOC         []TOCItem
	CurrentPath string
}

//go:embed templates/page.html
var pageTemplate embed.FS

//go:embed static/*
var staticFiles embed.FS

// NewBuilder creates a new builder
func NewBuilder(bookRoot, outputDir string) (*Builder, error) {
	b, err := book.LoadBook(bookRoot)
	if err != nil {
		return nil, err
	}

	if outputDir == "" {
		outputDir = filepath.Join(bookRoot, "_book")
	}

	absOutput, err := filepath.Abs(outputDir)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.ParseFS(pageTemplate, "templates/page.html")
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	// Initialize goldmark
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(), // Allow raw HTML tags like <img>
		),
	)

	return &Builder{
		Book:      b,
		OutputDir: absOutput,
		template:  tmpl,
		md:        md,
	}, nil
}

// Build builds the book
func (b *Builder) Build() error {
	// Clean output directory
	if err := os.RemoveAll(b.OutputDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean output directory: %w", err)
	}

	if err := os.MkdirAll(b.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Copy static assets
	if err := b.copyAssets(); err != nil {
		return fmt.Errorf("failed to copy assets: %w", err)
	}

	// Copy static files (CSS, JS)
	if err := b.copyStaticFiles(); err != nil {
		return fmt.Errorf("failed to copy static files: %w", err)
	}

	// Generate HTML pages
	if err := b.generatePages(); err != nil {
		return fmt.Errorf("failed to generate pages: %w", err)
	}

	// Generate index.html
	if err := b.generateIndex(); err != nil {
		return fmt.Errorf("failed to generate index: %w", err)
	}

	return nil
}

func (b *Builder) copyAssets() error {
	// Walk the book root directory and copy all non-markdown files
	// while preserving the directory structure
	return filepath.Walk(b.Book.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories that should not be copied
		if info.IsDir() {
			// Skip output directory and common hidden/system directories
			base := filepath.Base(path)
			if base == "_book" || base == ".git" || base == ".gitbook" || strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip markdown files
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".md" || ext == ".markdown" {
			return nil
		}

		// Calculate relative path from book root
		relPath, err := filepath.Rel(b.Book.Root, path)
		if err != nil {
			return err
		}

		// Build destination path
		dstPath := filepath.Join(b.OutputDir, relPath)

		// Create destination directory if needed
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		// Copy file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if err := os.WriteFile(dstPath, data, info.Mode()); err != nil {
			return err
		}

		return nil
	})
}

func (b *Builder) copyStaticFiles() error {
	// Copy static files from embedded FS to _book/static
	staticDst := filepath.Join(b.OutputDir, "static")
	if err := os.MkdirAll(staticDst, 0755); err != nil {
		return err
	}

	// Walk embedded static files
	return b.copyEmbeddedFiles(staticFiles, "static", staticDst)
}

func (b *Builder) copyEmbeddedFiles(fs embed.FS, srcDir, dstDir string) error {
	entries, err := fs.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())

		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				return err
			}
			if err := b.copyEmbeddedFiles(fs, srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := fs.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *Builder) generatePages() error {
	if b.Book.Summary == nil {
		return nil
	}

	// Build navigation tree
	navTree := b.buildNavTree(b.Book.Summary.Chapters, "")

	// Generate all pages
	return b.generateChapters(b.Book.Summary.Chapters, "", navTree, "")
}

func (b *Builder) generateChapters(chapters []book.Chapter, basePath string, navTree []NavItem, currentPath string) error {
	for _, chapter := range chapters {
		if chapter.Path == "" {
			continue
		}

		// Read markdown file
		mdPath := filepath.Join(b.Book.Root, chapter.Path)
		content, err := os.ReadFile(mdPath)
		if err != nil {
			// Skip if file doesn't exist
			continue
		}

		// Convert markdown to HTML
		html, err := b.markdownToHTML(string(content))
		if err != nil {
			return fmt.Errorf("failed to convert markdown: %w", err)
		}

		// Extract TOC from HTML to ensure IDs match exactly with goldmark's generated IDs
		toc := b.extractTOCFromHTML(html)

		// Generate HTML page path
		htmlPath := strings.TrimSuffix(chapter.Path, ".md") + ".html"
		if basePath != "" {
			htmlPath = filepath.Join(basePath, htmlPath)
		}
		outputPath := filepath.Join(b.OutputDir, htmlPath)
		relPath := htmlPath

		// Create directory if needed
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return err
		}

		// Mark active item in nav tree
		activeNavTree := b.markActiveNavItem(navTree, relPath)

		// Generate full HTML page
		bookTitle := "GitBook"
		if b.Book.Config != nil && b.Book.Config.Title != "" {
			bookTitle = b.Book.Config.Title
		}

		pageData := PageData{
			Title:       chapter.Title,
			BookTitle:   bookTitle,
			Content:     template.HTML(html),
			NavTree:     activeNavTree,
			TOC:         toc,
			CurrentPath: relPath,
		}

		fullHTML, err := b.renderTemplate(pageData)
		if err != nil {
			return fmt.Errorf("failed to render template: %w", err)
		}

		if err := os.WriteFile(outputPath, []byte(fullHTML), 0644); err != nil {
			return err
		}

		// Recursively generate sub-chapters
		if len(chapter.Articles) > 0 {
			if err := b.generateChapters(chapter.Articles, filepath.Dir(htmlPath), navTree, relPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func (b *Builder) generateIndex() error {
	// Generate main index.html
	readmePath := filepath.Join(b.Book.Root, "README.md")
	if b.Book.Config != nil && b.Book.Config.Structure != nil && b.Book.Config.Structure.Readme != "" {
		readmePath = filepath.Join(b.Book.Root, b.Book.Config.Structure.Readme)
	}

	var content template.HTML
	var toc []TOCItem
	if data, err := os.ReadFile(readmePath); err == nil {
		html, err := b.markdownToHTML(string(data))
		if err == nil {
			content = template.HTML(html)
			toc = b.extractTOCFromHTML(html)
		}
	}

	title := "Introduction"
	bookTitle := "GitBook"
	if b.Book.Config != nil && b.Book.Config.Title != "" {
		bookTitle = b.Book.Config.Title
	}

	// Build navigation tree
	var navTree []NavItem
	if b.Book.Summary != nil {
		navTree = b.buildNavTree(b.Book.Summary.Chapters, "")
		navTree = b.markActiveNavItem(navTree, "index.html")
	}

	pageData := PageData{
		Title:       title,
		BookTitle:   bookTitle,
		Content:     content,
		NavTree:     navTree,
		TOC:         toc,
		CurrentPath: "index.html",
	}

	fullHTML, err := b.renderTemplate(pageData)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	return os.WriteFile(filepath.Join(b.OutputDir, "index.html"), []byte(fullHTML), 0644)
}

func (b *Builder) markdownToHTML(md string) (string, error) {
	var buf bytes.Buffer
	if err := b.md.Convert([]byte(md), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (b *Builder) renderTemplate(data PageData) (string, error) {
	var buf bytes.Buffer
	if err := b.template.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (b *Builder) buildNavTree(chapters []book.Chapter, basePath string) []NavItem {
	return b.buildNavTreeWithLevel(chapters, basePath, 1)
}

func (b *Builder) buildNavTreeWithLevel(chapters []book.Chapter, basePath string, level int) []NavItem {
	var items []NavItem
	for _, chapter := range chapters {
		item := NavItem{
			Title: chapter.Title,
			Path:  chapter.Path,
			Level: level,
		}

		if chapter.Path != "" {
			htmlPath := strings.TrimSuffix(chapter.Path, ".md") + ".html"
			if basePath != "" {
				htmlPath = filepath.Join(basePath, htmlPath)
			}
			// Convert to relative URL
			item.URL = "/" + strings.ReplaceAll(htmlPath, "\\", "/")
		}

		if len(chapter.Articles) > 0 {
			nextBasePath := basePath
			if chapter.Path != "" {
				nextBasePath = filepath.Dir(item.URL)
				if nextBasePath == "." {
					nextBasePath = ""
				}
			}
			item.Children = b.buildNavTreeWithLevel(chapter.Articles, nextBasePath, level+1)
		}

		items = append(items, item)
	}
	return items
}

func (b *Builder) markActiveNavItem(navTree []NavItem, currentPath string) []NavItem {
	result := make([]NavItem, len(navTree))
	for i, item := range navTree {
		result[i] = item
		if item.URL == "/"+strings.ReplaceAll(currentPath, "\\", "/") {
			result[i].Active = true
		}
		if len(item.Children) > 0 {
			result[i].Children = b.markActiveNavItem(item.Children, currentPath)
			// If any child is active, mark parent as active too
			for _, child := range result[i].Children {
				if child.Active {
					result[i].Active = true
					break
				}
			}
		}
	}
	return result
}

// extractTOCFromHTML extracts TOC from HTML, ensuring IDs match exactly with goldmark's generated IDs
func (b *Builder) extractTOCFromHTML(html string) []TOCItem {
	var toc []TOCItem

	// Regex to match HTML headings: <h1 id="xxx">, <h2 id="yyy">, etc.
	// This ensures we get the exact IDs that goldmark generated
	// Use (?s) flag to make . match newlines, and handle id attribute in any position
	headingRegex := regexp.MustCompile(`(?s)<h([1-6])[^>]*id="([^"]+)"[^>]*>([\s\S]*?)</h[1-6]>`)
	var stack []*TOCItem

	matches := headingRegex.FindAllStringSubmatch(html, -1)
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		levelStr := match[1]
		id := match[2]
		titleHTML := match[3]

		// Parse level
		var level int
		fmt.Sscanf(levelStr, "%d", &level)

		// Extract plain text from HTML title (remove HTML tags)
		title := b.extractTextFromHTML(titleHTML)

		// Skip empty titles
		if title == "" {
			continue
		}

		item := TOCItem{
			Title: title,
			ID:    id,
			Level: level,
		}

		// Find parent in stack
		for len(stack) > 0 && stack[len(stack)-1].Level >= level {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			toc = append(toc, item)
			stack = append(stack, &toc[len(toc)-1])
		} else {
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, item)
			stack = append(stack, &parent.Children[len(parent.Children)-1])
		}
	}

	return toc
}

// extractTextFromHTML extracts plain text from HTML, removing all HTML tags
func (b *Builder) extractTextFromHTML(html string) string {
	// Remove HTML tags
	text := regexp.MustCompile(`<[^>]+>`).ReplaceAllString(html, "")
	// Decode HTML entities (basic ones)
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	// Clean up whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)
	return text
}

// removeMarkdownFormatting removes markdown formatting from text
// This matches goldmark's behavior when generating heading IDs
func (b *Builder) removeMarkdownFormatting(text string) string {
	// Process in order to avoid conflicts - iterate until no more changes

	for {
		oldText := text

		// Remove code blocks: `code` or ``code``
		text = regexp.MustCompile("`+[^`]+`+").ReplaceAllString(text, "")

		// Remove images: ![alt](url)
		text = regexp.MustCompile(`!\[([^\]]*)\]\([^\)]+\)`).ReplaceAllString(text, "$1")

		// Remove links: [text](url) or [text][ref]
		text = regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`).ReplaceAllString(text, "$1")
		text = regexp.MustCompile(`\[([^\]]+)\]\[[^\]]+\]`).ReplaceAllString(text, "$1")

		// Remove strikethrough: ~~text~~
		text = regexp.MustCompile(`~~([^~]+)~~`).ReplaceAllString(text, "$1")

		// Remove bold: **text** or __text__
		text = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(text, "$1")
		text = regexp.MustCompile(`__([^_]+)__`).ReplaceAllString(text, "$1")

		// Remove italic: *text* or _text_ (single, not part of ** or __)
		// Match *text* where * is not followed or preceded by another *
		text = regexp.MustCompile(`([^*]|^)\*([^*\s][^*]*[^*\s])\*([^*]|$)`).ReplaceAllString(text, "${1}${2}${3}")
		text = regexp.MustCompile(`([^_]|^)_([^_\s][^_]*[^_\s])_([^_]|$)`).ReplaceAllString(text, "${1}${2}${3}")

		// Remove any remaining markdown link references: [text]
		text = regexp.MustCompile(`\[([^\]]+)\]`).ReplaceAllString(text, "$1")

		// Remove HTML tags if any
		text = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(text, "")

		// If no changes, break
		if text == oldText {
			break
		}
	}

	// Clean up multiple spaces
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	// Trim whitespace
	text = strings.TrimSpace(text)

	return text
}

// generateID generates an ID from a title, matching goldmark's WithAutoHeadingID behavior
// This follows GitHub's heading ID generation algorithm
func (b *Builder) generateID(title string) string {
	// First remove markdown formatting
	text := b.removeMarkdownFormatting(title)

	// Convert to lowercase
	id := strings.ToLower(text)

	// Replace spaces and special characters with hyphens
	// This matches GitHub's behavior: any sequence of non-alphanumeric characters becomes a single hyphen
	id = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(id, "-")

	// Remove leading and trailing hyphens
	id = strings.Trim(id, "-")

	// If empty after processing, use a default
	if id == "" {
		id = "heading"
	}

	return id
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, data, info.Mode())
	})
}
