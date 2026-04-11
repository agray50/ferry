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

// registry is the package-level language map, defined in languages.go.
var registry = languageRegistry

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

// ResolveLanguages resolves a list of LanguageConfig entries against the registry.
// TODO: rewrite in Phase 3 to fully handle Tier, Formatters, Linters fields.
func ResolveLanguages(langs []config.LanguageConfig) ([]Language, error) {
	var out []Language
	for _, lc := range langs {
		l, err := Get(lc.Name)
		if err != nil {
			return nil, err
		}

		if lc.LSP != "" {
			if !IsValidLSP(lc.Name, lc.LSP) {
				return nil, fmt.Errorf("invalid LSP %q for language %q", lc.LSP, lc.Name)
			}
			l.LSP = lc.LSP
		}
		if l.Runtime != nil && (lc.RuntimeVersion != "" || len(lc.ExtraPackages) > 0) {
			rt := *l.Runtime
			if lc.RuntimeVersion != "" {
				rt.DefaultVersion = lc.RuntimeVersion
			}
			if len(lc.ExtraPackages) > 0 {
				rt.ExtraPackages = append(rt.ExtraPackages, lc.ExtraPackages...)
			}
			l.Runtime = &rt
		}
		out = append(out, l)
	}
	return out, nil
}

