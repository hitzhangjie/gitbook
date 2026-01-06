package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hitzhangjie/gitbook/book"
	"github.com/hitzhangjie/gitbook/builder"
)

// Server represents a GitBook development server
type Server struct {
	Book       *book.Book
	Port       int
	Host       string
	OutputDir  string
	httpServer *http.Server
}

// NewServer creates a new development server
func NewServer(bookRoot string) (*Server, error) {
	port := 4000
	host := "localhost"

	b, err := book.LoadBook(bookRoot)
	if err != nil {
		return nil, err
	}

	outputDir := filepath.Join(bookRoot, "_book")

	return &Server{
		Book:      b,
		Port:      port,
		Host:      host,
		OutputDir: outputDir,
	}, nil
}

// Start starts the development server
func (s *Server) Start() error {
	// Build the book first
	builder, err := builder.NewBuilder(s.Book.Root, s.OutputDir)
	if err != nil {
		return fmt.Errorf("failed to create builder: %w", err)
	}

	if err := builder.Build(); err != nil {
		return fmt.Errorf("failed to build book: %w", err)
	}

	// Setup HTTP server
	mux := http.NewServeMux()

	// Serve static files
	mux.HandleFunc("/", s.handleRequest)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.Host, s.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("Serving book on http://%s:%d\n", s.Host, s.Port)
	fmt.Println("Press Ctrl+C to stop the server")

	return s.httpServer.ListenAndServe()
}

// Stop stops the server
func (s *Server) Stop() error {
	if s.httpServer != nil {
		return s.httpServer.Close()
	}
	return nil
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	// Remove leading slash
	path = strings.TrimPrefix(path, "/")

	// Check if file exists
	fp := filepath.Join(s.OutputDir, path)

	// Security check: ensure path is within output directory
	absOutput, _ := filepath.Abs(s.OutputDir)
	absFile, _ := filepath.Abs(fp)
	if !strings.HasPrefix(absFile, absOutput) {
		http.NotFound(w, r)
		return
	}

	// Check if it's a directory, serve index.html
	info, err := os.Stat(fp)
	if err == nil && info.IsDir() {
		fp = filepath.Join(fp, "index.html")
	}

	// Serve file
	if _, err := os.Stat(fp); err == nil {
		http.ServeFile(w, r, fp)
		return
	}

	// Try .html extension
	if !strings.HasSuffix(fp, ".html") {
		htmlPath := fp + ".html"
		if _, err := os.Stat(htmlPath); err == nil {
			http.ServeFile(w, r, htmlPath)
			return
		}
	}

	http.NotFound(w, r)
}
