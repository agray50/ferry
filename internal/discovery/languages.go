package discovery

import (
	"os/exec"
	"strings"

	"github.com/anthropics/ferry/internal/registry"
)

// LanguageDiscovery holds results of language toolchain scanning.
type LanguageDiscovery struct {
	Name     string
	Detected bool
	LSPFound string
	Tools    []ToolInfo
}

// ToolInfo holds metadata about a single discovered tool binary.
type ToolInfo struct {
	Name    string
	Path    string
	Version string
	Static  bool
	Bundled string // "direct" | "via-runtime" | "system" | "cannot"
}

// DiscoverLanguages checks PATH for known LSPs, linters, formatters.
func DiscoverLanguages() []LanguageDiscovery {
	var results []LanguageDiscovery
	for _, lang := range registry.All() {
		d := LanguageDiscovery{Name: lang.Name}

		// check LSP
		if path, err := exec.LookPath(lang.LSP); err == nil {
			d.Detected = true
			d.LSPFound = lang.LSP
			d.Tools = append(d.Tools, probeTool(lang.LSP, path, "direct"))
		}
		// check formatters and linters — allocate a new slice to avoid mutating
		// lang.Formatters' backing array if it has spare capacity.
		allTools := make([]string, 0, len(lang.Formatters)+len(lang.Linters))
		allTools = append(allTools, lang.Formatters...)
		allTools = append(allTools, lang.Linters...)
		for _, tool := range allTools {
			if path, err := exec.LookPath(tool); err == nil {
				d.Tools = append(d.Tools, probeTool(tool, path, "system"))
				if !d.Detected {
					d.Detected = true
				}
			}
		}

		if d.Detected {
			results = append(results, d)
		}
	}
	return results
}

func probeTool(name, path, bundled string) ToolInfo {
	info := ToolInfo{
		Name:    name,
		Path:    path,
		Bundled: bundled,
	}

	// detect static linking
	if out, err := exec.Command("file", path).Output(); err == nil {
		lower := strings.ToLower(string(out))
		info.Static = strings.Contains(lower, "statically linked")
	}

	// try --version
	if out, err := exec.Command(path, "--version").Output(); err == nil {
		lines := strings.SplitN(string(out), "\n", 2)
		if len(lines) > 0 {
			info.Version = strings.TrimSpace(lines[0])
		}
	}

	return info
}
