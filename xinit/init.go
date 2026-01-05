package xinit

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hitzhangjie/gitbook/book"
)

// Init initializes a new GitBook project
func Init(bookRoot string) error {
	absRoot, err := filepath.Abs(bookRoot)
	if err != nil {
		return err
	}

	// Check if directory exists
	if _, err := os.Stat(absRoot); os.IsNotExist(err) {
		if err := os.MkdirAll(absRoot, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Check if book.json already exists
	configPath := filepath.Join(absRoot, "book.json")
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("book.json already exists in %s", absRoot)
	}

	// Create default book.json
	config := &book.Config{
		Title:   "My Book",
		Author:  "",
		Plugins: []string{},
	}

	if err := book.SaveConfig(configPath, config); err != nil {
		return fmt.Errorf("failed to create book.json: %w", err)
	}

	// Create README.md if it doesn't exist
	readmePath := filepath.Join(absRoot, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		readmeContent := `# Introduction

This is the introduction to your book.`
		if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
			return fmt.Errorf("failed to create README.md: %w", err)
		}
	}

	// Create SUMMARY.md if it doesn't exist
	summaryPath := filepath.Join(absRoot, "SUMMARY.md")
	if _, err := os.Stat(summaryPath); os.IsNotExist(err) {
		summaryContent := `# Summary

* [Introduction](README.md)`
		if err := os.WriteFile(summaryPath, []byte(summaryContent), 0644); err != nil {
			return fmt.Errorf("failed to create SUMMARY.md: %w", err)
		}
	}

	fmt.Printf("GitBook initialized in %s\n", absRoot)
	return nil
}
