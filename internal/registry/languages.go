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
		ApproxSizeMB:      80, // shares nvm runtime with javascript
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

	"ruby": {
		Name:              "ruby",
		TreesitterParsers: []string{"ruby"},
		LSP:               "ruby-lsp",
		Formatters:        []string{"rubocop"},
		Linters:           []string{"rubocop"},
		ApproxSizeMB:      60,
		ApproxLSPOnlyMB:   15,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "rbenv",
			DefaultVersion:    "3.3.0",
			AvailableVersions: []string{"3.3.0", "3.2.4", "3.1.6"},
			PackageManager:    "gem",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `apt-get install -y libssl-dev libreadline-dev zlib1g-dev 2>/dev/null || true`},
				{Kind: "run", Value: `git clone https://github.com/rbenv/rbenv.git /root/.rbenv && git clone https://github.com/rbenv/ruby-build.git /root/.rbenv/plugins/ruby-build`},
				{Kind: "env", Value: `PATH=/root/.rbenv/bin:/root/.rbenv/shims:$PATH`},
				{Kind: "run", Value: `rbenv install {VERSION} && rbenv global {VERSION}`},
				{Kind: "run", Value: `gem install ruby-lsp rubocop`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/root/.rbenv/versions/{VERSION}/", InstallPath: "~/.ferry/runtimes/ruby-{VERSION}/"},
			},
			MacOSDownloads: []MacOSDownload{
				{URL: "https://cache.ruby-lang.org/pub/ruby/3.3/ruby-{VERSION}.tar.gz", Arch: "universal", ArchiveRoot: "ruby-{VERSION}/", InstallPath: "~/.ferry/runtimes/ruby-{VERSION}/"},
			},
			ShellInit: []string{
				`export PATH="$HOME/.ferry/runtimes/ruby-{VERSION}/bin:$PATH"`,
				`export GEM_HOME="$HOME/.ferry/runtimes/ruby-{VERSION}/gems"`,
			},
			LSPBuildSteps: []BuildStep{{Kind: "run", Value: `gem install ruby-lsp`}},
		},
		LSPOnlyRuntime: &Runtime{
			Manager:        "rbenv",
			DefaultVersion: "3.3.0",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `git clone https://github.com/rbenv/rbenv.git /root/.rbenv && git clone https://github.com/rbenv/ruby-build.git /root/.rbenv/plugins/ruby-build`},
				{Kind: "env", Value: `PATH=/root/.rbenv/bin:/root/.rbenv/shims:$PATH`},
				{Kind: "run", Value: `rbenv install {VERSION} && rbenv global {VERSION}`},
				{Kind: "run", Value: `gem install ruby-lsp`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/root/.rbenv/versions/{VERSION}/", InstallPath: "~/.ferry/runtimes/ruby-lsp-{VERSION}/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/ruby-lsp-{VERSION}/bin:$PATH"`},
		},
	},

	"java": {
		Name:              "java",
		TreesitterParsers: []string{"java"},
		LSP:               "jdtls",
		Formatters:        []string{"google-java-format"},
		Linters:           []string{},
		ApproxSizeMB:      120,
		ApproxLSPOnlyMB:   40,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "direct",
			DefaultVersion:    "21",
			AvailableVersions: []string{"21", "17", "11"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -LO https://github.com/adoptium/temurin21-binaries/releases/download/jdk-21+37/OpenJDK21U-jdk_{ARCH}_linux_hotspot_21_37.tar.gz && mkdir -p /opt/java/21 && tar -xzf OpenJDK21U-jdk_{ARCH}_linux_hotspot_21_37.tar.gz --strip-components=1 -C /opt/java/21 && rm *.tar.gz`},
				{Kind: "env", Value: `PATH=/opt/java/21/bin:$PATH`},
				{Kind: "env", Value: `JAVA_HOME=/opt/java/21`},
				{Kind: "run", Value: `curl -L "https://www.eclipse.org/downloads/download.php?file=/jdtls/milestones/1.31.0/jdt-language-server-1.31.0-202401111522.tar.gz" -o jdtls.tar.gz && mkdir -p /opt/jdtls && tar -xzf jdtls.tar.gz -C /opt/jdtls && rm jdtls.tar.gz`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/opt/java/21/", InstallPath: "~/.ferry/runtimes/java-21/"},
				{Container: "/opt/jdtls/", InstallPath: "~/.ferry/runtimes/jdtls/"},
			},
			MacOSDownloads: []MacOSDownload{
				{URL: "https://github.com/adoptium/temurin21-binaries/releases/download/jdk-21+37/OpenJDK21U-jdk_x64_mac_hotspot_21_37.tar.gz", Arch: "x86_64", ArchiveRoot: "jdk-21+37/Contents/Home/", InstallPath: "~/.ferry/runtimes/java-21/"},
				{URL: "https://github.com/adoptium/temurin21-binaries/releases/download/jdk-21+37/OpenJDK21U-jdk_aarch64_mac_hotspot_21_37.tar.gz", Arch: "arm64", ArchiveRoot: "jdk-21+37/Contents/Home/", InstallPath: "~/.ferry/runtimes/java-21/"},
			},
			ShellInit: []string{
				`export PATH="$HOME/.ferry/runtimes/java-21/bin:$PATH"`,
				`export JAVA_HOME="$HOME/.ferry/runtimes/java-21"`,
			},
			LSPBuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -L "https://www.eclipse.org/downloads/download.php?file=/jdtls/milestones/1.31.0/jdt-language-server-1.31.0-202401111522.tar.gz" -o jdtls.tar.gz && mkdir -p /opt/jdtls && tar -xzf jdtls.tar.gz -C /opt/jdtls && rm jdtls.tar.gz`},
			},
		},
		LSPOnlyRuntime: &Runtime{
			Manager:        "direct",
			DefaultVersion: "21",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -LO https://github.com/adoptium/temurin21-binaries/releases/download/jdk-21+37/OpenJDK21U-jdk_{ARCH}_linux_hotspot_21_37.tar.gz && mkdir -p /opt/java/21 && tar -xzf OpenJDK21U-jdk_{ARCH}_linux_hotspot_21_37.tar.gz --strip-components=1 -C /opt/java/21 && rm *.tar.gz`},
				{Kind: "env", Value: `PATH=/opt/java/21/bin:$PATH`},
				{Kind: "run", Value: `curl -L "https://www.eclipse.org/downloads/download.php?file=/jdtls/milestones/1.31.0/jdt-language-server-1.31.0-202401111522.tar.gz" -o jdtls.tar.gz && mkdir -p /opt/jdtls && tar -xzf jdtls.tar.gz -C /opt/jdtls && rm jdtls.tar.gz`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/opt/java/21/", InstallPath: "~/.ferry/runtimes/java-lsp-21/"},
				{Container: "/opt/jdtls/", InstallPath: "~/.ferry/runtimes/jdtls/"},
			},
			ShellInit: []string{
				`export PATH="$HOME/.ferry/runtimes/java-lsp-21/bin:$PATH"`,
				`export JAVA_HOME="$HOME/.ferry/runtimes/java-lsp-21"`,
			},
		},
	},

	"c": {
		Name:              "c",
		TreesitterParsers: []string{"c"},
		LSP:               "clangd",
		Formatters:        []string{"clang-format"},
		Linters:           []string{"clang-tidy"},
		ApproxSizeMB:      200,
		ApproxLSPOnlyMB:   200,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "direct",
			DefaultVersion:    "17",
			AvailableVersions: []string{"17", "16", "15"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -LO https://github.com/llvm/llvm-project/releases/download/llvmorg-{VERSION}.0.0/clang+llvm-{VERSION}.0.0-{ARCH}-linux-gnu.tar.xz && mkdir -p /opt/llvm && tar -xJf clang+llvm-{VERSION}.0.0-{ARCH}-linux-gnu.tar.xz --strip-components=1 -C /opt/llvm && rm *.tar.xz`},
				{Kind: "env", Value: `PATH=/opt/llvm/bin:$PATH`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/opt/llvm/", InstallPath: "~/.ferry/runtimes/llvm-{VERSION}/"},
			},
			MacOSDownloads: []MacOSDownload{
				{URL: "https://github.com/llvm/llvm-project/releases/download/llvmorg-{VERSION}.0.0/clang+llvm-{VERSION}.0.0-{ARCH}-apple-darwin22.0.tar.xz", Arch: "x86_64", ArchiveRoot: "clang+llvm-{VERSION}.0.0-x86_64-apple-darwin22.0/", InstallPath: "~/.ferry/runtimes/llvm-{VERSION}/"},
				{URL: "https://github.com/llvm/llvm-project/releases/download/llvmorg-{VERSION}.0.0/clang+llvm-{VERSION}.0.0-arm64-apple-darwin22.0.tar.xz", Arch: "arm64", ArchiveRoot: "clang+llvm-{VERSION}.0.0-arm64-apple-darwin22.0/", InstallPath: "~/.ferry/runtimes/llvm-{VERSION}/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/llvm-{VERSION}/bin:$PATH"`},
		},
	},

	"cpp": {
		Name:              "cpp",
		TreesitterParsers: []string{"cpp", "c"},
		LSP:               "clangd",
		Formatters:        []string{"clang-format"},
		Linters:           []string{"clang-tidy"},
		ApproxSizeMB:      0, // shares LLVM runtime with c
		ApproxLSPOnlyMB:   0,
		MacOSSupported:    true,
		Runtime:           nil, // shares with "c"
	},

	"csharp": {
		Name:              "csharp",
		TreesitterParsers: []string{"c_sharp"},
		LSP:               "csharp-ls",
		Formatters:        []string{"dotnet-csharpier"},
		Linters:           []string{},
		ApproxSizeMB:      220,
		ApproxLSPOnlyMB:   30,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "direct",
			DefaultVersion:    "8.0",
			AvailableVersions: []string{"8.0", "7.0", "6.0"},
			PackageManager:    "dotnet",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -LO https://dot.net/v1/dotnet-install.sh && chmod +x dotnet-install.sh && ./dotnet-install.sh --channel {VERSION} --install-dir /opt/dotnet && rm dotnet-install.sh`},
				{Kind: "env", Value: `PATH=/opt/dotnet:$PATH`},
				{Kind: "env", Value: `DOTNET_ROOT=/opt/dotnet`},
				{Kind: "run", Value: `dotnet tool install --global csharp-ls && dotnet tool install --global dotnet-csharpier`},
				{Kind: "env", Value: `PATH=/root/.dotnet/tools:$PATH`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/opt/dotnet/", InstallPath: "~/.ferry/runtimes/dotnet-{VERSION}/"},
				{Container: "/root/.dotnet/tools/", InstallPath: "~/.ferry/runtimes/dotnet-tools/"},
			},
			MacOSDownloads: []MacOSDownload{
				{URL: "https://dot.net/v1/dotnet-install.sh", Arch: "universal", ArchiveRoot: "", InstallPath: "~/.ferry/runtimes/dotnet-{VERSION}/"},
			},
			ShellInit: []string{
				`export PATH="$HOME/.ferry/runtimes/dotnet-{VERSION}:$HOME/.ferry/runtimes/dotnet-tools:$PATH"`,
				`export DOTNET_ROOT="$HOME/.ferry/runtimes/dotnet-{VERSION}"`,
			},
			LSPBuildSteps: []BuildStep{
				{Kind: "run", Value: `dotnet tool install --global csharp-ls`},
				{Kind: "env", Value: `PATH=/root/.dotnet/tools:$PATH`},
			},
		},
		LSPOnlyRuntime: &Runtime{
			Manager:        "direct",
			DefaultVersion: "8.0",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -LO https://dot.net/v1/dotnet-install.sh && chmod +x dotnet-install.sh && ./dotnet-install.sh --channel {VERSION} --install-dir /opt/dotnet && rm dotnet-install.sh`},
				{Kind: "env", Value: `PATH=/opt/dotnet:$PATH`},
				{Kind: "run", Value: `dotnet tool install --global csharp-ls`},
				{Kind: "env", Value: `PATH=/root/.dotnet/tools:$PATH`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/opt/dotnet/", InstallPath: "~/.ferry/runtimes/dotnet-lsp-{VERSION}/"},
				{Container: "/root/.dotnet/tools/", InstallPath: "~/.ferry/runtimes/dotnet-tools/"},
			},
			ShellInit: []string{
				`export PATH="$HOME/.ferry/runtimes/dotnet-lsp-{VERSION}:$HOME/.ferry/runtimes/dotnet-tools:$PATH"`,
			},
		},
	},

	"php": {
		Name:              "php",
		TreesitterParsers: []string{"php"},
		LSP:               "intelephense",
		Formatters:        []string{"php-cs-fixer"},
		Linters:           []string{},
		ApproxSizeMB:      50,
		ApproxLSPOnlyMB:   15,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "system",
			DefaultVersion:    "8.3",
			AvailableVersions: []string{"8.3", "8.2", "8.1"},
			PackageManager:    "composer",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `apt-get install -y software-properties-common 2>/dev/null || true && apt-get install -y php8.3 php8.3-cli php8.3-mbstring php8.3-xml php8.3-curl 2>/dev/null || true && curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer`},
				{Kind: "env", Value: `PATH=/usr/local/bin:$PATH`},
				{Kind: "run", Value: `curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash && export NVM_DIR=/root/.nvm && . $NVM_DIR/nvm.sh && nvm install lts && npm install -g intelephense`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/usr/bin/", InstallPath: "~/.ferry/runtimes/php-{VERSION}/bin/"},
				{Container: "/usr/local/bin/composer", InstallPath: "~/.ferry/runtimes/php-{VERSION}/bin/composer"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/php-{VERSION}/bin:$PATH"`},
		},
	},

	"kotlin": {
		Name:              "kotlin",
		TreesitterParsers: []string{"kotlin"},
		LSP:               "kotlin-language-server",
		Formatters:        []string{"ktlint"},
		Linters:           []string{"ktlint"},
		ApproxSizeMB:      300,
		ApproxLSPOnlyMB:   30,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "sdkman",
			DefaultVersion:    "2.0.0",
			AvailableVersions: []string{"2.0.0", "1.9.24", "1.8.22"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -s "https://get.sdkman.io" | bash`},
				{Kind: "env", Value: `SDKMAN_DIR=/root/.sdkman`},
				{Kind: "run", Value: `bash -c "source $SDKMAN_DIR/bin/sdkman-init.sh && sdk install java 21-tem && sdk install kotlin {VERSION}"`},
				{Kind: "run", Value: `curl -LO https://github.com/fwcd/kotlin-language-server/releases/download/1.3.12/server.zip && unzip server.zip -d /opt/kotlin-ls && rm server.zip`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/root/.sdkman/candidates/kotlin/{VERSION}/", InstallPath: "~/.ferry/runtimes/kotlin-{VERSION}/"},
				{Container: "/root/.sdkman/candidates/java/21.0.4-tem/", InstallPath: "~/.ferry/runtimes/java-kotlin/"},
				{Container: "/opt/kotlin-ls/", InstallPath: "~/.ferry/runtimes/kotlin-ls/"},
			},
			ShellInit: []string{
				`export PATH="$HOME/.ferry/runtimes/kotlin-{VERSION}/bin:$HOME/.ferry/runtimes/java-kotlin/bin:$PATH"`,
				`export JAVA_HOME="$HOME/.ferry/runtimes/java-kotlin"`,
			},
		},
		LSPOnlyRuntime: &Runtime{
			Manager:        "direct",
			DefaultVersion: "1.3.12",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -LO https://github.com/fwcd/kotlin-language-server/releases/download/{VERSION}/server.zip && unzip server.zip -d /opt/kotlin-ls && rm server.zip`},
				{Kind: "run", Value: `curl -LO https://github.com/adoptium/temurin21-binaries/releases/download/jdk-21+37/OpenJDK21U-jdk_{ARCH}_linux_hotspot_21_37.tar.gz && mkdir -p /opt/java/21 && tar -xzf OpenJDK21U-jdk_{ARCH}_linux_hotspot_21_37.tar.gz --strip-components=1 -C /opt/java/21 && rm *.tar.gz`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/opt/kotlin-ls/", InstallPath: "~/.ferry/runtimes/kotlin-ls/"},
				{Container: "/opt/java/21/", InstallPath: "~/.ferry/runtimes/java-kotlin/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/kotlin-ls/server/bin:$HOME/.ferry/runtimes/java-kotlin/bin:$PATH"`},
		},
	},

	"scala": {
		Name:              "scala",
		TreesitterParsers: []string{"scala"},
		LSP:               "metals",
		Formatters:        []string{"scalafmt"},
		Linters:           []string{},
		ApproxSizeMB:      250,
		ApproxLSPOnlyMB:   25,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "coursier",
			DefaultVersion:    "3.4.0",
			AvailableVersions: []string{"3.4.0", "3.3.3", "2.13.14"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -fLo cs https://github.com/coursier/launchers/raw/master/cs-{ARCH}-pc-linux && chmod +x cs && ./cs setup -y && rm cs`},
				{Kind: "env", Value: `PATH=/root/.local/share/coursier/bin:$PATH`},
				{Kind: "run", Value: `cs install scala:{VERSION} scalac:{VERSION} metals scalafmt`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/root/.local/share/coursier/", InstallPath: "~/.ferry/runtimes/coursier/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/coursier/bin:$PATH"`},
		},
		LSPOnlyRuntime: &Runtime{
			Manager:        "coursier",
			DefaultVersion: "3.4.0",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -fLo cs https://github.com/coursier/launchers/raw/master/cs-{ARCH}-pc-linux && chmod +x cs && ./cs setup -y && rm cs`},
				{Kind: "env", Value: `PATH=/root/.local/share/coursier/bin:$PATH`},
				{Kind: "run", Value: `cs install metals`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/root/.local/share/coursier/", InstallPath: "~/.ferry/runtimes/coursier/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/coursier/bin:$PATH"`},
		},
	},

	"swift": {
		Name:              "swift",
		TreesitterParsers: []string{"swift"},
		LSP:               "sourcekit-lsp",
		Formatters:        []string{"swift-format"},
		Linters:           []string{},
		ApproxSizeMB:      400,
		ApproxLSPOnlyMB:   400,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "direct",
			DefaultVersion:    "5.10.1",
			AvailableVersions: []string{"5.10.1", "5.9.2"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `apt-get install -y binutils libc6-dev libcurl4-openssl-dev libedit2 libgcc-9-dev libsqlite3-0 libstdc++-9-dev libxml2-dev libz3-dev pkg-config tzdata zlib1g-dev 2>/dev/null || true`},
				{Kind: "run", Value: `curl -LO https://download.swift.org/swift-{VERSION}-release/ubuntu2204/swift-{VERSION}-RELEASE/swift-{VERSION}-RELEASE-ubuntu22.04.tar.gz && mkdir -p /opt/swift/{VERSION} && tar -xzf swift-{VERSION}-RELEASE-ubuntu22.04.tar.gz --strip-components=1 -C /opt/swift/{VERSION} && rm *.tar.gz`},
				{Kind: "env", Value: `PATH=/opt/swift/{VERSION}/usr/bin:$PATH`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/opt/swift/{VERSION}/", InstallPath: "~/.ferry/runtimes/swift-{VERSION}/"},
			},
			MacOSDownloads: []MacOSDownload{
				{URL: "https://download.swift.org/swift-{VERSION}-release/xcode/swift-{VERSION}-RELEASE/swift-{VERSION}-RELEASE-osx.pkg", Arch: "universal", ArchiveRoot: "", InstallPath: "~/.ferry/runtimes/swift-{VERSION}/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/swift-{VERSION}/usr/bin:$PATH"`},
		},
	},

	"zig": {
		Name:              "zig",
		TreesitterParsers: []string{"zig"},
		LSP:               "zls",
		Formatters:        []string{"zig fmt"},
		Linters:           []string{},
		ApproxSizeMB:      50,
		ApproxLSPOnlyMB:   50,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "direct",
			DefaultVersion:    "0.13.0",
			AvailableVersions: []string{"0.13.0", "0.12.1"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -LO https://ziglang.org/download/{VERSION}/zig-linux-{ARCH}-{VERSION}.tar.xz && mkdir -p /opt/zig && tar -xJf zig-linux-{ARCH}-{VERSION}.tar.xz --strip-components=1 -C /opt/zig && rm *.tar.xz`},
				{Kind: "env", Value: `PATH=/opt/zig:$PATH`},
				{Kind: "run", Value: `curl -LO https://github.com/zigtools/zls/releases/download/{VERSION}/zls-{ARCH}-linux.tar.xz && mkdir -p /opt/zls && tar -xJf zls-{ARCH}-linux.tar.xz -C /opt/zls && rm *.tar.xz`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/opt/zig/", InstallPath: "~/.ferry/runtimes/zig-{VERSION}/"},
				{Container: "/opt/zls/", InstallPath: "~/.ferry/runtimes/zls/"},
			},
			MacOSDownloads: []MacOSDownload{
				{URL: "https://ziglang.org/download/{VERSION}/zig-macos-{ARCH}-{VERSION}.tar.xz", Arch: "x86_64", ArchiveRoot: "zig-macos-x86_64-{VERSION}/", InstallPath: "~/.ferry/runtimes/zig-{VERSION}/"},
				{URL: "https://ziglang.org/download/{VERSION}/zig-macos-{ARCH}-{VERSION}.tar.xz", Arch: "arm64", ArchiveRoot: "zig-macos-aarch64-{VERSION}/", InstallPath: "~/.ferry/runtimes/zig-{VERSION}/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/zig-{VERSION}:$HOME/.ferry/runtimes/zls:$PATH"`},
		},
	},

	"elixir": {
		Name:              "elixir",
		TreesitterParsers: []string{"elixir", "heex"},
		LSP:               "elixir-ls",
		Formatters:        []string{"mix format"},
		Linters:           []string{},
		ApproxSizeMB:      150,
		ApproxLSPOnlyMB:   20,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "asdf",
			DefaultVersion:    "1.17.0",
			AvailableVersions: []string{"1.17.0", "1.16.3", "1.15.8"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `apt-get install -y erlang 2>/dev/null || true`},
				{Kind: "run", Value: `git clone https://github.com/asdf-vm/asdf.git /root/.asdf --branch v0.14.0`},
				{Kind: "env", Value: `PATH=/root/.asdf/bin:/root/.asdf/shims:$PATH`},
				{Kind: "run", Value: `asdf plugin add elixir && asdf install elixir {VERSION} && asdf global elixir {VERSION}`},
				{Kind: "run", Value: `curl -LO https://github.com/elixir-lsp/elixir-ls/releases/download/v0.21.3/elixir-ls-v0.21.3.zip && mkdir -p /opt/elixir-ls && unzip elixir-ls-v0.21.3.zip -d /opt/elixir-ls && rm *.zip && chmod +x /opt/elixir-ls/language_server.sh`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/root/.asdf/installs/elixir/{VERSION}/", InstallPath: "~/.ferry/runtimes/elixir-{VERSION}/"},
				{Container: "/opt/elixir-ls/", InstallPath: "~/.ferry/runtimes/elixir-ls/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/elixir-{VERSION}/bin:$HOME/.ferry/runtimes/elixir-ls:$PATH"`},
		},
		LSPOnlyRuntime: &Runtime{
			Manager:        "direct",
			DefaultVersion: "0.21.3",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `apt-get install -y erlang 2>/dev/null || true`},
				{Kind: "run", Value: `curl -LO https://github.com/elixir-lsp/elixir-ls/releases/download/v{VERSION}/elixir-ls-v{VERSION}.zip && mkdir -p /opt/elixir-ls && unzip elixir-ls-v{VERSION}.zip -d /opt/elixir-ls && rm *.zip && chmod +x /opt/elixir-ls/language_server.sh`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/opt/elixir-ls/", InstallPath: "~/.ferry/runtimes/elixir-ls/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/elixir-ls:$PATH"`},
		},
	},

	"dart": {
		Name:              "dart",
		TreesitterParsers: []string{"dart"},
		LSP:               "dart",
		Formatters:        []string{"dart format"},
		Linters:           []string{"dart analyze"},
		ApproxSizeMB:      200,
		ApproxLSPOnlyMB:   200,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "direct",
			DefaultVersion:    "3.5.0",
			AvailableVersions: []string{"3.5.0", "3.4.4", "3.3.4"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -LO https://storage.googleapis.com/dart-archive/channels/stable/release/{VERSION}/sdk/dartsdk-linux-{ARCH}-release.zip && mkdir -p /opt/dart && unzip dartsdk-linux-{ARCH}-release.zip -d /opt && mv /opt/dart-sdk /opt/dart/{VERSION} && rm *.zip`},
				{Kind: "env", Value: `PATH=/opt/dart/{VERSION}/bin:$PATH`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/opt/dart/{VERSION}/", InstallPath: "~/.ferry/runtimes/dart-{VERSION}/"},
			},
			MacOSDownloads: []MacOSDownload{
				{URL: "https://storage.googleapis.com/dart-archive/channels/stable/release/{VERSION}/sdk/dartsdk-macos-x64-release.zip", Arch: "x86_64", ArchiveRoot: "dart-sdk/", InstallPath: "~/.ferry/runtimes/dart-{VERSION}/"},
				{URL: "https://storage.googleapis.com/dart-archive/channels/stable/release/{VERSION}/sdk/dartsdk-macos-arm64-release.zip", Arch: "arm64", ArchiveRoot: "dart-sdk/", InstallPath: "~/.ferry/runtimes/dart-{VERSION}/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/dart-{VERSION}/bin:$PATH"`},
		},
	},

	"r": {
		Name:              "r",
		TreesitterParsers: []string{"r"},
		LSP:               "r-languageserver",
		Formatters:        []string{"styler"},
		Linters:           []string{"lintr"},
		ApproxSizeMB:      300,
		ApproxLSPOnlyMB:   15,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "system",
			DefaultVersion:    "4.4",
			AvailableVersions: []string{"4.4", "4.3"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `apt-get install -y r-base r-base-dev 2>/dev/null || true`},
				{Kind: "run", Value: `Rscript -e "install.packages(c('languageserver', 'styler', 'lintr'), repos='https://cloud.r-project.org')"`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/usr/lib/R/", InstallPath: "~/.ferry/runtimes/r-{VERSION}/"},
				{Container: "/usr/local/lib/R/", InstallPath: "~/.ferry/runtimes/r-{VERSION}-local/"},
			},
			ShellInit: []string{
				`export PATH="$HOME/.ferry/runtimes/r-{VERSION}/bin:$PATH"`,
				`export R_HOME="$HOME/.ferry/runtimes/r-{VERSION}"`,
			},
		},
		LSPOnlyRuntime: &Runtime{
			Manager:        "system",
			DefaultVersion: "4.4",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `apt-get install -y r-base 2>/dev/null || true`},
				{Kind: "run", Value: `Rscript -e "install.packages('languageserver', repos='https://cloud.r-project.org')"`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/usr/lib/R/", InstallPath: "~/.ferry/runtimes/r-lsp-{VERSION}/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/r-lsp-{VERSION}/bin:$PATH"`},
		},
	},

	"lua": {
		Name:              "lua",
		TreesitterParsers: []string{"lua", "luadoc", "luap"},
		LSP:               "lua-language-server",
		Formatters:        []string{"stylua"},
		ApproxSizeMB:      5,
		ApproxLSPOnlyMB:   5,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:        "system",
			DefaultVersion: "3.9.1",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -LO https://github.com/LuaLS/lua-language-server/releases/download/3.9.1/lua-language-server-3.9.1-linux-{ARCH}.tar.gz && mkdir -p /opt/lua-ls && tar -xzf lua-language-server-3.9.1-linux-{ARCH}.tar.gz -C /opt/lua-ls && rm *.tar.gz`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/opt/lua-ls/", InstallPath: "~/.ferry/runtimes/lua-ls/"},
			},
			MacOSDownloads: []MacOSDownload{
				{URL: "https://github.com/LuaLS/lua-language-server/releases/download/3.9.1/lua-language-server-3.9.1-darwin-{ARCH}.tar.gz", Arch: "x86_64", ArchiveRoot: "", InstallPath: "~/.ferry/runtimes/lua-ls/"},
				{URL: "https://github.com/LuaLS/lua-language-server/releases/download/3.9.1/lua-language-server-3.9.1-darwin-arm64.tar.gz", Arch: "arm64", ArchiveRoot: "", InstallPath: "~/.ferry/runtimes/lua-ls/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/lua-ls/bin:$PATH"`},
		},
	},

	"bash": {
		Name:              "bash",
		TreesitterParsers: []string{"bash"},
		LSP:               "bash-language-server",
		Formatters:        []string{"shfmt"},
		Linters:           []string{"shellcheck"},
		ApproxSizeMB:      0,
		MacOSSupported:    true,
		Runtime:           nil,
	},

	"sh": {
		Name:              "sh",
		TreesitterParsers: []string{"bash"},
		LSP:               "bash-language-server",
		Formatters:        []string{"shfmt"},
		Linters:           []string{"shellcheck"},
		ApproxSizeMB:      0,
		MacOSSupported:    true,
		Runtime:           nil,
	},

	"yaml": {
		Name:              "yaml",
		TreesitterParsers: []string{"yaml"},
		LSP:               "yaml-language-server",
		Formatters:        []string{"prettier"},
		MacOSSupported:    true,
		Runtime:           nil,
	},

	"json": {
		Name:              "json",
		TreesitterParsers: []string{"json", "jsonc"},
		LSP:               "json-language-server",
		Formatters:        []string{"prettier", "jq"},
		MacOSSupported:    true,
		Runtime:           nil,
	},

	"markdown": {
		Name:              "markdown",
		TreesitterParsers: []string{"markdown", "markdown_inline"},
		LSP:               "marksman",
		Formatters:        []string{"prettier"},
		MacOSSupported:    true,
		Runtime:           nil,
	},

	"dockerfile": {
		Name:              "dockerfile",
		TreesitterParsers: []string{"dockerfile"},
		LSP:               "dockerfile-language-server",
		Formatters:        []string{"hadolint"},
		Linters:           []string{"hadolint"},
		MacOSSupported:    true,
		Runtime:           nil,
	},
}
