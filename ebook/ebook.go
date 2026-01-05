package ebook

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hitzhangjie/gitbook/builder"
)

// Generator generates ebooks (PDF, EPUB, MOBI)
type Generator struct {
	BookRoot  string
	OutputDir string
	Format    string // pdf, epub, mobi
}

// NewGenerator creates a new ebook generator
func NewGenerator(bookRoot, outputDir, format string) (*Generator, error) {
	if outputDir == "" {
		outputDir = filepath.Join(bookRoot, "_book")
	}

	absOutput, err := filepath.Abs(outputDir)
	if err != nil {
		return nil, err
	}

	return &Generator{
		BookRoot:  bookRoot,
		OutputDir: absOutput,
		Format:    format,
	}, nil
}

// Generate generates an ebook
func (g *Generator) Generate(outputPath string) error {
	// First build the book
	builder, err := builder.NewBuilder(g.BookRoot, g.OutputDir)
	if err != nil {
		return fmt.Errorf("failed to create builder: %w", err)
	}

	if err := builder.Build(); err != nil {
		return fmt.Errorf("failed to build book: %w", err)
	}

	// Generate ebook using calibre or pandoc
	switch g.Format {
	case "pdf":
		return g.generatePDF(outputPath)
	case "epub":
		return g.generateEPUB(outputPath)
	case "mobi":
		return g.generateMOBI(outputPath)
	default:
		return fmt.Errorf("unsupported format: %s", g.Format)
	}
}

func (g *Generator) generatePDF(outputPath string) error {
	// Try to use calibre's ebook-convert
	if _, err := exec.LookPath("ebook-convert"); err == nil {
		return g.convertWithCalibre("pdf", outputPath)
	}

	// Try to use pandoc
	if _, err := exec.LookPath("pandoc"); err == nil {
		return g.convertWithPandoc("pdf", outputPath)
	}

	return fmt.Errorf("neither calibre (ebook-convert) nor pandoc found. Please install one of them")
}

func (g *Generator) generateEPUB(outputPath string) error {
	// Try to use calibre's ebook-convert
	if _, err := exec.LookPath("ebook-convert"); err == nil {
		return g.convertWithCalibre("epub", outputPath)
	}

	// Try to use pandoc
	if _, err := exec.LookPath("pandoc"); err == nil {
		return g.convertWithPandoc("epub", outputPath)
	}

	return fmt.Errorf("neither calibre (ebook-convert) nor pandoc found. Please install one of them")
}

func (g *Generator) generateMOBI(outputPath string) error {
	// Try to use calibre's ebook-convert
	if _, err := exec.LookPath("ebook-convert"); err == nil {
		return g.convertWithCalibre("mobi", outputPath)
	}

	return fmt.Errorf("calibre (ebook-convert) not found. Please install calibre")
}

func (g *Generator) convertWithCalibre(format, outputPath string) error {
	indexPath := filepath.Join(g.OutputDir, "index.html")
	
	cmd := exec.Command("ebook-convert", indexPath, outputPath)
	cmd.Dir = g.OutputDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

func (g *Generator) convertWithPandoc(format, outputPath string) error {
	indexPath := filepath.Join(g.OutputDir, "index.html")
	
	var pandocFormat string
	switch format {
	case "pdf":
		pandocFormat = "pdf"
	case "epub":
		pandocFormat = "epub"
	default:
		return fmt.Errorf("pandoc does not support format: %s", format)
	}
	
	cmd := exec.Command("pandoc", indexPath, "-o", outputPath, "-t", pandocFormat)
	cmd.Dir = g.OutputDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

