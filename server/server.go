package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"github.com/hitzhangjie/gitbook/book"
	"github.com/hitzhangjie/gitbook/builder"
)

// Server represents a GitBook development server
type Server struct {
	Book             *book.Book
	Port             int
	Host             string
	OutputDir        string
	httpServer       *http.Server
	watcher          *fsnotify.Watcher
	builder          *builder.Builder
	clients          map[*websocket.Conn]bool
	clientsMutex     sync.RWMutex
	rebuildDebouncer *time.Timer
	rebuildMutex     sync.Mutex
}

// UpdateMessage represents a message sent to clients
type UpdateMessage struct {
	Type    string `json:"type"` // "rebuild_start", "rebuild_complete", "rebuild_error"
	Message string `json:"message,omitempty"`
	Path    string `json:"path,omitempty"`
}

const defaultHTTPAddr = "localhost:4000"

// NewServer creates a new development server. httpAddr is the listen address (e.g. "localhost:4000" or "0.0.0.0:8080");
// if empty, default "localhost:4000" is used.
func NewServer(bookRoot string, httpAddr string) (*Server, error) {
	if httpAddr == "" {
		httpAddr = defaultHTTPAddr
	}
	host, portStr, err := net.SplitHostPort(httpAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid http address %q: %w", httpAddr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid port in http address %q", httpAddr)
	}

	b, err := book.LoadBook(bookRoot)
	if err != nil {
		return nil, err
	}

	outputDir := filepath.Join(bookRoot, "_book")

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	return &Server{
		Book:      b,
		Port:      port,
		Host:      host,
		OutputDir: outputDir,
		watcher:   watcher,
		clients:   make(map[*websocket.Conn]bool),
	}, nil
}

// Start starts the development server
func (s *Server) Start() error {
	// Build the book first
	b, err := builder.NewBuilder(s.Book.Root, s.OutputDir)
	if err != nil {
		return fmt.Errorf("failed to create builder: %w", err)
	}
	s.builder = b

	if err := s.builder.Build(); err != nil {
		return fmt.Errorf("failed to build book: %w", err)
	}

	// Start file watcher
	if err := s.startWatcher(); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	// Setup HTTP server
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", s.handleWebSocket)

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
	fmt.Println("Live reload enabled - watching for file changes...")
	fmt.Println("Press Ctrl+C to stop the server")

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
		<-sigChan
		fmt.Println("\nShutting down...")
		s.Stop()
	}()

	if err := s.httpServer.ListenAndServe(); err != nil {
		if err != http.ErrServerClosed {
			return fmt.Errorf("failed to start server: %w", err)
		}
		// log.Printf("Server closed: %v", err)
	}

	// Clean temporary _book when Start() returns
	if err := os.RemoveAll(s.OutputDir); err != nil {
		log.Printf("Failed to remove _book: %v", err)
	} else {
		log.Println("Cleaned up _book")
	}
	return err
}

// Stop stops the server
func (s *Server) Stop() error {
	// Close watcher
	if s.watcher != nil {
		s.watcher.Close()
	}

	// Close all WebSocket connections
	s.clientsMutex.Lock()
	for conn := range s.clients {
		conn.Close()
		delete(s.clients, conn)
	}
	s.clientsMutex.Unlock()

	// Stop HTTP server
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

// startWatcher starts watching for file changes
func (s *Server) startWatcher() error {
	// Watch the book root directory recursively
	err := filepath.Walk(s.Book.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip output directory and hidden directories
		base := filepath.Base(path)
		if base == "_book" || base == ".git" || strings.HasPrefix(base, ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Only watch directories (files will be watched through their parent directory)
		if info.IsDir() {
			return s.watcher.Add(path)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Start watching for events in a goroutine
	go s.watchEvents()

	return nil
}

// watchEvents handles file system events
func (s *Server) watchEvents() {
	for {
		select {
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}

			// Only handle write and create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				// Check if it's a relevant file
				if s.shouldRebuild(event.Name) {
					// Add new directories to watcher
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						s.watcher.Add(event.Name)
					}

					// Debounce rebuilds
					s.triggerRebuild(event.Name)
				}
			}

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

// shouldRebuild checks if a file change should trigger a rebuild
func (s *Server) shouldRebuild(path string) bool {
	// Skip output directory
	if strings.Contains(path, "_book") {
		return false
	}

	ext := strings.ToLower(filepath.Ext(path))
	base := filepath.Base(path)

	// Watch markdown files
	if ext == ".md" || ext == ".markdown" {
		return true
	}

	// Watch configuration files
	if base == "book.json" || base == "SUMMARY.md" || base == "README.md" {
		return true
	}

	// Watch static assets (images, etc.)
	if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".svg" ||
		ext == ".css" || ext == ".js" {
		return true
	}

	return false
}

// triggerRebuild triggers a rebuild with debouncing
func (s *Server) triggerRebuild(changedPath string) {
	s.rebuildMutex.Lock()
	defer s.rebuildMutex.Unlock()

	// Cancel previous debounce timer if exists
	if s.rebuildDebouncer != nil {
		s.rebuildDebouncer.Stop()
	}

	// Set new debounce timer (300ms)
	s.rebuildDebouncer = time.AfterFunc(300*time.Millisecond, func() {
		s.rebuild(changedPath)
	})
}

// rebuild rebuilds the book and notifies clients
func (s *Server) rebuild(changedPath string) {
	// Notify clients that rebuild started
	s.broadcast(UpdateMessage{
		Type:    "rebuild_start",
		Message: "Rebuilding...",
		Path:    changedPath,
	})

	// Rebuild the book
	err := s.builder.Build()
	if err != nil {
		log.Printf("Rebuild error: %v", err)
		s.broadcast(UpdateMessage{
			Type:    "rebuild_error",
			Message: fmt.Sprintf("Rebuild failed: %v", err),
			Path:    changedPath,
		})
		return
	}

	// Notify clients that rebuild completed
	s.broadcast(UpdateMessage{
		Type:    "rebuild_complete",
		Message: "Rebuild complete",
		Path:    changedPath,
	})
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Register client
	s.clientsMutex.Lock()
	s.clients[conn] = true
	s.clientsMutex.Unlock()

	// Send initial connection message
	conn.WriteJSON(UpdateMessage{
		Type:    "connected",
		Message: "Connected to live reload",
	})

	// Handle client disconnection
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				s.clientsMutex.Lock()
				delete(s.clients, conn)
				s.clientsMutex.Unlock()
				conn.Close()
				return
			}
		}
	}()
}

// broadcast sends a message to all connected clients
func (s *Server) broadcast(msg UpdateMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	s.clientsMutex.RLock()
	defer s.clientsMutex.RUnlock()

	for conn := range s.clients {
		err := conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Printf("Failed to send message to client: %v", err)
			// Remove dead connection
			delete(s.clients, conn)
			conn.Close()
		}
	}
}
