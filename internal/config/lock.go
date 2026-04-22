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
type ProfileConfig struct {
	Description string           `toml:"description"`
	Languages   []LanguageConfig `toml:"languages"`
	Plugins     []string         `toml:"plugins"`
	CLI         []string         `toml:"cli"`
	IncludeNvim *bool            `toml:"include_nvim,omitempty"`
	Shell       *ShellProfile    `toml:"shell,omitempty"`
}

// NvimEnabled reports whether nvim should be bundled for this profile.
// A nil pointer (field absent from ferry.lock) means true for backward compatibility.
func (p ProfileConfig) NvimEnabled() bool {
	return p.IncludeNvim == nil || *p.IncludeNvim
}

// ShellProfile describes the zsh setup to bundle for a profile.
// nil means no shell bundling.
type ShellProfile struct {
	Type            string `toml:"type"`                        // always "zsh"
	Framework       string `toml:"framework,omitempty"`         // "oh-my-zsh" | "zinit" | "zplug" | ""
	FrameworkPath   string `toml:"framework_path,omitempty"`
	RCPath          string `toml:"rc_path,omitempty"`
	Theme           string `toml:"theme,omitempty"`             // "p10k" | "starship" | "pure" | ""
	ThemeConfigPath string `toml:"theme_config_path,omitempty"`
}

type NvimConfig struct {
	Version string `toml:"version"`
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
				Description: "full dev environment",
				Languages:   []LanguageConfig{},
				IncludeNvim: boolPtr(true),
				CLI:         []string{"rg", "fzf", "zoxide", "jq"},
			},
			"minimal": {
				Description: "nvim + shell only",
				IncludeNvim: boolPtr(true),
			},
			"server": {
				Description: "shell and CLI tools, lightweight nvim",
				IncludeNvim: boolPtr(true),
				CLI:         []string{"rg", "fzf", "jq"},
			},
		},
	}
}

func boolPtr(b bool) *bool { return &b }
