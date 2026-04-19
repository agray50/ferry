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
	// Only emitted for the first language claiming this Manager; subsequent
	// languages sharing the same Manager have BuildSteps cleared by deduplication.
	BuildSteps []BuildStep

	// AddonBuildSteps are always emitted regardless of runtime deduplication.
	// Use these for tool installs (LSP, formatter, linter) that piggyback on a
	// shared runtime managed by another language (e.g. npm tools on a shared
	// nvm node install). {VERSION} and {ARCH} substituted at bundle time.
	AddonBuildSteps []BuildStep

	// Per-tool install steps keyed by formatter/linter name.
	// Dockerfile generator emits only the steps for the effective tool list.
	FormatterBuildSteps map[string][]BuildStep
	LinterBuildSteps    map[string][]BuildStep

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

// ResolvedLanguage is a language with its effective runtime and LSP selected
// based on the LanguageConfig from ferry.lock.
type ResolvedLanguage struct {
	Language           Language
	Runtime            *Runtime // nil if language has no runtime or tier is lsp-only with nil LSPOnlyRuntime
	EffectiveLSP       string
	EffectiveFormatters []string // from LanguageConfig override, or Language.Formatters default
	EffectiveLinters   []string // from LanguageConfig override, or Language.Linters default
	Config             config.LanguageConfig
}

// ResolveFromProfile resolves a list of LanguageConfigs from a profile into
// ResolvedLanguages with the correct runtime tier and LSP applied.
// Returns error if any language name is unknown or LSP override is invalid.
// tools may be nil; when provided, custom LSPs defined in the tools file are
// accepted as valid overrides.
func ResolveFromProfile(langs []config.LanguageConfig, tools *config.ToolsFile) ([]ResolvedLanguage, error) {
	seen := map[string]bool{} // manager → already have runtime component
	var out []ResolvedLanguage

	for _, lc := range langs {
		l, err := Get(lc.Name)
		if err != nil {
			return nil, err
		}

		// Validate LSP override
		lsp := lc.LSP
		if lsp == "" {
			lsp = l.LSP
		} else if !IsValidLSP(lc.Name, lsp) && !tools.HasLSP(lsp) {
			return nil, fmt.Errorf("invalid LSP %q for language %q", lsp, lc.Name)
		}

		// Select runtime tier
		var rt *Runtime
		switch lc.Tier {
		case "lsp-only":
			rt = l.LSPOnlyRuntime
		default: // "full" or empty
			rt = l.Runtime
		}

		// Deduplicate by runtime manager: if two languages share a manager (e.g.
		// javascript and typescript both use nvm), only the first gets BuildSteps
		// and ContainerPaths. Subsequent languages keep AddonBuildSteps so their
		// tool installs (LSP, formatter, linter) still run against the shared runtime.
		if rt != nil && rt.Manager != "" && rt.Manager != "system" && rt.Manager != "none" {
			if seen[rt.Manager] {
				dedup := *rt
				dedup.BuildSteps = nil
				dedup.ContainerPaths = nil
				dedup.LSPBuildSteps = nil
				rt = &dedup
			} else {
				seen[rt.Manager] = true
			}
		}

		// Apply version and extra-package overrides
		if rt != nil && (lc.RuntimeVersion != "" || len(lc.ExtraPackages) > 0) {
			cp := *rt
			if lc.RuntimeVersion != "" {
				cp.DefaultVersion = lc.RuntimeVersion
			}
			if len(lc.ExtraPackages) > 0 {
				cp.ExtraPackages = append(append([]string{}, rt.ExtraPackages...), lc.ExtraPackages...)
			}
			rt = &cp
		}

		// Resolve effective formatters and linters: use the lock file override if
		// set, otherwise fall back to the registry defaults.
		effectiveFmt := l.Formatters
		if len(lc.Formatters) > 0 {
			effectiveFmt = lc.Formatters
		}
		effectiveLint := l.Linters
		if len(lc.Linters) > 0 {
			effectiveLint = lc.Linters
		}

		out = append(out, ResolvedLanguage{
			Language:            l,
			Runtime:             rt,
			EffectiveLSP:        lsp,
			EffectiveFormatters: effectiveFmt,
			EffectiveLinters:    effectiveLint,
			Config:              lc,
		})
	}
	return out, nil
}


