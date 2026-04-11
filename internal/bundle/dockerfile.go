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
func GenerateDockerfile(track BuildTrack, lock *config.LockFile, langs []registry.ResolvedLanguage) (string, error) {
	if track.BuildMethod != "docker" {
		return "", fmt.Errorf("GenerateDockerfile: track %s/%s uses build method %q, not docker", track.OS, track.Arch, track.BuildMethod)
	}

	dockerArch, ok := archDockerMap[track.Arch]
	if !ok {
		dockerArch = track.Arch
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf("FROM %s\n", track.DockerBase))
	b.WriteString("ENV DEBIAN_FRONTEND=noninteractive\n\n")

	b.WriteString("# Base deps\n")
	b.WriteString("RUN apt-get update && apt-get install -y \\\n")
	b.WriteString("    curl git unzip tar gzip xz-utils build-essential wget zip \\\n")
	b.WriteString("    && rm -rf /var/lib/apt/lists/*\n\n")

	// Neovim
	nvimVersion := lock.Nvim.Version
	if nvimVersion == "" {
		nvimVersion = "0.10.0"
	}
	nvimArch := "x86_64"
	if track.Arch == "arm64" {
		nvimArch = "arm64"
	}
	b.WriteString("# neovim\n")
	b.WriteString(fmt.Sprintf("RUN curl -LO https://github.com/neovim/neovim/releases/download/v%s/nvim-linux-%s.tar.gz \\\n", nvimVersion, nvimArch))
	b.WriteString(fmt.Sprintf("    && tar -xzf nvim-linux-%s.tar.gz \\\n", nvimArch))
	b.WriteString(fmt.Sprintf("    && mv nvim-linux-%s /opt/nvim \\\n", nvimArch))
	b.WriteString("    && ln -s /opt/nvim/bin/nvim /usr/local/bin/nvim \\\n")
	b.WriteString(fmt.Sprintf("    && rm nvim-linux-%s.tar.gz\n\n", nvimArch))

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

	// Language runtimes — data-driven from registry BuildSteps.
	// Each ResolvedLanguage carries its Runtime (already deduplicated by ResolveFromProfile).
	for _, rl := range langs {
		rt := rl.Runtime
		if rt == nil || len(rt.BuildSteps) == 0 {
			continue
		}
		version := rt.DefaultVersion
		b.WriteString(fmt.Sprintf("# %s runtime (%s)\n", rl.Language.Name, version))
		for _, step := range rt.BuildSteps {
			val := substituteVars(step.Value, version, dockerArch)
			switch step.Kind {
			case "run":
				b.WriteString(fmt.Sprintf("RUN %s\n", val))
			case "env":
				b.WriteString(fmt.Sprintf("ENV %s\n", val))
			}
		}
		// LSP install steps (after runtime is ready)
		if len(rt.LSPBuildSteps) > 0 {
			b.WriteString(fmt.Sprintf("# %s LSP (%s)\n", rl.Language.Name, rl.EffectiveLSP))
			for _, step := range rt.LSPBuildSteps {
				val := substituteVars(step.Value, version, dockerArch)
				switch step.Kind {
				case "run":
					b.WriteString(fmt.Sprintf("RUN %s\n", val))
				case "env":
					b.WriteString(fmt.Sprintf("ENV %s\n", val))
				}
			}
		}
		b.WriteString("\n")
	}

	// Static CLI tools
	for _, name := range flattenCLI(lock) {
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
		b.WriteString(fmt.Sprintf("RUN curl -fsSL %s -o /tmp/%s.download && chmod +x /tmp/%s.download && mv /tmp/%s.download /usr/local/bin/%s || true\n\n",
			url, name, name, name, name))
	}

	b.WriteString("RUN echo \"ferry-build-complete\"\n")
	return b.String(), nil
}

// substituteVars replaces {VERSION} and {ARCH} in a template string.
func substituteVars(s, version, arch string) string {
	s = strings.ReplaceAll(s, "{VERSION}", version)
	s = strings.ReplaceAll(s, "{ARCH}", arch)
	return s
}

// flattenCLI returns the union of CLI tools across all profiles.
func flattenCLI(lock *config.LockFile) []string {
	seen := map[string]bool{}
	var out []string
	for _, prof := range lock.Profiles {
		for _, name := range prof.CLI {
			if !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
	}
	return out
}
