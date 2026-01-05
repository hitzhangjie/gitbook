package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hitzhangjie/gitbook/config"
	"github.com/hitzhangjie/gitbook/tags"
)

type NPMRegistryResponse struct {
	Versions map[string]interface{} `json:"versions"`
	DistTags map[string]string      `json:"dist-tags"`
}

type AvailableVersions struct {
	Versions []string
	Tags     map[string]string
}

// Versions returns a list of versions available in the registry (npm)
func Versions() (*AvailableVersions, error) {
	// Try to use npm command first, fallback to HTTP API
	result, err := fetchFromNPM()
	if err != nil {
		// Fallback to HTTP API
		return fetchFromHTTP()
	}
	return result, nil
}

func fetchFromNPM() (*AvailableVersions, error) {
	cmd := exec.Command("npm", "view", "gitbook", "versions", "dist-tags", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var npmData map[string]interface{}
	if err := json.Unmarshal(output, &npmData); err != nil {
		return nil, err
	}

	versionsRaw, ok := npmData["versions"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid npm response format")
	}

	distTagsRaw, ok := npmData["dist-tags"].(map[string]interface{})
	if !ok {
		distTagsRaw = make(map[string]interface{})
	}

	validVersions := []string{}
	for _, v := range versionsRaw {
		version, ok := v.(string)
		if !ok {
			continue
		}
		if tags.IsValid(version, config.GITBOOK_VERSION) {
			validVersions = append(validVersions, version)
		}
	}

	// Sort versions
	for i := 0; i < len(validVersions)-1; i++ {
		for j := i + 1; j < len(validVersions); j++ {
			if tags.Sort(validVersions[i], validVersions[j]) < 0 {
				validVersions[i], validVersions[j] = validVersions[j], validVersions[i]
			}
		}
	}

	validTags := make(map[string]string)
	for tag, version := range distTagsRaw {
		versionStr, ok := version.(string)
		if !ok {
			continue
		}
		if tags.IsValid(versionStr, config.GITBOOK_VERSION) {
			validTags[tag] = versionStr
		}
	}

	if len(validVersions) == 0 {
		return nil, fmt.Errorf("no valid version on the NPM registry")
	}

	return &AvailableVersions{
		Versions: validVersions,
		Tags:     validTags,
	}, nil
}

func fetchFromHTTP() (*AvailableVersions, error) {
	resp, err := http.Get("https://registry.npmjs.org/gitbook")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var npmData NPMRegistryResponse
	if err := json.Unmarshal(body, &npmData); err != nil {
		return nil, err
	}

	validVersions := []string{}
	for version := range npmData.Versions {
		if tags.IsValid(version, config.GITBOOK_VERSION) {
			validVersions = append(validVersions, version)
		}
	}

	// Sort versions
	for i := 0; i < len(validVersions)-1; i++ {
		for j := i + 1; j < len(validVersions); j++ {
			if tags.Sort(validVersions[i], validVersions[j]) < 0 {
				validVersions[i], validVersions[j] = validVersions[j], validVersions[i]
			}
		}
	}

	validTags := make(map[string]string)
	for tag, version := range npmData.DistTags {
		if tags.IsValid(version, config.GITBOOK_VERSION) {
			validTags[tag] = version
		}
	}

	if len(validVersions) == 0 {
		return nil, fmt.Errorf("no valid version on the NPM registry")
	}

	return &AvailableVersions{
		Versions: validVersions,
		Tags:     validTags,
	}, nil
}

// Resolve resolves a version name or tag to an installable absolute version
func Resolve(version string) (string, error) {
	available, err := Versions()
	if err != nil {
		return "", err
	}

	// Resolve if tag
	if resolvedVersion, ok := available.Tags[version]; ok {
		version = resolvedVersion
	}

	// Find matching version
	for _, v := range available.Versions {
		if tags.Satisfies(v, version, false) {
			return v, nil
		}
	}

	return "", fmt.Errorf("invalid version or tag \"%s\", see available using \"gitbook ls-remote\"", version)
}

// Install installs a specific version of gitbook
func Install(version string, forceInstall bool) (string, error) {
	resolvedVersion, err := Resolve(version)
	if err != nil {
		return "", err
	}

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "gitbook-install-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	fmt.Printf("Installing GitBook %s\n", resolvedVersion)

	// Install using npm
	cmd := exec.Command("npm", "install", "--prefix", tmpDir, "gitbook@"+resolvedVersion)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to install gitbook: %w", err)
	}

	gitbookRoot := filepath.Join(tmpDir, "node_modules", "gitbook")
	packageJsonPath := filepath.Join(gitbookRoot, "package.json")

	// Read package.json to get actual version
	packageJsonData, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return "", err
	}

	var packageJson struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(packageJsonData, &packageJson); err != nil {
		return "", err
	}

	if packageJson.Name != "gitbook" {
		return "", fmt.Errorf("installed package is not gitbook")
	}

	actualVersion := packageJson.Version
	if !tags.IsValid(actualVersion, config.GITBOOK_VERSION) {
		return "", fmt.Errorf("invalid GitBook version, should satisfy %s", config.GITBOOK_VERSION)
	}

	// Copy to the install folder
	outputFolder := filepath.Join(config.VERSIONS_ROOT, actualVersion)
	if err := os.RemoveAll(outputFolder); err != nil && !os.IsNotExist(err) {
		return "", err
	}

	// Copy directory
	if err := copyDir(gitbookRoot, outputFolder); err != nil {
		return "", err
	}

	return actualVersion, nil
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
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}
