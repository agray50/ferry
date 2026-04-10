package registry

import (
	"fmt"
	"sort"

	"github.com/anthropics/ferry/internal/config"
)

// BuildStep is a single instruction in a Docker build sequence.
// Kind "run" emits a RUN instruction; kind "env" emits an ENV instruction.
type BuildStep struct {
	Kind  string // "run" | "env"
	Value string // shell command for "run"; "KEY=VALUE" for "env"
}

// ContainerPath maps a path inside a Linux Docker container to a target install path.
// {VERSION} is substituted with the resolved runtime version at bundle time.
// {ARCH} is substituted with the build track arch (x86_64 or arm64).
type ContainerPath struct {
	Container   string // e.g. /root/.pyenv/versions/{VERSION}/
	InstallPath string // e.g. ~/.ferry/runtimes/python-{VERSION}/
}

// MacOSDownload describes a direct binary download for darwin build tracks.
// {VERSION} and {ARCH} are substituted at bundle time.
type MacOSDownload struct {
	URL         string // direct download URL template
	Arch        string // "x86_64" | "arm64" | "universal"
	ArchiveRoot string // subdirectory within the downloaded archive; "" = root
	InstallPath string // e.g. ~/.ferry/runtimes/go-{VERSION}/
}

// AlternateLSP is an alternative language server for a language.
// Its BuildSteps replace the default LSPBuildSteps when selected.
type AlternateLSP struct {
	Name       string
	BuildSteps []BuildStep
}

// Runtime describes how a language runtime is installed, extracted, and activated.
// Version managers (pyenv, nvm, etc.) are Docker build tools only — they are
// never shipped to the target. ContainerPaths point at the versioned output
// of the version manager, not the manager itself.
type Runtime struct {
	Manager           string   // pyenv | nvm | rustup | rbenv | sdkman | direct | system
	DefaultVersion    string   // used when no version override is specified
	AvailableVersions []string // shown in TUI version picker
	PackageManager    string   // pip | npm | cargo | gem | go
	ExtraPackages     []string // installed via PackageManager after runtime

	// Linux Docker build sequence: alternating RUN and ENV instructions.
	// {VERSION} and {ARCH} are substituted at bundle time.
	BuildSteps []BuildStep

	// Paths to extract from the Linux container after build.
	// These point at the versioned runtime directory, not the version manager.
	// {VERSION} and {ARCH} substituted at bundle time.
	ContainerPaths []ContainerPath

	// Direct download specs for darwin/macOS build tracks (no Docker).
	// {VERSION} and {ARCH} substituted at bundle time.
	MacOSDownloads []MacOSDownload

	// Lines appended to ~/.zshrc on the target after installation.
	// Point directly at ~/.ferry/runtimes/{lang}-{version}/bin.
	// {VERSION} substituted at install-script generation time.
	ShellInit []string

	// Docker steps to install the default LSP inside the container.
	// Executed after BuildSteps so the runtime is available.
	LSPBuildSteps []BuildStep
}

// Language describes a language's full toolchain configuration.
type Language struct {
	Name              string
	TreesitterParsers []string
	LSP               string
	AlternateLSPs     []AlternateLSP
	Formatters        []string
	Linters           []string
	Runtime           *Runtime // nil = no runtime needed (yaml, json, markdown, dockerfile)
	LSPOnlyRuntime    *Runtime // lighter path: installs LSP binary only, no full toolchain
	ApproxSizeMB      int      // estimated compressed size of full runtime, for TUI
	ApproxLSPOnlyMB   int      // estimated compressed size of LSP-only tier
	MacOSSupported    bool     // false = darwin tracks produce no component for this language
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
		AlternateLSPs:     []AlternateLSP{{Name: "pylsp"}},
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
		if alt.Name == lsp {
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

