package bundle

// DownloadSpec describes how to download a tool binary.
type DownloadSpec struct {
	URLTemplate string
	Version     string            // pinned release version used in URLTemplate
	Arches      map[string]string // ferry arch → download arch name
	InstallVia  string            // "npm" | "pip" | "go" | "" (direct download)
	Package     string            // npm/pip/go package name
}

// LSPDownloads maps LSP server names to their download specifications.
var LSPDownloads = map[string]DownloadSpec{
	"lua-language-server": {
		URLTemplate: "https://github.com/LuaLS/lua-language-server/releases/download/{VERSION}/lua-language-server-{VERSION}-linux-{ARCH}.tar.gz",
		Version:     "3.9.1",
		Arches:      map[string]string{"x86_64": "x64", "arm64": "arm64"},
	},
	"pyright": {
		InstallVia: "npm",
		Package:    "pyright",
	},
	"typescript-language-server": {
		InstallVia: "npm",
		Package:    "typescript-language-server typescript",
	},
	"gopls": {
		InstallVia: "go",
		Package:    "golang.org/x/tools/gopls@latest",
	},
	"rust-analyzer": {
		URLTemplate: "https://github.com/rust-lang/rust-analyzer/releases/download/{VERSION}/rust-analyzer-{ARCH}-unknown-linux-musl.gz",
		Version:     "2024-01-29",
		Arches:      map[string]string{"x86_64": "x86_64", "arm64": "aarch64"},
	},
	"bash-language-server": {
		InstallVia: "npm",
		Package:    "bash-language-server",
	},
	"yaml-language-server": {
		InstallVia: "npm",
		Package:    "yaml-language-server",
	},
	"json-language-server": {
		InstallVia: "npm",
		Package:    "vscode-langservers-extracted",
	},
	"dockerfile-language-server": {
		InstallVia: "npm",
		Package:    "dockerfile-language-server-nodejs",
	},
	"marksman": {
		URLTemplate: "https://github.com/artempyanykh/marksman/releases/download/{VERSION}/marksman-linux-{ARCH}",
		Version:     "2023-12-09",
		Arches:      map[string]string{"x86_64": "x64", "arm64": "arm64"},
	},
}

// CLIDownloads maps CLI tool names to their download specifications.
var CLIDownloads = map[string]DownloadSpec{
	"rg": {
		URLTemplate: "https://github.com/BurntSushi/ripgrep/releases/download/{VERSION}/ripgrep-{VERSION}-{ARCH}-unknown-linux-musl.tar.gz",
		Version:     "14.1.1",
		Arches:      map[string]string{"x86_64": "x86_64", "arm64": "aarch64"},
	},
	"fzf": {
		URLTemplate: "https://github.com/junegunn/fzf/releases/download/v{VERSION}/fzf-{VERSION}-linux_{ARCH}.tar.gz",
		Version:     "0.54.3",
		Arches:      map[string]string{"x86_64": "amd64", "arm64": "arm64"},
	},
	"zoxide": {
		URLTemplate: "https://github.com/ajeetdsouza/zoxide/releases/download/v{VERSION}/zoxide-{VERSION}-{ARCH}-unknown-linux-musl.tar.gz",
		Version:     "0.9.4",
		Arches:      map[string]string{"x86_64": "x86_64", "arm64": "aarch64"},
	},
	"jq": {
		URLTemplate: "https://github.com/jqlang/jq/releases/download/jq-{VERSION}/jq-linux-{ARCH}",
		Version:     "1.7.1",
		Arches:      map[string]string{"x86_64": "amd64", "arm64": "arm64"},
	},
	"shfmt": {
		URLTemplate: "https://github.com/mvdan/sh/releases/download/v{VERSION}/shfmt_v{VERSION}_linux_{ARCH}",
		Version:     "3.8.0",
		Arches:      map[string]string{"x86_64": "amd64", "arm64": "arm64"},
	},
	"shellcheck": {
		URLTemplate: "https://github.com/koalaman/shellcheck/releases/download/v{VERSION}/shellcheck-v{VERSION}.linux.{ARCH}.tar.xz",
		Version:     "0.10.0",
		Arches:      map[string]string{"x86_64": "x86_64", "arm64": "aarch64"},
	},
	"stylua": {
		URLTemplate: "https://github.com/JohnnyMorganz/StyLua/releases/download/v{VERSION}/stylua-linux-{ARCH}.zip",
		Version:     "0.20.0",
		Arches:      map[string]string{"x86_64": "x86_64", "arm64": "aarch64"},
	},
	"hadolint": {
		URLTemplate: "https://github.com/hadolint/hadolint/releases/download/v{VERSION}/hadolint-Linux-{ARCH}",
		Version:     "2.12.0",
		Arches:      map[string]string{"x86_64": "x86_64", "arm64": "arm64"},
	},
}
