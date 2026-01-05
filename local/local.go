package local

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hitzhangjie/gitbook/config"
	"github.com/hitzhangjie/gitbook/tags"
)

type VersionInfo struct {
	Name    string // The name associated in the folder
	Version string // The real absolute version
	Path    string // Location of this version
	Link    string // Location if it's a symlink (empty if not)
	Tag     string // Type of release, latest, beta, etc ?
}

// Versions returns a list of all available versions on this system
func Versions() ([]VersionInfo, error) {
	entries, err := os.ReadDir(config.VERSIONS_ROOT)
	if err != nil {
		if os.IsNotExist(err) {
			return []VersionInfo{}, nil
		}
		return nil, err
	}

	var versions []VersionInfo

	for _, entry := range entries {
		tag := entry.Name()

		// Version matches requirements?
		if !tags.IsValid(tag, config.GITBOOK_VERSION) {
			continue
		}

		versionFolder := filepath.Join(config.VERSIONS_ROOT, tag)
		info, err := os.Lstat(versionFolder)
		if err != nil {
			continue
		}

		var linkPath string
		if info.Mode()&os.ModeSymlink != 0 {
			linkPath, err = os.Readlink(versionFolder)
			if err != nil {
				continue
			}
		}

		// Read package.json to determine version
		packageJsonPath := filepath.Join(versionFolder, "package.json")
		packageJsonData, err := os.ReadFile(packageJsonPath)
		if err != nil {
			continue
		}

		var packageJson struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}
		if err := json.Unmarshal(packageJsonData, &packageJson); err != nil {
			continue
		}

		// Is it gitbook?
		if packageJson.Name != "gitbook" {
			continue
		}

		versions = append(versions, VersionInfo{
			Name:    tag,
			Version: packageJson.Version,
			Path:    versionFolder,
			Link:    linkPath,
			Tag:     tags.GetTag(packageJson.Version),
		})
	}

	// Sort by version
	for i := 0; i < len(versions)-1; i++ {
		for j := i + 1; j < len(versions); j++ {
			if tags.Sort(versions[i].Version, versions[j].Version) < 0 {
				versions[i], versions[j] = versions[j], versions[i]
			}
		}
	}

	return versions, nil
}

// VersionRoot returns path to a specific version
func VersionRoot(version string) string {
	return filepath.Join(config.VERSIONS_ROOT, version)
}

// Resolve resolves a version using a condition
func Resolve(condition string) (*VersionInfo, error) {
	versions, err := Versions()
	if err != nil {
		return nil, err
	}

	for _, v := range versions {
		if tags.Satisfies(v.Name, condition, true) {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("no version match: %s", condition)
}

// Remove removes an installed version of gitbook
func Remove(version string) error {
	if version == "" {
		return fmt.Errorf("no version specified")
	}

	outputFolder := VersionRoot(version)
	info, err := os.Lstat(outputFolder)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("version %s is not installed", version)
		}
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return os.Remove(outputFolder)
	}

	return os.RemoveAll(outputFolder)
}

// Load loads a gitbook version
func Load(version interface{}) (string, error) {
	var resolved *VersionInfo
	var err error

	switch v := version.(type) {
	case string:
		resolved, err = Resolve(v)
		if err != nil {
			return "", err
		}
	case *VersionInfo:
		resolved = v
	default:
		return "", fmt.Errorf("invalid version type")
	}

	// Verify the path exists
	if _, err := os.Stat(resolved.Path); err != nil {
		return "", fmt.Errorf("GitBook version %s is corrupted: %w", resolved.Tag, err)
	}

	return resolved.Path, nil
}

// Link links a folder to a tag
func Link(name, folder string) error {
	if name == "" {
		return fmt.Errorf("require a name to represent this GitBook version")
	}
	if folder == "" {
		return fmt.Errorf("require a folder")
	}

	absFolder, err := filepath.Abs(folder)
	if err != nil {
		return err
	}

	outputFolder := VersionRoot(name)

	// Remove existing if it exists
	if _, err := os.Lstat(outputFolder); err == nil {
		if err := os.Remove(outputFolder); err != nil {
			return err
		}
	}

	return os.Symlink(absFolder, outputFolder)
}
