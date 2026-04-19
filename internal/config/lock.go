package config

import (
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// LockFile is the top-level ferry.lock structure.
// Languages and CLI tools are now per-profile; there are no global language settings.
type LockFile struct {
	Nvim     NvimConfig               `toml:"nvim"`
	Shell    ShellConfig              `toml:"shell"`
	Bundle   BundleConfig             `toml:"bundle"`
	Profiles map[string]ProfileConfig `toml:"profiles"`
}

// LanguageConfig is a language selection within a profile.
// Tier selects between full runtime and LSP-only.
// All fields except Name and Tier are optional — empty means use registry defaults.
type LanguageConfig struct {
	Name           string   `toml:"name"`
	Tier           string   `toml:"tier"` // "full" | "lsp-only"
	LSP            string   `toml:"lsp,omitempty"`
	Formatters     []string `toml:"formatters,omitempty"`
	Linters        []string `toml:"linters,omitempty"`
	RuntimeVersion string   `toml:"runtime_version,omitempty"`
	ExtraPackages  []string `toml:"extra_packages,omitempty"`
}

// ProfileConfig describes a complete, independently deployable environment.
// Every field is explicit — no IncludeAll flags.
type ProfileConfig struct {
	Description  string           `toml:"description"`
	Languages    []LanguageConfig `toml:"languages"`
	Plugins      []string         `toml:"plugins"`
	CLI          []string         `toml:"cli"`
	IncludeShell bool             `toml:"include_shell"`
}

type NvimConfig struct {
	Version string `toml:"version"`
}

type ShellConfig struct {
	Type              string `toml:"type"`
	Config            string `toml:"config"`
	Theme             string `toml:"theme"`
	ThemeConfig       string `toml:"theme_config"`
	PluginManager     string `toml:"plugin_manager"`
	PluginManagerPath string `toml:"plugin_manager_path"`
}

type BundleConfig struct {
	Compress string         `toml:"compress"`
	Exclude  []string       `toml:"exclude"`
	Requires BundleRequires `toml:"requires"`
}

type BundleRequires struct {
	Packages      []string `toml:"packages"`
	MinZshVersion string   `toml:"min_zsh_version"`
}

// ReadLockFile reads and parses ferry.lock from the current directory.
func ReadLockFile() (*LockFile, error) {
	var lf LockFile
	_, err := toml.DecodeFile(LockFilePath(), &lf)
	if err != nil {
		return nil, fmt.Errorf("reading ferry.lock: %w", err)
	}
	return &lf, nil
}

// WriteLockFile writes a LockFile to ferry.lock in the current directory.
func WriteLockFile(lf *LockFile) error {
	f, err := os.Create(LockFilePath())
	if err != nil {
		return fmt.Errorf("creating ferry.lock: %w", err)
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(lf)
}

// LockFileHash returns the SHA256 hash of the raw ferry.lock bytes.
func LockFileHash() (string, error) {
	data, err := os.ReadFile(LockFilePath())
	if err != nil {
		return "", fmt.Errorf("reading ferry.lock for hash: %w", err)
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum), nil
}

// DefaultLockFile returns a LockFile with sensible defaults applied.
func DefaultLockFile() *LockFile {
	return &LockFile{
		Nvim: NvimConfig{Version: "0.10.4"},
		Bundle: BundleConfig{
			Compress: "zstd",
			Exclude: []string{
				"lazy/*/test/",
				"lazy/*/.git/",
				"lazy/*/doc/",
				"lazy/nvim-treesitter/parser/*.so",
			},
			Requires: BundleRequires{
				Packages:      []string{"git", "curl", "tmux", "zsh"},
				MinZshVersion: "5.8",
			},
		},
		Profiles: map[string]ProfileConfig{
			"default": {
				Description:  "full dev environment",
				Languages:    []LanguageConfig{},
				IncludeShell: true,
				CLI:          []string{"rg", "fzf", "zoxide", "jq"},
			},
			"minimal": {
				Description:  "nvim + shell only",
				IncludeShell: true,
			},
			"server": {
				Description:  "shell and CLI tools, lightweight nvim",
				IncludeShell: true,
				CLI:          []string{"rg", "fzf", "jq"},
			},
		},
	}
}
