package book

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Book represents a GitBook project configuration
type Book struct {
	Root    string
	Config  *Config
	Summary *Summary
}

// Config represents book.json configuration
type Config struct {
	Title         string                 `json:"title,omitempty"`
	Author        string                 `json:"author,omitempty"`
	Description   string                 `json:"description,omitempty"`
	Language      string                 `json:"language,omitempty"`
	Gitbook       string                 `json:"gitbook,omitempty"`
	Root          string                 `json:"root,omitempty"`
	Structure     *Structure             `json:"structure,omitempty"`
	Plugins       []string               `json:"plugins,omitempty"`
	PluginsConfig map[string]interface{} `json:"pluginsConfig,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
}

// Structure defines custom file structure
type Structure struct {
	Readme    string `json:"readme,omitempty"`
	Summary   string `json:"summary,omitempty"`
	Glossary  string `json:"glossary,omitempty"`
	Languages string `json:"languages,omitempty"`
}

// Summary represents the SUMMARY.md structure
type Summary struct {
	Chapters []Chapter
}

// Chapter represents a chapter in the summary
type Chapter struct {
	Title    string
	Path     string
	Articles []Chapter
}

// LoadBook loads a book from a directory
func LoadBook(root string) (*Book, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	book := &Book{
		Root: absRoot,
	}

	// Load book.json
	configPath := filepath.Join(absRoot, "book.json")
	config, err := LoadConfig(configPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	book.Config = config

	// Load SUMMARY.md
	summaryPath := filepath.Join(absRoot, "SUMMARY.md")
	if book.Config != nil && book.Config.Structure != nil && book.Config.Structure.Summary != "" {
		summaryPath = filepath.Join(absRoot, book.Config.Structure.Summary)
	}
	summary, err := LoadSummary(summaryPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	book.Summary = summary

	return book, nil
}

// LoadConfig loads book.json
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse book.json: %w", err)
	}

	return &config, nil
}

// SaveConfig saves book.json
func SaveConfig(path string, config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// LoadSummary loads SUMMARY.md
func LoadSummary(path string) (*Summary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	summary, err := ParseSummary(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse SUMMARY.md: %w", err)
	}

	return summary, nil
}

// ParseSummary parses SUMMARY.md content
func ParseSummary(content string) (*Summary, error) {
	// Simple parser for SUMMARY.md
	// Format: * [Title](path)
	summary := &Summary{
		Chapters: []Chapter{},
	}

	lines := splitLines(content)
	var stack []*[]Chapter
	stack = append(stack, &summary.Chapters)

	for _, line := range lines {
		line = trimSpace(line)
		if line == "" || !strings.HasPrefix(line, "*") {
			continue
		}

		// Parse markdown link: * [Title](path)
		title, path := parseSummaryLine(line)
		if title == "" {
			continue
		}

		chapter := Chapter{
			Title:    title,
			Path:     path,
			Articles: []Chapter{},
		}

		// Determine nesting level
		level := getIndentLevel(line)
		for len(stack) > level+1 {
			stack = stack[:len(stack)-1]
		}
		for len(stack) <= level {
			// Create new level
			if len(stack) == 0 {
				stack = append(stack, &summary.Chapters)
			} else {
				lastChapters := stack[len(stack)-1]
				if len(*lastChapters) > 0 {
					lastChapter := &(*lastChapters)[len(*lastChapters)-1]
					stack = append(stack, &lastChapter.Articles)
				} else {
					stack = append(stack, &summary.Chapters)
				}
			}
		}

		*stack[level] = append(*stack[level], chapter)
	}

	return summary, nil
}

// Helper functions
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, r := range s {
		if r == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func parseSummaryLine(line string) (title, path string) {
	// Remove leading * and spaces
	line = strings.TrimLeft(line, "* \t")

	// Parse [Title](path)
	titleStart := strings.Index(line, "[")
	titleEnd := strings.Index(line, "]")
	pathStart := strings.Index(line, "(")
	pathEnd := strings.Index(line, ")")

	if titleStart >= 0 && titleEnd > titleStart && pathStart > titleEnd && pathEnd > pathStart {
		title = line[titleStart+1 : titleEnd]
		path = line[pathStart+1 : pathEnd]
	}

	return
}

func getIndentLevel(line string) int {
	level := 0
	for _, r := range line {
		if r == ' ' {
			level++
		} else if r == '\t' {
			level += 4
		} else if r == '*' {
			return level / 2 // Assuming 2 spaces per level
		} else {
			break
		}
	}
	return 0
}
