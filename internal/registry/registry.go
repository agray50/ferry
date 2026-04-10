package registry

import (
	"fmt"
	"sort"

	"github.com/anthropics/ferry/internal/config"
)

// Language describes a language's full toolchain.
type Language struct {
	Name              string
	TreesitterParsers []string
	LSP               string
	AlternateLSPs     []string
	Formatters        []string
	Linters           []string
	Runtime           *Runtime
}

// Runtime describes a version manager and package manager for a language.
type Runtime struct {
	Manager        string   // pyenv | nvm | goenv | none
	DefaultVersion string   // "3.12" | "lts" | "stable"
	PackageManager string   // pip | npm | go get
	ExtraPackages  []string // additional packages to install at bundle time
}

var registry = map[string]Language{
	"lua": {
		Name:              "lua",
		TreesitterParsers: []string{"lua", "luadoc", "luap"},
		LSP:               "lua-language-server",
		Formatters:        []string{"stylua"},
		Runtime:           &Runtime{Manager: "none"},
	},
	"python": {
		Name:              "python",
		TreesitterParsers: []string{"python"},
		LSP:               "pyright",
		AlternateLSPs:     []string{"pylsp"},
		Formatters:        []string{"black", "isort"},
		Linters:           []string{"flake8", "mypy"},
		Runtime:           &Runtime{Manager: "pyenv", DefaultVersion: "3.12", PackageManager: "pip"},
	},
	"typescript": {
		Name:              "typescript",
		TreesitterParsers: []string{"typescript", "javascript", "tsx", "jsdoc"},
		LSP:               "typescript-language-server",
		Formatters:        []string{"prettier"},
		Linters:           []string{"eslint_d"},
		Runtime:           &Runtime{Manager: "nvm", DefaultVersion: "lts", PackageManager: "npm"},
	},
	"javascript": {
		Name:              "javascript",
		TreesitterParsers: []string{"javascript", "jsdoc"},
		LSP:               "typescript-language-server",
		Formatters:        []string{"prettier"},
		Linters:           []string{"eslint_d"},
		Runtime:           &Runtime{Manager: "nvm", DefaultVersion: "lts", PackageManager: "npm"},
	},
	"dockerfile": {
		Name:              "dockerfile",
		TreesitterParsers: []string{"dockerfile"},
		LSP:               "dockerfile-language-server",
		Formatters:        []string{"hadolint"},
		Linters:           []string{"hadolint"},
		Runtime:           &Runtime{Manager: "none"},
	},
	"go": {
		Name:              "go",
		TreesitterParsers: []string{"go", "gomod", "gosum", "gowork"},
		LSP:               "gopls",
		Formatters:        []string{"gofmt", "goimports"},
		Linters:           []string{"staticcheck"},
		Runtime:           &Runtime{Manager: "goenv", DefaultVersion: "stable", PackageManager: "go get"},
	},
	"rust": {
		Name:              "rust",
		TreesitterParsers: []string{"rust", "toml"},
		LSP:               "rust-analyzer",
		Formatters:        []string{"rustfmt"},
		Runtime:           &Runtime{Manager: "none"},
	},
	"bash": {
		Name:              "bash",
		TreesitterParsers: []string{"bash"},
		LSP:               "bash-language-server",
		Formatters:        []string{"shfmt"},
		Linters:           []string{"shellcheck"},
		Runtime:           &Runtime{Manager: "none"},
	},
	"sh": {
		Name:              "sh",
		TreesitterParsers: []string{"bash"},
		LSP:               "bash-language-server",
		Formatters:        []string{"shfmt"},
		Linters:           []string{"shellcheck"},
		Runtime:           &Runtime{Manager: "none"},
	},
	"yaml": {
		Name:              "yaml",
		TreesitterParsers: []string{"yaml"},
		LSP:               "yaml-language-server",
		Formatters:        []string{"prettier"},
		Runtime:           &Runtime{Manager: "none"},
	},
	"json": {
		Name:              "json",
		TreesitterParsers: []string{"json", "jsonc"},
		LSP:               "json-language-server",
		Formatters:        []string{"prettier", "jq"},
		Runtime:           &Runtime{Manager: "none"},
	},
	"markdown": {
		Name:              "markdown",
		TreesitterParsers: []string{"markdown", "markdown_inline"},
		LSP:               "marksman",
		Formatters:        []string{"prettier"},
		Runtime:           &Runtime{Manager: "none"},
	},
}

// Get returns a Language by name.
func Get(name string) (Language, error) {
	l, ok := registry[name]
	if !ok {
		return Language{}, fmt.Errorf("unknown language %q", name)
	}
	return l, nil
}

// All returns all registered languages.
func All() []Language {
	out := make([]Language, 0, len(registry))
	for _, l := range registry {
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Names returns all registered language names, sorted.
func Names() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// IsValid returns true if the language name is registered.
func IsValid(name string) bool {
	_, ok := registry[name]
	return ok
}

// IsValidLSP returns true if the given LSP is valid for the language.
func IsValidLSP(language, lsp string) bool {
	l, ok := registry[language]
	if !ok {
		return false
	}
	if l.LSP == lsp {
		return true
	}
	for _, alt := range l.AlternateLSPs {
		if alt == lsp {
			return true
		}
	}
	return false
}

// ResolveLanguages takes a list of language names and applies overrides.
func ResolveLanguages(enabled []string, overrides map[string]config.LanguageOverride) ([]Language, error) {
	var out []Language
	for _, name := range enabled {
		l, err := Get(name)
		if err != nil {
			return nil, err
		}

		if ov, ok := overrides[name]; ok {
			if ov.LSP != "" {
				if !IsValidLSP(name, ov.LSP) {
					return nil, fmt.Errorf("invalid LSP %q for language %q", ov.LSP, name)
				}
				l.LSP = ov.LSP
			}
			if l.Runtime != nil && (ov.RuntimeVersion != "" || len(ov.ExtraPackages) > 0) {
				rt := *l.Runtime
				if ov.RuntimeVersion != "" {
					rt.DefaultVersion = ov.RuntimeVersion
				}
				if len(ov.ExtraPackages) > 0 {
					rt.ExtraPackages = append(rt.ExtraPackages, ov.ExtraPackages...)
				}
				l.Runtime = &rt
			}
		}
		out = append(out, l)
	}
	return out, nil
}
