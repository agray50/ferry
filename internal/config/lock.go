package config

import (
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type LockFile struct {
	Languages LanguagesConfig          `toml:"languages"`
	Nvim      NvimConfig               `toml:"nvim"`
	Shell     ShellConfig              `toml:"shell"`
	CLI       map[string]string        `toml:"cli"`
	Configs   []ConfigFile             `toml:"config"`
	Bundle    BundleConfig             `toml:"bundle"`
	Profiles  map[string]ProfileConfig `toml:"profiles"`
}

type LanguagesConfig struct {
	Enabled   []string                    `toml:"enabled"`
	Overrides map[string]LanguageOverride `toml:",remain"`
}

type LanguageOverride struct {
	LSP            string   `toml:"lsp"`
	RuntimeVersion string   `toml:"runtime_version"`
	ExtraPackages  []string `toml:"extra_packages"`
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

type ConfigFile struct {
	Source  string `toml:"source"`
	Target  string `toml:"target"`
	Encrypt bool   `toml:"encrypt"`
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

type ProfileConfig struct {
	Description       string   `toml:"description"`
	Languages         []string `toml:"languages"`
	Plugins           []string `toml:"plugins"`
	IncludeAllPlugins bool     `toml:"include_all_plugins"`
	IncludeShell      bool     `toml:"include_shell"`
	IncludeCLI        bool     `toml:"include_cli"`
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
				Description:       "full dev environment",
				IncludeAllPlugins: true,
				IncludeShell:      true,
				IncludeCLI:        true,
			},
			"minimal": {
				Description:  "nvim + shell only",
				IncludeShell: true,
				IncludeCLI:   false,
			},
			"server": {
				Description:  "shell and CLI tools, lightweight nvim",
				IncludeShell: true,
				IncludeCLI:   true,
			},
		},
	}
}
