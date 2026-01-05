package config

import (
	"os"
	"path/filepath"
)

var (
	CONFIG_ROOT     string
	VERSIONS_ROOT   string
	GITBOOK_VERSION = ">1.x.x"
)

// Init initializes and prepares configuration for gitbook-cli
// It creates the required folders
func Init() error {
	if CONFIG_ROOT == "" {
		SetRoot(getDefaultRoot())
	}

	if err := os.MkdirAll(CONFIG_ROOT, 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(VERSIONS_ROOT, 0755); err != nil {
		return err
	}

	return nil
}

// SetRoot replaces root folder to use
func SetRoot(root string) {
	CONFIG_ROOT = filepath.Clean(root)
	VERSIONS_ROOT = filepath.Join(CONFIG_ROOT, "versions")
}

func getDefaultRoot() string {
	if gitbookDir := os.Getenv("GITBOOK_DIR"); gitbookDir != "" {
		return filepath.Clean(gitbookDir)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home is not available
		return filepath.Join(".", ".gitbook")
	}

	return filepath.Join(homeDir, ".gitbook")
}
