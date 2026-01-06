package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hitzhangjie/gitbook/book"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewInitCommand creates the init command
func NewInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init [book]",
		Short: "Setup and initialize a book",
		Long:  "Initialize a book structure in the current directory or specified directory",
		Run: func(cmd *cobra.Command, args []string) {
			runCommand("init", cmd.Flags(), args)
		},
	}
}

func handleInit(bookRoot string, fset *pflag.FlagSet, args []string) error {
	initDir := bookRoot
	if len(args) > 0 {
		initDir = args[0]
	}
	return doInit(initDir)
}

// doInit initializes a new GitBook project
func doInit(bookRoot string) error {
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
