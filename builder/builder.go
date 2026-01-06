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
	// Copy images, fonts, etc.
	assetsDirs := []string{"images", "fonts", "assets"}
	for _, dir := range assetsDirs {
		src := filepath.Join(b.Book.Root, dir)
		if _, err := os.Stat(src); err == nil {
			dst := filepath.Join(b.OutputDir, dir)
			if err := copyDir(src, dst); err != nil {
				return err
			}
		}
	}

	// Also copy .jpg, .jpeg, .png files in Book.Root to OutputDir (for book covers)
	imagePatterns := []string{"*.jpg", "*.jpeg", "*.png"}
	for _, pattern := range imagePatterns {
		matches, err := filepath.Glob(filepath.Join(b.Book.Root, pattern))
		if err != nil {
			continue
		}
		for _, src := range matches {
			base := filepath.Base(src)
			dst := filepath.Join(b.OutputDir, base)
			data, err := os.ReadFile(src)
			if err != nil {
				continue
			}
			if err := os.WriteFile(dst, data, 0644); err != nil {
				continue
			}
		}
	}

	return nil
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

		// Extract TOC from markdown
		toc := b.extractTOC(string(content))

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
			toc = b.extractTOC(string(data))
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
	var items []NavItem
	for _, chapter := range chapters {
		item := NavItem{
			Title: chapter.Title,
			Path:  chapter.Path,
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
			item.Children = b.buildNavTree(chapter.Articles, nextBasePath)
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

func (b *Builder) extractTOC(md string) []TOCItem {
	var toc []TOCItem
	lines := strings.Split(md, "\n")

	// Regex to match markdown headers
	headerRegex := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	var stack []*TOCItem

	for _, line := range lines {
		matches := headerRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		level := len(matches[1])
		title := strings.TrimSpace(matches[2])

		// Generate ID from title
		id := b.generateID(title)

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

func (b *Builder) generateID(title string) string {
	// Convert title to ID (similar to GitHub's heading ID generation)
	id := strings.ToLower(title)
	id = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(id, "-")
	id = strings.Trim(id, "-")
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
