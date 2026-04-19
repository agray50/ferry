package bundle

// DownloadSpec describes how to download a tool binary.
type DownloadSpec struct {
	URLTemplate      string
	Version          string            // pinned release version used in URLTemplate
	Arches           map[string]string // ferry arch → download arch name
	InstallVia       string            // "npm" | "pip" | "go" | "" (direct download)
	Package          string            // npm/pip/go package name
	ArchiveInnerPath string            // path inside archive to the binary (empty = direct binary download)
}

// CLIDownloads maps CLI tool names to their download specifications.
var CLIDownloads = map[string]DownloadSpec{
	"rg": {
		URLTemplate:      "https://github.com/BurntSushi/ripgrep/releases/download/{VERSION}/ripgrep-{VERSION}-{ARCH}-unknown-linux-musl.tar.gz",
		Version:          "14.1.1",
		Arches:           map[string]string{"x86_64": "x86_64", "arm64": "aarch64"},
		ArchiveInnerPath: "ripgrep-{VERSION}-{ARCH}-unknown-linux-musl/rg",
	},
	"fzf": {
		URLTemplate:      "https://github.com/junegunn/fzf/releases/download/v{VERSION}/fzf-{VERSION}-linux_{ARCH}.tar.gz",
		Version:          "0.54.3",
		Arches:           map[string]string{"x86_64": "amd64", "arm64": "arm64"},
		ArchiveInnerPath: "fzf",
	},
	"zoxide": {
		URLTemplate:      "https://github.com/ajeetdsouza/zoxide/releases/download/v{VERSION}/zoxide-{VERSION}-{ARCH}-unknown-linux-musl.tar.gz",
		Version:          "0.9.4",
		Arches:           map[string]string{"x86_64": "x86_64", "arm64": "aarch64"},
		ArchiveInnerPath: "zoxide",
	},
	"jq": {
		URLTemplate: "https://github.com/jqlang/jq/releases/download/jq-{VERSION}/jq-linux-{ARCH}",
		Version:     "1.7.1",
		Arches:      map[string]string{"x86_64": "amd64", "arm64": "arm64"},
		// direct binary (no ArchiveInnerPath)
	},
	"shfmt": {
		URLTemplate: "https://github.com/mvdan/sh/releases/download/v{VERSION}/shfmt_v{VERSION}_linux_{ARCH}",
		Version:     "3.8.0",
		Arches:      map[string]string{"x86_64": "amd64", "arm64": "arm64"},
		// direct binary
	},
	"shellcheck": {
		URLTemplate:      "https://github.com/koalaman/shellcheck/releases/download/v{VERSION}/shellcheck-v{VERSION}.linux.{ARCH}.tar.xz",
		Version:          "0.10.0",
		Arches:           map[string]string{"x86_64": "x86_64", "arm64": "aarch64"},
		ArchiveInnerPath: "shellcheck-v{VERSION}/shellcheck",
	},
	"stylua": {
		URLTemplate:      "https://github.com/JohnnyMorganz/StyLua/releases/download/v{VERSION}/stylua-linux-{ARCH}.zip",
		Version:          "0.20.0",
		Arches:           map[string]string{"x86_64": "x86_64", "arm64": "aarch64"},
		ArchiveInnerPath: "stylua",
	},
	"hadolint": {
		URLTemplate: "https://github.com/hadolint/hadolint/releases/download/v{VERSION}/hadolint-Linux-{ARCH}",
		Version:     "2.12.0",
		Arches:      map[string]string{"x86_64": "x86_64", "arm64": "arm64"},
		// direct binary
	},
	"fd": {
		URLTemplate:      "https://github.com/sharkdp/fd/releases/download/v{VERSION}/fd-v{VERSION}-{ARCH}-unknown-linux-musl.tar.gz",
		Version:          "10.2.0",
		Arches:           map[string]string{"x86_64": "x86_64", "arm64": "aarch64"},
		ArchiveInnerPath: "fd-v{VERSION}-{ARCH}-unknown-linux-musl/fd",
	},
	"bat": {
		URLTemplate:      "https://github.com/sharkdp/bat/releases/download/v{VERSION}/bat-v{VERSION}-{ARCH}-unknown-linux-musl.tar.gz",
		Version:          "0.24.0",
		Arches:           map[string]string{"x86_64": "x86_64", "arm64": "aarch64"},
		ArchiveInnerPath: "bat-v{VERSION}-{ARCH}-unknown-linux-musl/bat",
	},
	"eza": {
		URLTemplate:      "https://github.com/eza-community/eza/releases/download/v{VERSION}/eza_{ARCH}-unknown-linux-musl.tar.gz",
		Version:          "0.20.14",
		Arches:           map[string]string{"x86_64": "x86_64", "arm64": "aarch64"},
		ArchiveInnerPath: "eza",
	},
	"yq": {
		URLTemplate: "https://github.com/mikefarah/yq/releases/download/v{VERSION}/yq_linux_{ARCH}",
		Version:     "4.44.3",
		Arches:      map[string]string{"x86_64": "amd64", "arm64": "arm64"},
		// direct binary
	},
	"marksman": {
		URLTemplate: "https://github.com/artempyanykh/marksman/releases/download/{VERSION}/marksman-linux-{ARCH}",
		Version:     "2023-12-09",
		Arches:      map[string]string{"x86_64": "x64", "arm64": "arm64"},
		// direct binary
	},
}

// CLIMacOSDownloads maps CLI tool names to their macOS download specifications.
var CLIMacOSDownloads = map[string]DownloadSpec{
	"rg": {
		URLTemplate:      "https://github.com/BurntSushi/ripgrep/releases/download/{VERSION}/ripgrep-{VERSION}-{ARCH}-apple-darwin.tar.gz",
		Version:          "14.1.1",
		Arches:           map[string]string{"x86_64": "x86_64", "arm64": "aarch64"},
		ArchiveInnerPath: "ripgrep-{VERSION}-{ARCH}-apple-darwin/rg",
	},
	"fzf": {
		URLTemplate:      "https://github.com/junegunn/fzf/releases/download/v{VERSION}/fzf-{VERSION}-darwin_{ARCH}.tar.gz",
		Version:          "0.54.3",
		Arches:           map[string]string{"x86_64": "amd64", "arm64": "arm64"},
		ArchiveInnerPath: "fzf",
	},
	"zoxide": {
		URLTemplate:      "https://github.com/ajeetdsouza/zoxide/releases/download/v{VERSION}/zoxide-{VERSION}-{ARCH}-apple-darwin.tar.gz",
		Version:          "0.9.4",
		Arches:           map[string]string{"x86_64": "x86_64", "arm64": "aarch64"},
		ArchiveInnerPath: "zoxide",
	},
	"jq": {
		URLTemplate: "https://github.com/jqlang/jq/releases/download/jq-{VERSION}/jq-macos-{ARCH}",
		Version:     "1.7.1",
		Arches:      map[string]string{"x86_64": "amd64", "arm64": "arm64"},
		// direct binary
	},
	"shfmt": {
		URLTemplate: "https://github.com/mvdan/sh/releases/download/v{VERSION}/shfmt_v{VERSION}_darwin_{ARCH}",
		Version:     "3.8.0",
		Arches:      map[string]string{"x86_64": "amd64", "arm64": "arm64"},
		// direct binary
	},
	"shellcheck": {
		URLTemplate:      "https://github.com/koalaman/shellcheck/releases/download/v{VERSION}/shellcheck-v{VERSION}.darwin.{ARCH}.tar.xz",
		Version:          "0.10.0",
		Arches:           map[string]string{"x86_64": "x86_64", "arm64": "aarch64"},
		ArchiveInnerPath: "shellcheck-v{VERSION}/shellcheck",
	},
	"stylua": {
		URLTemplate:      "https://github.com/JohnnyMorganz/StyLua/releases/download/v{VERSION}/stylua-macos-{ARCH}.zip",
		Version:          "0.20.0",
		Arches:           map[string]string{"x86_64": "x86_64", "arm64": "aarch64"},
		ArchiveInnerPath: "stylua",
	},
	"hadolint": {
		URLTemplate: "https://github.com/hadolint/hadolint/releases/download/v{VERSION}/hadolint-Darwin-{ARCH}",
		Version:     "2.12.0",
		Arches:      map[string]string{"x86_64": "x86_64", "arm64": "arm64"},
		// direct binary
	},
	"fd": {
		URLTemplate:      "https://github.com/sharkdp/fd/releases/download/v{VERSION}/fd-v{VERSION}-{ARCH}-apple-darwin.tar.gz",
		Version:          "10.2.0",
		Arches:           map[string]string{"x86_64": "x86_64", "arm64": "aarch64"},
		ArchiveInnerPath: "fd-v{VERSION}-{ARCH}-apple-darwin/fd",
	},
	"bat": {
		URLTemplate:      "https://github.com/sharkdp/bat/releases/download/v{VERSION}/bat-v{VERSION}-{ARCH}-apple-darwin.tar.gz",
		Version:          "0.24.0",
		Arches:           map[string]string{"x86_64": "x86_64", "arm64": "aarch64"},
		ArchiveInnerPath: "bat-v{VERSION}-{ARCH}-apple-darwin/bat",
	},
	"eza": {
		URLTemplate:      "https://github.com/eza-community/eza/releases/download/v{VERSION}/eza_{ARCH}-apple-darwin.tar.gz",
		Version:          "0.20.14",
		Arches:           map[string]string{"x86_64": "x86_64", "arm64": "aarch64"},
		ArchiveInnerPath: "eza",
	},
	"yq": {
		URLTemplate: "https://github.com/mikefarah/yq/releases/download/v{VERSION}/yq_darwin_{ARCH}",
		Version:     "4.44.3",
		Arches:      map[string]string{"x86_64": "amd64", "arm64": "arm64"},
		// direct binary
	},
	"marksman": {
		URLTemplate: "https://github.com/artempyanykh/marksman/releases/download/{VERSION}/marksman-macos",
		Version:     "2023-12-09",
		Arches:      map[string]string{"x86_64": "", "arm64": ""},
		// universal binary — arch field unused in URL
	},
}
