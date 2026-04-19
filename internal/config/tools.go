package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ToolsFile represents the ferry-tools.yaml / ~/.ferry/tools.yaml config.
// It lets users define custom formatters, linters, and LSPs without modifying
// ferry source code. Project-level (ferry-tools.yaml) wins on conflicts.
type ToolsFile struct {
	Formatters map[string]CustomTool `yaml:"formatters"`
	Linters    map[string]CustomTool `yaml:"linters"`
	LSPs       map[string]CustomTool `yaml:"lsps"`
}

// CustomTool defines how to build/install a custom tool.
// Build is a shell command run inside the Docker container (Linux builds).
// MacOSDownloads provides direct download specs for darwin build tracks.
type CustomTool struct {
	Build          string               `yaml:"build"`
	MacOSDownloads []CustomMacOSDownload `yaml:"macos_downloads"`
}

// CustomMacOSDownload describes a direct binary/archive download for a macOS arch.
type CustomMacOSDownload struct {
	Arch        string `yaml:"arch"`         // "x86_64" | "arm64" | "universal"
	URL         string `yaml:"url"`          // direct download URL
	ArchiveRoot string `yaml:"archive_root"` // subdirectory prefix to strip; "" = root
	InnerPath   string `yaml:"inner_path"`   // path to binary inside archive; "" = direct binary
}

// LoadToolsFile loads and merges global (~/.ferry/tools.yaml) and project-level
// (ferry-tools.yaml) config files. Project-level wins on conflicts.
// Returns a non-nil ToolsFile even when no files are present.
func LoadToolsFile() (*ToolsFile, error) {
	merged := &ToolsFile{
		Formatters: make(map[string]CustomTool),
		Linters:    make(map[string]CustomTool),
		LSPs:       make(map[string]CustomTool),
	}

	// Load global first, then project-level so project wins.
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolving home dir: %w", err)
	}

	globalPath := filepath.Join(home, ".ferry", "tools.yaml")
	if err := loadAndMerge(globalPath, merged); err != nil {
		return nil, fmt.Errorf("reading %s: %w", globalPath, err)
	}

	projectPath := "ferry-tools.yaml"
	if err := loadAndMerge(projectPath, merged); err != nil {
		return nil, fmt.Errorf("reading %s: %w", projectPath, err)
	}

	return merged, nil
}

// loadAndMerge parses a YAML tools file and merges its entries into dst.
// Missing file is silently ignored.
func loadAndMerge(path string, dst *ToolsFile) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	var tf ToolsFile
	if err := yaml.Unmarshal(data, &tf); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	for k, v := range tf.Formatters {
		dst.Formatters[k] = v
	}
	for k, v := range tf.Linters {
		dst.Linters[k] = v
	}
	for k, v := range tf.LSPs {
		dst.LSPs[k] = v
	}
	return nil
}

// HasFormatter returns true if name is defined in the custom tools file.
func (tf *ToolsFile) HasFormatter(name string) bool {
	if tf == nil {
		return false
	}
	_, ok := tf.Formatters[name]
	return ok
}

// HasLinter returns true if name is defined in the custom tools file.
func (tf *ToolsFile) HasLinter(name string) bool {
	if tf == nil {
		return false
	}
	_, ok := tf.Linters[name]
	return ok
}

// HasLSP returns true if name is defined in the custom tools file.
func (tf *ToolsFile) HasLSP(name string) bool {
	if tf == nil {
		return false
	}
	_, ok := tf.LSPs[name]
	return ok
}
