package registry

// languageRegistry is the source-of-truth map of all supported languages.
// Add new languages here; no other file needs to change.
var languageRegistry = map[string]Language{
	"python": {
		Name:              "python",
		TreesitterParsers: []string{"python"},
		LSP:               "pyright",
		AlternateLSPs: []AlternateLSP{
			{
				Name: "pylsp",
				BuildSteps: []BuildStep{
					{Kind: "run", Value: `pip install python-lsp-server`},
				},
			},
		},
		Formatters:      []string{"black", "isort"},
		Linters:         []string{"flake8", "mypy"},
		ApproxSizeMB:    150,
		ApproxLSPOnlyMB: 25,
		MacOSSupported:  true,
		Runtime: &Runtime{
			Manager:           "pyenv",
			DefaultVersion:    "3.12",
			AvailableVersions: []string{"3.12", "3.11", "3.10", "3.9"},
			PackageManager:    "pip",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl https://pyenv.run | bash`},
				{Kind: "env", Value: `PATH=/root/.pyenv/bin:/root/.pyenv/shims:$PATH`},
				{Kind: "run", Value: `pyenv install {VERSION} && pyenv global {VERSION}`},
				{Kind: "run", Value: `pip install black isort flake8 mypy`},
			},
			ContainerPaths: []ContainerPath{
				{
					Container:   "/root/.pyenv/versions/{VERSION}/",
					InstallPath: "~/.ferry/runtimes/python-{VERSION}/",
				},
			},
			MacOSDownloads: []MacOSDownload{
				{
					URL:         "https://www.python.org/ftp/python/{VERSION}/python-{VERSION}-macos11.pkg",
					Arch:        "universal",
					ArchiveRoot: "",
					InstallPath: "~/.ferry/runtimes/python-{VERSION}/",
				},
			},
			ShellInit: []string{
				`export PATH="$HOME/.ferry/runtimes/python-{VERSION}/bin:$PATH"`,
			},
			LSPBuildSteps: []BuildStep{
				{Kind: "run", Value: `pip install pyright`},
			},
		},
		LSPOnlyRuntime: &Runtime{
			Manager:        "pyenv",
			DefaultVersion: "3.12",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl https://pyenv.run | bash`},
				{Kind: "env", Value: `PATH=/root/.pyenv/bin:/root/.pyenv/shims:$PATH`},
				{Kind: "run", Value: `pyenv install {VERSION} && pyenv global {VERSION}`},
				{Kind: "run", Value: `pip install pyright`},
			},
			ContainerPaths: []ContainerPath{
				{
					Container:   "/root/.pyenv/versions/{VERSION}/",
					InstallPath: "~/.ferry/runtimes/python-lsp-{VERSION}/",
				},
			},
			ShellInit: []string{
				`export PATH="$HOME/.ferry/runtimes/python-lsp-{VERSION}/bin:$PATH"`,
			},
		},
	},

	"javascript": {
		Name:              "javascript",
		TreesitterParsers: []string{"javascript", "jsdoc"},
		LSP:               "typescript-language-server",
		Formatters:        []string{"prettier"},
		Linters:           []string{"eslint_d"},
		ApproxSizeMB:      80,
		ApproxLSPOnlyMB:   15,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "nvm",
			DefaultVersion:    "lts",
			AvailableVersions: []string{"lts", "22", "20", "18"},
			PackageManager:    "npm",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash`},
				{Kind: "env", Value: `NVM_DIR=/root/.nvm`},
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && nvm install {VERSION} && nvm use {VERSION} && nvm alias default {VERSION}`},
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && npm install -g typescript-language-server typescript prettier eslint_d`},
			},
			ContainerPaths: []ContainerPath{
				{
					Container:   "/root/.nvm/versions/node/v{VERSION}/",
					InstallPath: "~/.ferry/runtimes/node-{VERSION}/",
				},
			},
			MacOSDownloads: []MacOSDownload{
				{
					URL:         "https://nodejs.org/dist/v{VERSION}/node-v{VERSION}-darwin-{ARCH}.tar.gz",
					Arch:        "x86_64",
					ArchiveRoot: "node-v{VERSION}-darwin-x64/",
					InstallPath: "~/.ferry/runtimes/node-{VERSION}/",
				},
				{
					URL:         "https://nodejs.org/dist/v{VERSION}/node-v{VERSION}-darwin-{ARCH}.tar.gz",
					Arch:        "arm64",
					ArchiveRoot: "node-v{VERSION}-darwin-arm64/",
					InstallPath: "~/.ferry/runtimes/node-{VERSION}/",
				},
			},
			ShellInit: []string{
				`export PATH="$HOME/.ferry/runtimes/node-{VERSION}/bin:$PATH"`,
			},
			LSPBuildSteps: []BuildStep{
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && npm install -g typescript-language-server typescript`},
			},
		},
		LSPOnlyRuntime: &Runtime{
			Manager:        "nvm",
			DefaultVersion: "lts",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash`},
				{Kind: "env", Value: `NVM_DIR=/root/.nvm`},
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && nvm install {VERSION} && nvm use {VERSION}`},
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && npm install -g typescript-language-server typescript`},
			},
			ContainerPaths: []ContainerPath{
				{
					Container:   "/root/.nvm/versions/node/v{VERSION}/",
					InstallPath: "~/.ferry/runtimes/node-lsp-{VERSION}/",
				},
			},
			ShellInit: []string{
				`export PATH="$HOME/.ferry/runtimes/node-lsp-{VERSION}/bin:$PATH"`,
			},
		},
	},

	"typescript": {
		Name:              "typescript",
		TreesitterParsers: []string{"typescript", "javascript", "tsx", "jsdoc"},
		LSP:               "typescript-language-server",
		Formatters:        []string{"prettier"},
		Linters:           []string{"eslint_d"},
		ApproxSizeMB:      0, // shares nvm runtime with javascript
		ApproxLSPOnlyMB:   0,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "nvm",
			DefaultVersion:    "lts",
			AvailableVersions: []string{"lts", "22", "20", "18"},
			PackageManager:    "npm",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash`},
				{Kind: "env", Value: `NVM_DIR=/root/.nvm`},
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && nvm install {VERSION} && nvm use {VERSION} && nvm alias default {VERSION}`},
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && npm install -g typescript-language-server typescript prettier eslint_d`},
			},
			ContainerPaths: []ContainerPath{
				{
					Container:   "/root/.nvm/versions/node/v{VERSION}/",
					InstallPath: "~/.ferry/runtimes/node-{VERSION}/",
				},
			},
			ShellInit: []string{
				`export PATH="$HOME/.ferry/runtimes/node-{VERSION}/bin:$PATH"`,
			},
		},
	},

	"go": {
		Name:              "go",
		TreesitterParsers: []string{"go", "gomod", "gosum", "gowork"},
		LSP:               "gopls",
		Formatters:        []string{"gofmt", "goimports"},
		Linters:           []string{"staticcheck"},
		ApproxSizeMB:      130,
		ApproxLSPOnlyMB:   20,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "direct",
			DefaultVersion:    "1.22.5",
			AvailableVersions: []string{"1.22.5", "1.21.11", "1.20.14"},
			PackageManager:    "go",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -LO https://go.dev/dl/go{VERSION}.linux-{ARCH}.tar.gz && tar -C /usr/local -xzf go{VERSION}.linux-{ARCH}.tar.gz && rm go{VERSION}.linux-{ARCH}.tar.gz`},
				{Kind: "env", Value: `PATH=/usr/local/go/bin:$PATH`},
				{Kind: "env", Value: `GOPATH=/root/.ferry-gopath`},
				{Kind: "run", Value: `go install golang.org/x/tools/gopls@latest && go install golang.org/x/tools/cmd/goimports@latest && go install honnef.co/go/tools/cmd/staticcheck@latest`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/usr/local/go/", InstallPath: "~/.ferry/runtimes/go-{VERSION}/"},
				{Container: "/root/.ferry-gopath/", InstallPath: "~/.ferry/runtimes/go-tools/"},
			},
			MacOSDownloads: []MacOSDownload{
				{
					URL:         "https://go.dev/dl/go{VERSION}.darwin-{ARCH}.tar.gz",
					Arch:        "x86_64",
					ArchiveRoot: "go/",
					InstallPath: "~/.ferry/runtimes/go-{VERSION}/",
				},
				{
					URL:         "https://go.dev/dl/go{VERSION}.darwin-{ARCH}.tar.gz",
					Arch:        "arm64",
					ArchiveRoot: "go/",
					InstallPath: "~/.ferry/runtimes/go-{VERSION}/",
				},
			},
			ShellInit: []string{
				`export PATH="$HOME/.ferry/runtimes/go-{VERSION}/bin:$HOME/.ferry/runtimes/go-tools/bin:$PATH"`,
				`export GOPATH="$HOME/.ferry/runtimes/go-tools"`,
			},
			LSPBuildSteps: []BuildStep{
				{Kind: "run", Value: `go install golang.org/x/tools/gopls@latest`},
			},
		},
		LSPOnlyRuntime: &Runtime{
			Manager:        "direct",
			DefaultVersion: "1.22.5",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -LO https://go.dev/dl/go{VERSION}.linux-{ARCH}.tar.gz && tar -C /usr/local -xzf go{VERSION}.linux-{ARCH}.tar.gz && rm go{VERSION}.linux-{ARCH}.tar.gz`},
				{Kind: "env", Value: `PATH=/usr/local/go/bin:$PATH`},
				{Kind: "env", Value: `GOPATH=/root/.ferry-gopath`},
				{Kind: "run", Value: `go install golang.org/x/tools/gopls@latest`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/usr/local/go/", InstallPath: "~/.ferry/runtimes/go-{VERSION}/"},
				{Container: "/root/.ferry-gopath/", InstallPath: "~/.ferry/runtimes/go-tools/"},
			},
			ShellInit: []string{
				`export PATH="$HOME/.ferry/runtimes/go-{VERSION}/bin:$HOME/.ferry/runtimes/go-tools/bin:$PATH"`,
				`export GOPATH="$HOME/.ferry/runtimes/go-tools"`,
			},
		},
	},

	"rust": {
		Name:              "rust",
		TreesitterParsers: []string{"rust", "toml"},
		LSP:               "rust-analyzer",
		Formatters:        []string{"rustfmt"},
		Linters:           []string{},
		ApproxSizeMB:      800,
		ApproxLSPOnlyMB:   80,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "rustup",
			DefaultVersion:    "stable",
			AvailableVersions: []string{"stable", "beta", "nightly"},
			PackageManager:    "cargo",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain {VERSION}`},
				{Kind: "env", Value: `PATH=/root/.cargo/bin:$PATH`},
				{Kind: "run", Value: `rustup component add rustfmt clippy rust-analyzer`},
			},
			ContainerPaths: []ContainerPath{
				{
					Container:   "/root/.rustup/toolchains/{VERSION}-{ARCH}-unknown-linux-gnu/",
					InstallPath: "~/.ferry/runtimes/rust-{VERSION}/",
				},
				{
					Container:   "/root/.cargo/bin/",
					InstallPath: "~/.ferry/runtimes/cargo-bin/",
				},
			},
			MacOSDownloads: []MacOSDownload{
				{
					URL:         "https://static.rust-lang.org/dist/rust-{VERSION}-{ARCH}-apple-darwin.tar.gz",
					Arch:        "x86_64",
					ArchiveRoot: "rust-{VERSION}-x86_64-apple-darwin/",
					InstallPath: "~/.ferry/runtimes/rust-{VERSION}/",
				},
				{
					URL:         "https://static.rust-lang.org/dist/rust-{VERSION}-aarch64-apple-darwin.tar.gz",
					Arch:        "arm64",
					ArchiveRoot: "rust-{VERSION}-aarch64-apple-darwin/",
					InstallPath: "~/.ferry/runtimes/rust-{VERSION}/",
				},
			},
			ShellInit: []string{
				`export PATH="$HOME/.ferry/runtimes/rust-{VERSION}/bin:$HOME/.ferry/runtimes/cargo-bin:$PATH"`,
			},
			LSPBuildSteps: []BuildStep{
				{Kind: "run", Value: `rustup component add rust-analyzer`},
			},
		},
		LSPOnlyRuntime: &Runtime{
			Manager:        "direct",
			DefaultVersion: "2024-01-29",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -LO https://github.com/rust-lang/rust-analyzer/releases/download/{VERSION}/rust-analyzer-{ARCH}-unknown-linux-musl.gz && gunzip rust-analyzer-{ARCH}-unknown-linux-musl.gz && chmod +x rust-analyzer-{ARCH}-unknown-linux-musl && mv rust-analyzer-{ARCH}-unknown-linux-musl /usr/local/bin/rust-analyzer`},
			},
			ContainerPaths: []ContainerPath{
				{
					Container:   "/usr/local/bin/rust-analyzer",
					InstallPath: "~/.ferry/runtimes/rust-lsp/bin/rust-analyzer",
				},
			},
			ShellInit: []string{
				`export PATH="$HOME/.ferry/runtimes/rust-lsp/bin:$PATH"`,
			},
		},
	},

	"lua": {
		Name:              "lua",
		TreesitterParsers: []string{"lua", "luadoc", "luap"},
		LSP:               "lua-language-server",
		Formatters:        []string{"stylua"},
		Runtime:           &Runtime{Manager: "none"},
	},
	"dockerfile": {
		Name:              "dockerfile",
		TreesitterParsers: []string{"dockerfile"},
		LSP:               "dockerfile-language-server",
		Formatters:        []string{"hadolint"},
		Linters:           []string{"hadolint"},
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
