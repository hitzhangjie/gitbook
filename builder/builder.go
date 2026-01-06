package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hitzhangjie/gitbook/book"
)

// Builder builds a GitBook project
type Builder struct {
	Book      *book.Book
	OutputDir string
}

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

	return &Builder{
		Book:      b,
		OutputDir: absOutput,
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

	return nil
}

func (b *Builder) generatePages() error {
	if b.Book.Summary == nil {
		return nil
	}

	return b.generateChapters(b.Book.Summary.Chapters, "")
}

func (b *Builder) generateChapters(chapters []book.Chapter, basePath string) error {
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

		// Generate HTML page
		htmlPath := strings.TrimSuffix(chapter.Path, ".md") + ".html"
		if basePath != "" {
			htmlPath = filepath.Join(basePath, htmlPath)
		}
		outputPath := filepath.Join(b.OutputDir, htmlPath)

		// Create directory if needed
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return err
		}

		// Generate full HTML page
		fullHTML := b.generatePageHTML(chapter.Title, html, chapter.Path)

		if err := os.WriteFile(outputPath, []byte(fullHTML), 0644); err != nil {
			return err
		}

		// Recursively generate sub-chapters
		if len(chapter.Articles) > 0 {
			if err := b.generateChapters(chapter.Articles, filepath.Dir(htmlPath)); err != nil {
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

	var content string
	if data, err := os.ReadFile(readmePath); err == nil {
		html, err := b.markdownToHTML(string(data))
		if err == nil {
			content = html
		}
	}

	title := "GitBook"
	if b.Book.Config != nil && b.Book.Config.Title != "" {
		title = b.Book.Config.Title
	}

	html := b.generatePageHTML(title, content, "README.md")
	return os.WriteFile(filepath.Join(b.OutputDir, "index.html"), []byte(html), 0644)
}

func (b *Builder) markdownToHTML(md string) (string, error) {
	// Simple markdown to HTML conversion
	// In a full implementation, you would use a proper markdown library
	html := md
	html = strings.ReplaceAll(html, "\n\n", "</p><p>")
	html = strings.ReplaceAll(html, "\n", "<br>")
	html = strings.ReplaceAll(html, "**", "<strong>")
	html = strings.ReplaceAll(html, "*", "<em>")
	html = "<p>" + html + "</p>"
	return html, nil
}

func (b *Builder) generatePageHTML(title, content, path string) string {
	// Generate a simple HTML page
	// In a full implementation, you would use templates
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>%s</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        h1, h2, h3 { color: #333; }
        code { background: #f4f4f4; padding: 2px 4px; border-radius: 3px; }
        pre { background: #f4f4f4; padding: 10px; border-radius: 5px; overflow-x: auto; }
    </style>
</head>
<body>
    <h1>%s</h1>
    %s
</body>
</html>`, title, title, content)
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
