package manager

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hitzhangjie/gitbook/local"
	"github.com/hitzhangjie/gitbook/registry"
	"github.com/hitzhangjie/gitbook/tags"
)

// BookVersion returns book version (string) required by a book
func BookVersion(bookRoot string) string {
	bookJsonPath := filepath.Join(bookRoot, "book.json")
	data, err := os.ReadFile(bookJsonPath)
	if err != nil {
		return "*"
	}

	var bookJson struct {
		Gitbook string `json:"gitbook"`
	}
	if err := json.Unmarshal(data, &bookJson); err != nil {
		return "*"
	}

	if bookJson.Gitbook != "" {
		return bookJson.Gitbook
	}

	return "*"
}

// EnsureVersion ensures that a version exists or installs it
func EnsureVersion(bookRoot, version string, install bool) (*local.VersionInfo, error) {
	// If not defined, load version required from book.json
	if version == "" {
		version = BookVersion(bookRoot)
	}

	// Resolve version locally
	resolved, err := local.Resolve(version)
	if err != nil {
		if !install {
			return nil, err
		}

		// Install if needed
		_, installErr := registry.Install(version, false)
		if installErr != nil {
			return nil, installErr
		}

		// Retry after installation
		return EnsureVersion(bookRoot, version, false)
	}

	return resolved, nil
}

// GetVersion gets version in a book
func GetVersion(bookRoot, version string) (*local.VersionInfo, error) {
	return EnsureVersion(bookRoot, version, false)
}

// EnsureAndLoad ensures a version exists (or installs it), then loads it and returns the gitbook path
func EnsureAndLoad(bookRoot, version string) (string, error) {
	resolved, err := EnsureVersion(bookRoot, version, true)
	if err != nil {
		return "", err
	}

	return local.Load(resolved)
}

// UpdateVersion updates current version
func UpdateVersion(tag string) (string, error) {
	if tag == "" {
		tag = "latest"
	}

	// Get current version (if any)
	versions, err := local.Versions()
	var currentV *local.VersionInfo
	if err == nil && len(versions) > 0 {
		currentV = &versions[0] // First one is latest
	}

	// Get remote versions
	available, err := registry.Versions()
	if err != nil {
		return "", err
	}

	remoteVersion, ok := available.Tags[tag]
	if !ok {
		return "", fmt.Errorf("tag doesn't exist: %s", tag)
	}

	// Check if update is needed
	if currentV != nil {
		if tags.Sort(remoteVersion, currentV.Version) >= 0 {
			return "", nil // No update needed
		}
	}

	// Install new version
	installedVersion, err := registry.Install(remoteVersion, false)
	if err != nil {
		return "", err
	}

	// Remove previous version if exists
	if currentV != nil {
		_ = local.Remove(currentV.Tag)
	}

	return installedVersion, nil
}
