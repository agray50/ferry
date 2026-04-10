package bundle

import (
	"fmt"
	"strings"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/registry"
)

// goVersion is the pinned Go toolchain version used when building Go projects.
const goVersion = "1.22.5"

// GenerateDockerfile generates a Dockerfile string for a given build track.
func GenerateDockerfile(track BuildTrack, lock *config.LockFile, langs []registry.Language) (string, error) {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("FROM %s\n", track.DockerBase))

	if track.Libc == "glibc" {
		b.WriteString("ENV DEBIAN_FRONTEND=noninteractive\n\n")
		b.WriteString("# Base deps\n")
		b.WriteString("RUN apt-get update && apt-get install -y \\\n")
		b.WriteString("    curl git unzip tar gzip xz-utils build-essential wget \\\n")
		b.WriteString("    && rm -rf /var/lib/apt/lists/*\n\n")
	} else {
		b.WriteString("\n# Base deps\n")
		b.WriteString("RUN apk add --no-cache \\\n")
		b.WriteString("    curl git unzip tar gzip xz build-base wget bash\n\n")
	}

	// Neovim — download arch name matches GitHub release naming
	nvimVersion := lock.Nvim.Version
	if nvimVersion == "" {
		nvimVersion = "0.10.0"
	}
	nvimArch := "x86_64"
	if track.Arch == "arm64" {
		nvimArch = "arm64"
	}
	b.WriteString("# neovim\n")
	b.WriteString(fmt.Sprintf("RUN curl -LO https://github.com/neovim/neovim/releases/download/v%s/nvim-linux-%s.tar.gz \\\n",
		nvimVersion, nvimArch))
	b.WriteString(fmt.Sprintf("    && tar -xzf nvim-linux-%s.tar.gz \\\n", nvimArch))
	b.WriteString(fmt.Sprintf("    && mv nvim-linux-%s /opt/nvim \\\n", nvimArch))
	b.WriteString("    && ln -s /opt/nvim/bin/nvim /usr/local/bin/nvim \\\n")
	b.WriteString(fmt.Sprintf("    && rm nvim-linux-%s.tar.gz\n\n", nvimArch))

	// Nvim config
	b.WriteString("# nvim config\n")
	b.WriteString("COPY nvim-config /root/.config/nvim\n\n")

	// Lazy.nvim bootstrap
	b.WriteString("# bootstrap lazy.nvim plugins\n")
	b.WriteString("RUN nvim --headless \"+Lazy! sync\" +qa || true\n\n")

	// Treesitter parsers
	var parsers []string
	parserSet := map[string]bool{}
	for _, lang := range langs {
		for _, p := range lang.TreesitterParsers {
			if !parserSet[p] {
				parserSet[p] = true
				parsers = append(parsers, p)
			}
		}
	}
	if len(parsers) > 0 {
		b.WriteString("# treesitter parsers\n")
		b.WriteString(fmt.Sprintf("RUN nvim --headless \"+TSInstall %s\" +qa || true\n\n",
			strings.Join(parsers, " ")))
	}

	// Language runtimes — one block per unique manager
	pyenvDone, nvmDone, goDone := false, false, false
	for _, lang := range langs {
		if lang.Runtime == nil || lang.Runtime.Manager == "none" {
			continue
		}
		version := lang.Runtime.DefaultVersion
		if ov, ok := lock.Languages.Overrides[lang.Name]; ok && ov.RuntimeVersion != "" {
			version = ov.RuntimeVersion
		}

		switch lang.Runtime.Manager {
		case "pyenv":
			if pyenvDone {
				continue
			}
			pyenvDone = true
			extraPkgs := ""
			if len(lang.Runtime.ExtraPackages) > 0 {
				extraPkgs = " " + strings.Join(lang.Runtime.ExtraPackages, " ")
			}
			b.WriteString("# python via pyenv\n")
			b.WriteString("RUN curl https://pyenv.run | bash\n")
			b.WriteString("ENV PATH=\"/root/.pyenv/bin:/root/.pyenv/shims:$PATH\"\n")
			b.WriteString(fmt.Sprintf("RUN pyenv install %s && pyenv global %s\n", version, version))
			b.WriteString(fmt.Sprintf("RUN pip install black isort flake8 mypy%s\n\n", extraPkgs))

		case "nvm":
			if nvmDone {
				continue
			}
			nvmDone = true
			extraPkgs := ""
			if len(lang.Runtime.ExtraPackages) > 0 {
				extraPkgs = " " + strings.Join(lang.Runtime.ExtraPackages, " ")
			}
			b.WriteString("# node via nvm\n")
			b.WriteString("RUN curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash\n")
			b.WriteString("ENV NVM_DIR=/root/.nvm\n")
			b.WriteString(fmt.Sprintf("RUN . $NVM_DIR/nvm.sh && nvm install %s && nvm use %s\n", version, version))
			b.WriteString(fmt.Sprintf("RUN . $NVM_DIR/nvm.sh && npm install -g typescript prettier eslint_d typescript-language-server%s\n\n", extraPkgs))

		case "goenv":
			if goDone {
				continue
			}
			goDone = true
			// Select correct Go download arch
			goArch := "amd64"
			if track.Arch == "arm64" {
				goArch = "arm64"
			}
			b.WriteString("# go\n")
			b.WriteString(fmt.Sprintf("RUN curl -LO https://go.dev/dl/go%s.linux-%s.tar.gz \\\n", goVersion, goArch))
			b.WriteString(fmt.Sprintf("    && tar -C /usr/local -xzf go%s.linux-%s.tar.gz \\\n", goVersion, goArch))
			b.WriteString(fmt.Sprintf("    && rm go%s.linux-%s.tar.gz\n", goVersion, goArch))
			b.WriteString("ENV PATH=\"/usr/local/go/bin:$PATH\"\n")
			b.WriteString("RUN go install golang.org/x/tools/gopls@latest\n\n")
		}
	}

	// Static CLI tools — use pinned versions from CLIDownloads
	for name := range lock.CLI {
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
		b.WriteString(fmt.Sprintf("RUN curl -fsSL %s -o /tmp/%s.download && "+
			"chmod +x /tmp/%s.download && "+
			"mv /tmp/%s.download /usr/local/bin/%s || true\n\n",
			url, name, name, name, name))
	}

	// Build marker
	b.WriteString("RUN echo \"ferry-build-complete\"\n")

	return b.String(), nil
}
