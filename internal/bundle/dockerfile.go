package bundle

import (
	"fmt"
	"strings"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/registry"
)

// archDockerMap translates ferry arch names to Docker/Go download arch names.
var archDockerMap = map[string]string{
	"x86_64": "amd64",
	"arm64":  "arm64",
}

// GenerateDockerfile generates a Dockerfile for a linux/docker build track.
// Returns error if called with a non-docker (darwin) track.
func GenerateDockerfile(track BuildTrack, lock *config.LockFile, profile string, langs []registry.ResolvedLanguage, tools *config.ToolsFile) (string, error) {
	if track.BuildMethod != "docker" {
		return "", fmt.Errorf("GenerateDockerfile: track %s/%s uses build method %q, not docker", track.OS, track.Arch, track.BuildMethod)
	}

	dockerArch, ok := archDockerMap[track.Arch]
	if !ok {
		dockerArch = track.Arch
	}

	// Normalize nil tools to an empty ToolsFile so callers don't need to nil-check.
	if tools == nil {
		tools = &config.ToolsFile{
			Formatters: make(map[string]config.CustomTool),
			Linters:    make(map[string]config.CustomTool),
			LSPs:       make(map[string]config.CustomTool),
		}
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf("FROM %s\n", track.DockerBase))
	b.WriteString("ENV DEBIAN_FRONTEND=noninteractive\n\n")

	b.WriteString("# Base deps\n")
	b.WriteString("RUN apt-get update && apt-get install -y \\\n")
	b.WriteString("    curl git unzip tar gzip xz-utils build-essential wget zip \\\n")
	b.WriteString("    libssl-dev libffi-dev zlib1g-dev libreadline-dev libbz2-dev \\\n")
	b.WriteString("    libsqlite3-dev liblzma-dev libncurses-dev tk-dev \\\n")
	b.WriteString("    && rm -rf /var/lib/apt/lists/*\n\n")

	prof := lock.Profiles[profile]

	if prof.NvimEnabled() {
		// Neovim — release asset naming: nvim-linux-{arch}.tar.gz (v0.10.4+)
		nvimVersion := lock.Nvim.Version
		if nvimVersion == "" {
			nvimVersion = "0.10.4"
		}
		nvimArch := "x86_64"
		if track.Arch == "arm64" {
			nvimArch = "arm64"
		}
		tarball := fmt.Sprintf("nvim-linux-%s.tar.gz", nvimArch)
		b.WriteString("# neovim\n")
		b.WriteString(fmt.Sprintf("RUN curl -fsSL https://github.com/neovim/neovim/releases/download/v%s/%s -o /tmp/%s \\\n", nvimVersion, tarball, tarball))
		b.WriteString(fmt.Sprintf("    && tar -xzf /tmp/%s -C /opt \\\n", tarball))
		b.WriteString(fmt.Sprintf("    && mv /opt/nvim-linux-%s /opt/nvim \\\n", nvimArch))
		b.WriteString("    && ln -s /opt/nvim/bin/nvim /usr/local/bin/nvim \\\n")
		b.WriteString(fmt.Sprintf("    && rm /tmp/%s\n\n", tarball))

		b.WriteString("# nvim config\n")
		b.WriteString("COPY nvim-config /root/.config/nvim\n\n")

		b.WriteString("# bootstrap lazy.nvim plugins\n")
		b.WriteString("RUN nvim --headless \"+Lazy! sync\" +qa || true\n\n")

		// Treesitter parsers (union across all languages)
		var parsers []string
		seen := map[string]bool{}
		for _, rl := range langs {
			for _, p := range rl.Language.TreesitterParsers {
				if !seen[p] {
					seen[p] = true
					parsers = append(parsers, p)
				}
			}
		}
		if len(parsers) > 0 {
			b.WriteString("# treesitter parsers\n")
			b.WriteString(fmt.Sprintf("RUN nvim --headless \"+TSInstall %s\" +qa || true\n\n", strings.Join(parsers, " ")))
		}
	}

	// Language runtimes — data-driven from registry BuildSteps.
	// Each ResolvedLanguage carries its Runtime (already deduplicated by ResolveFromProfile).
	// BuildSteps are nil for languages that share a manager with an earlier language.
	// AddonBuildSteps always run and install tools into the shared runtime.
	for _, rl := range langs {
		rt := rl.Runtime
		if rt == nil {
			continue
		}
		version := rt.DefaultVersion
		wrote := false

		if len(rt.BuildSteps) > 0 {
			b.WriteString(fmt.Sprintf("# %s runtime (%s)\n", rl.Language.Name, version))
			emitBuildSteps(&b, rt.BuildSteps, version, dockerArch)
			if len(rt.LSPBuildSteps) > 0 {
				b.WriteString(fmt.Sprintf("# %s LSP (%s)\n", rl.Language.Name, rl.EffectiveLSP))
				emitBuildSteps(&b, rt.LSPBuildSteps, version, dockerArch)
			} else if rl.EffectiveLSP != "" {
				if ct, ok := tools.LSPs[rl.EffectiveLSP]; ok {
					b.WriteString(fmt.Sprintf("# %s LSP: %s (custom)\n", rl.Language.Name, rl.EffectiveLSP))
					b.WriteString(fmt.Sprintf("RUN %s\n", ct.Build))
				}
			}
			wrote = true
		}

		// Build sets of default formatters/linters for this language so we can
		// distinguish user-overridden tools (which must be resolvable) from
		// registry defaults that are simply bundled with the runtime (e.g. gofmt,
		// rustfmt, zig fmt) and therefore have no separate install steps.
		defaultFmts := make(map[string]bool, len(rl.Language.Formatters))
		for _, f := range rl.Language.Formatters {
			defaultFmts[f] = true
		}
		defaultLints := make(map[string]bool, len(rl.Language.Linters))
		for _, l := range rl.Language.Linters {
			defaultLints[l] = true
		}

		// Formatter installs — only the tools in the effective formatter list.
		// Rules:
		//   key present, empty slice  → intentional piggyback, skip
		//   key absent, default list  → bundled with runtime (e.g. gofmt), skip
		//   key absent, user override → check custom tools file; error if not found
		for _, name := range rl.EffectiveFormatters {
			steps, keyExists := rt.FormatterBuildSteps[name]
			if keyExists && len(steps) == 0 {
				continue // intentional piggyback
			}
			if !keyExists {
				if ct, ok := tools.Formatters[name]; ok {
					b.WriteString(fmt.Sprintf("# %s formatter: %s (custom)\n", rl.Language.Name, name))
					b.WriteString(fmt.Sprintf("RUN %s\n", ct.Build))
					wrote = true
				} else if !defaultFmts[name] {
					return "", fmt.Errorf("formatter %q for language %q is not in the built-in registry; "+
						"add it to ferry-tools.yaml under formatters:", name, rl.Language.Name)
				}
				// else: default formatter bundled with runtime — skip
				continue
			}
			b.WriteString(fmt.Sprintf("# %s formatter: %s\n", rl.Language.Name, name))
			emitBuildSteps(&b, steps, version, dockerArch)
			wrote = true
		}

		// Linter installs — same rules as formatters above.
		for _, name := range rl.EffectiveLinters {
			steps, keyExists := rt.LinterBuildSteps[name]
			if keyExists && len(steps) == 0 {
				continue // intentional piggyback
			}
			if !keyExists {
				if ct, ok := tools.Linters[name]; ok {
					b.WriteString(fmt.Sprintf("# %s linter: %s (custom)\n", rl.Language.Name, name))
					b.WriteString(fmt.Sprintf("RUN %s\n", ct.Build))
					wrote = true
				} else if !defaultLints[name] {
					return "", fmt.Errorf("linter %q for language %q is not in the built-in registry; "+
						"add it to ferry-tools.yaml under linters:", name, rl.Language.Name)
				}
				// else: default linter bundled with runtime — skip
				continue
			}
			b.WriteString(fmt.Sprintf("# %s linter: %s\n", rl.Language.Name, name))
			emitBuildSteps(&b, steps, version, dockerArch)
			wrote = true
		}

		// AddonBuildSteps: tools installed into a shared runtime managed by another
		// language. Always emitted, even when BuildSteps were cleared by deduplication.
		if len(rt.AddonBuildSteps) > 0 {
			b.WriteString(fmt.Sprintf("# %s tools\n", rl.Language.Name))
			emitBuildSteps(&b, rt.AddonBuildSteps, version, dockerArch)
			wrote = true
		}

		if wrote {
			b.WriteString("\n")
		}
	}

	// Static CLI tools
	for _, name := range flattenCLI(lock, profile) {
		spec, ok := CLIDownloads[name]
		if !ok || spec.URLTemplate == "" {
			continue
		}
		dlArch := track.Arch
		if a, ok := spec.Arches[track.Arch]; ok {
			dlArch = a
		}
		url := strings.ReplaceAll(spec.URLTemplate, "{ARCH}", dlArch)
		url = strings.ReplaceAll(url, "{VERSION}", spec.Version)
		b.WriteString(fmt.Sprintf("# %s\n", name))
		if spec.ArchiveInnerPath == "" {
			// direct binary download — no || true: a download failure should fail the build
			b.WriteString(fmt.Sprintf("RUN curl -fsSL %s -o /usr/local/bin/%s && chmod +x /usr/local/bin/%s\n\n",
				url, name, name))
		} else {
			// archive: extract, find the binary inside, install
			innerPath := strings.ReplaceAll(spec.ArchiveInnerPath, "{ARCH}", dlArch)
			innerPath = strings.ReplaceAll(innerPath, "{VERSION}", spec.Version)
			tmpArch := fmt.Sprintf("/tmp/%s-archive", name)
			tmpDir := fmt.Sprintf("/tmp/%s-extract", name)
			var extractCmd string
			switch {
			case strings.HasSuffix(url, ".tar.gz") || strings.HasSuffix(url, ".tgz"):
				extractCmd = fmt.Sprintf("tar -xzf %s -C %s", tmpArch, tmpDir)
			case strings.HasSuffix(url, ".tar.xz"):
				extractCmd = fmt.Sprintf("tar -xJf %s -C %s", tmpArch, tmpDir)
			case strings.HasSuffix(url, ".zip"):
				extractCmd = fmt.Sprintf("unzip -q %s -d %s", tmpArch, tmpDir)
			default:
				extractCmd = fmt.Sprintf("tar -xf %s -C %s", tmpArch, tmpDir)
			}
			b.WriteString(fmt.Sprintf(
				"RUN curl -fsSL %s -o %s && mkdir -p %s && %s && mv %s/%s /usr/local/bin/%s && chmod +x /usr/local/bin/%s && rm -rf %s %s\n\n",
				url, tmpArch, tmpDir, extractCmd, tmpDir, innerPath, name, name, tmpArch, tmpDir))
		}
	}

	b.WriteString("RUN echo \"ferry-build-complete\"\n")
	return b.String(), nil
}

// emitBuildSteps writes a slice of BuildSteps to the Dockerfile builder,
// substituting {VERSION}, {ARCH}, and {ARCH_NATIVE} in each step value.
func emitBuildSteps(b *strings.Builder, steps []registry.BuildStep, version, arch string) {
	for _, step := range steps {
		val := substituteVars(step.Value, version, arch)
		switch step.Kind {
		case "run":
			b.WriteString(fmt.Sprintf("RUN %s\n", val))
		case "env":
			b.WriteString(fmt.Sprintf("ENV %s\n", val))
		}
	}
}

// substituteVars replaces {VERSION}, {ARCH}, and {ARCH_NATIVE} in a template string.
// {ARCH}        → Docker/Go arch name:  amd64 | arm64
// {ARCH_NATIVE} → native machine arch:  x86_64 | arm64 (for tools that use uname-style names)
func substituteVars(s, version, arch string) string {
	nativeArch := arch
	if arch == "amd64" {
		nativeArch = "x86_64"
	}
	s = strings.ReplaceAll(s, "{VERSION}", version)
	s = strings.ReplaceAll(s, "{ARCH_NATIVE}", nativeArch)
	s = strings.ReplaceAll(s, "{ARCH}", arch)
	return s
}

// flattenCLI returns the CLI tools configured for the given profile.
func flattenCLI(lock *config.LockFile, profile string) []string {
	prof, ok := lock.Profiles[profile]
	if !ok {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, name := range prof.CLI {
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}
