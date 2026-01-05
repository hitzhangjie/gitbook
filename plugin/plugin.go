package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Plugin represents a GitBook plugin
type Plugin struct {
	Name    string
	Version string
	Path    string
}

// Install installs plugins for a book
func Install(bookRoot string) error {
	// Load book.json
	configPath := filepath.Join(bookRoot, "book.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("book.json not found: %w", err)
	}

	var config struct {
		Plugins []string `json:"plugins"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse book.json: %w", err)
	}

	if len(config.Plugins) == 0 {
		fmt.Println("No plugins to install")
		return nil
	}

	// Install plugins using npm
	pluginsDir := filepath.Join(bookRoot, "node_modules")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return fmt.Errorf("failed to create node_modules directory: %w", err)
	}

	// Build npm install command
	packages := []string{}
	for _, plugin := range config.Plugins {
		if plugin == "" || plugin == "-" {
			continue
		}
		// Remove gitbook-plugin- prefix if present
		pluginName := strings.TrimPrefix(plugin, "gitbook-plugin-")
		packages = append(packages, "gitbook-plugin-"+pluginName)
	}

	if len(packages) == 0 {
		fmt.Println("No valid plugins to install")
		return nil
	}

	fmt.Printf("Installing plugins: %s\n", strings.Join(packages, ", "))

	// Change to book root
	originalDir, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(bookRoot); err != nil {
		return err
	}

	// Run npm install
	cmd := exec.Command("npm", append([]string{"install", "--save"}, packages...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install plugins: %w", err)
	}

	fmt.Println("Plugins installed successfully")
	return nil
}

// List lists installed plugins
func List(bookRoot string) ([]Plugin, error) {
	pluginsDir := filepath.Join(bookRoot, "node_modules")
	var plugins []Plugin

	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Plugin{}, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "gitbook-plugin-") {
			continue
		}

		packagePath := filepath.Join(pluginsDir, entry.Name(), "package.json")
		data, err := os.ReadFile(packagePath)
		if err != nil {
			continue
		}

		var pkg struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}
		if err := json.Unmarshal(data, &pkg); err != nil {
			continue
		}

		plugins = append(plugins, Plugin{
			Name:    pkg.Name,
			Version: pkg.Version,
			Path:    filepath.Join(pluginsDir, entry.Name()),
		})
	}

	return plugins, nil
}

