package registry

// llvmRuntime is the shared LLVM/Clang toolchain used by both "c" and "cpp".
// Both languages install the same LLVM release; the Manager "llvm" ensures
// they deduplicate against each other (but not against unrelated "direct"
// languages like Java) when both are selected in a profile.
var llvmRuntime = &Runtime{
	Manager:           "llvm",
	DefaultVersion:    "17",
	AvailableVersions: []string{"17", "16", "15"},
	BuildSteps: []BuildStep{
		{Kind: "run", Value: `curl -fsSL https://github.com/llvm/llvm-project/releases/download/llvmorg-{VERSION}.0.6/clang+llvm-{VERSION}.0.6-{ARCH_NATIVE}-linux-gnu-ubuntu-22.04.tar.xz -o /tmp/llvm.tar.xz && mkdir -p /opt/llvm && tar -xJf /tmp/llvm.tar.xz --strip-components=1 -C /opt/llvm && rm /tmp/llvm.tar.xz`},
		{Kind: "env", Value: `PATH=/opt/llvm/bin:$PATH`},
	},
	ContainerPaths: []ContainerPath{
		{Container: "/opt/llvm/", InstallPath: "~/.ferry/runtimes/llvm-{VERSION}/"},
	},
	MacOSDownloads: []MacOSDownload{
		{URL: "https://github.com/llvm/llvm-project/releases/download/llvmorg-{VERSION}.0.6/clang+llvm-{VERSION}.0.6-{ARCH}-apple-darwin22.0.tar.xz", Arch: "x86_64", ArchiveRoot: "clang+llvm-{VERSION}.0.6-x86_64-apple-darwin22.0/", InstallPath: "~/.ferry/runtimes/llvm-{VERSION}/"},
		{URL: "https://github.com/llvm/llvm-project/releases/download/llvmorg-{VERSION}.0.6/clang+llvm-{VERSION}.0.6-arm64-apple-darwin22.0.tar.xz", Arch: "arm64", ArchiveRoot: "clang+llvm-{VERSION}.0.6-arm64-apple-darwin22.0/", InstallPath: "~/.ferry/runtimes/llvm-{VERSION}/"},
	},
	ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/llvm-{VERSION}/bin:$PATH"`},
}

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
		// ruff is the dominant Python formatter and linter (replaces black+isort+flake8).
		// black and isort remain as alternatives for teams that prefer them.
		Formatters:      []string{"ruff", "black", "isort"},
		Linters:         []string{"ruff", "mypy"},
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
				{Kind: "run", Value: `pyenv install {VERSION} && pyenv global {VERSION} && ln -sf $(pyenv prefix) /root/.pyenv/versions/{VERSION}`},
			},
			FormatterBuildSteps: map[string][]BuildStep{
				"ruff":  {{Kind: "run", Value: `pip install ruff`}},
				"black": {{Kind: "run", Value: `pip install black`}},
				"isort": {{Kind: "run", Value: `pip install isort`}},
			},
			LinterBuildSteps: map[string][]BuildStep{
				"ruff": {}, // installed by FormatterBuildSteps["ruff"] if selected there
				"mypy": {{Kind: "run", Value: `pip install mypy`}},
			},
			ContainerPaths: []ContainerPath{
				{
					Container:   "/root/.pyenv/versions/{VERSION}/",
					InstallPath: "~/.ferry/runtimes/python-{VERSION}/",
				},
			},
			MacOSDownloads: []MacOSDownload{
				{
					// python-build-standalone provides relocatable Python builds.
					// Release date 20241002, Python 3.12.7; ArchiveRoot "python/" is the install tree.
					URL:         "https://github.com/indygreg/python-build-standalone/releases/download/20241002/cpython-3.12.7+20241002-x86_64-apple-darwin-install_only.tar.gz",
					Arch:        "x86_64",
					ArchiveRoot: "python/",
					InstallPath: "~/.ferry/runtimes/python-{VERSION}/",
				},
				{
					URL:         "https://github.com/indygreg/python-build-standalone/releases/download/20241002/cpython-3.12.7+20241002-aarch64-apple-darwin-install_only.tar.gz",
					Arch:        "arm64",
					ArchiveRoot: "python/",
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
			Manager:           "pyenv",
			DefaultVersion:    "3.12",
			AvailableVersions: []string{"3.12", "3.11", "3.10", "3.9"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl https://pyenv.run | bash`},
				{Kind: "env", Value: `PATH=/root/.pyenv/bin:/root/.pyenv/shims:$PATH`},
				{Kind: "run", Value: `pyenv install {VERSION} && pyenv global {VERSION} && ln -sf $(pyenv prefix) /root/.pyenv/versions/{VERSION}`},
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
			DefaultVersion:    "22",
			AvailableVersions: []string{"22", "20", "18"},
			PackageManager:    "npm",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash`},
				{Kind: "env", Value: `NVM_DIR=/root/.nvm`},
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && nvm install {VERSION} && nvm use {VERSION} && nvm alias default {VERSION} && ln -sf $(. $NVM_DIR/nvm.sh && nvm which {VERSION} | xargs dirname | xargs dirname) /root/.nvm/versions/node/v{VERSION}`},
			},
			FormatterBuildSteps: map[string][]BuildStep{
				"prettier": {{Kind: "run", Value: `. $NVM_DIR/nvm.sh && npm install -g prettier`}},
			},
			LinterBuildSteps: map[string][]BuildStep{
				"eslint_d": {{Kind: "run", Value: `. $NVM_DIR/nvm.sh && npm install -g eslint_d`}},
			},
			ContainerPaths: []ContainerPath{
				{
					Container:   "/root/.nvm/versions/node/v{VERSION}/",
					InstallPath: "~/.ferry/runtimes/node-{VERSION}/",
				},
			},
			MacOSDownloads: []MacOSDownload{
				{
					URL:         "https://nodejs.org/dist/v{VERSION}/node-v{VERSION}-darwin-x64.tar.gz",
					Arch:        "x86_64",
					ArchiveRoot: "node-v{VERSION}-darwin-x64/",
					InstallPath: "~/.ferry/runtimes/node-{VERSION}/",
				},
				{
					URL:         "https://nodejs.org/dist/v{VERSION}/node-v{VERSION}-darwin-arm64.tar.gz",
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
			Manager:           "nvm",
			DefaultVersion:    "22",
			AvailableVersions: []string{"22", "20", "18"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash`},
				{Kind: "env", Value: `NVM_DIR=/root/.nvm`},
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && nvm install {VERSION} && nvm use {VERSION} && ln -sf $(. $NVM_DIR/nvm.sh && nvm which {VERSION} | xargs dirname | xargs dirname) /root/.nvm/versions/node/v{VERSION}`},
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
			DefaultVersion:    "22",
			AvailableVersions: []string{"22", "20", "18"},
			PackageManager:    "npm",
			// BuildSteps cleared by deduplication when javascript is also selected.
			// AddonBuildSteps install typescript-language-server into the shared node dir.
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash`},
				{Kind: "env", Value: `NVM_DIR=/root/.nvm`},
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && nvm install {VERSION} && nvm use {VERSION} && nvm alias default {VERSION} && ln -sf $(. $NVM_DIR/nvm.sh && nvm which {VERSION} | xargs dirname | xargs dirname) /root/.nvm/versions/node/v{VERSION}`},
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && npm install -g typescript-language-server typescript`},
			},
			AddonBuildSteps: []BuildStep{
				// Runs even when deduplicated — installs typescript-ls into shared node.
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && npm install -g typescript-language-server typescript`},
			},
			FormatterBuildSteps: map[string][]BuildStep{
				"prettier": {{Kind: "run", Value: `. $NVM_DIR/nvm.sh && npm install -g prettier`}},
			},
			LinterBuildSteps: map[string][]BuildStep{
				"eslint_d": {{Kind: "run", Value: `. $NVM_DIR/nvm.sh && npm install -g eslint_d`}},
			},
			ContainerPaths: []ContainerPath{
				{
					Container:   "/root/.nvm/versions/node/v{VERSION}/",
					InstallPath: "~/.ferry/runtimes/node-{VERSION}/",
				},
			},
			MacOSDownloads: []MacOSDownload{
				{
					URL:         "https://nodejs.org/dist/v{VERSION}/node-v{VERSION}-darwin-x64.tar.gz",
					Arch:        "x86_64",
					ArchiveRoot: "node-v{VERSION}-darwin-x64/",
					InstallPath: "~/.ferry/runtimes/node-{VERSION}/",
				},
				{
					URL:         "https://nodejs.org/dist/v{VERSION}/node-v{VERSION}-darwin-arm64.tar.gz",
					Arch:        "arm64",
					ArchiveRoot: "node-v{VERSION}-darwin-arm64/",
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
				{Kind: "run", Value: `curl -fsSL https://go.dev/dl/go{VERSION}.linux-{ARCH}.tar.gz -o /tmp/go.tar.gz && tar -C /usr/local -xzf /tmp/go.tar.gz && rm /tmp/go.tar.gz`},
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
					URL:         "https://go.dev/dl/go{VERSION}.darwin-amd64.tar.gz",
					Arch:        "x86_64",
					ArchiveRoot: "go/",
					InstallPath: "~/.ferry/runtimes/go-{VERSION}/",
				},
				{
					URL:         "https://go.dev/dl/go{VERSION}.darwin-arm64.tar.gz",
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
			Manager:           "direct",
			DefaultVersion:    "1.22.5",
			AvailableVersions: []string{"1.22.5", "1.21.11", "1.20.14"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -fsSL https://go.dev/dl/go{VERSION}.linux-{ARCH}.tar.gz -o /tmp/go.tar.gz && tar -C /usr/local -xzf /tmp/go.tar.gz && rm /tmp/go.tar.gz`},
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
			DefaultVersion: "2026-04-13",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -fsSL https://github.com/rust-lang/rust-analyzer/releases/download/{VERSION}/rust-analyzer-{ARCH}-unknown-linux-musl.gz -o /tmp/ra.gz && gunzip /tmp/ra.gz && chmod +x /tmp/rust-analyzer-{ARCH}-unknown-linux-musl && mv /tmp/rust-analyzer-{ARCH}-unknown-linux-musl /usr/local/bin/rust-analyzer`},
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
			Manager:           "rbenv",
			DefaultVersion:    "3.3.0",
			AvailableVersions: []string{"3.3.0", "3.2.4", "3.1.6"},
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
			// Use Adoptium APT repo — avoids fragile release-tag/arch naming in direct downloads
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `wget -qO - https://packages.adoptium.net/artifactory/api/gpg/key/public | gpg --dearmor -o /etc/apt/keyrings/adoptium.gpg && echo "deb [signed-by=/etc/apt/keyrings/adoptium.gpg] https://packages.adoptium.net/artifactory/deb jammy main" > /etc/apt/sources.list.d/adoptium.list && apt-get update && apt-get install -y temurin-{VERSION}-jdk && rm -rf /var/lib/apt/lists/*`},
				{Kind: "env", Value: `PATH=/usr/lib/jvm/temurin-{VERSION}-jdk-amd64/bin:$PATH`},
				{Kind: "env", Value: `JAVA_HOME=/usr/lib/jvm/temurin-{VERSION}-jdk-amd64`},
				{Kind: "run", Value: `curl -fsSL "https://www.eclipse.org/downloads/download.php?file=/jdtls/milestones/1.31.0/jdt-language-server-1.31.0-202401111522.tar.gz" -o /tmp/jdtls.tar.gz && mkdir -p /opt/jdtls && tar -xzf /tmp/jdtls.tar.gz -C /opt/jdtls && rm /tmp/jdtls.tar.gz`},
			},
			// google-java-format: JAR downloaded into $JAVA_HOME/lib/ and a wrapper script
			// placed in $JAVA_HOME/bin/ so it is extracted with the JDK ContainerPath.
			FormatterBuildSteps: map[string][]BuildStep{
				"google-java-format": {
					{Kind: "run", Value: `curl -fsSL https://github.com/google/google-java-format/releases/download/v1.22.0/google-java-format-1.22.0-all-deps.jar -o $JAVA_HOME/lib/google-java-format.jar && printf '#!/bin/sh\nexec "$JAVA_HOME/bin/java" --add-exports jdk.compiler/com.sun.tools.javac.api=ALL-UNNAMED --add-exports jdk.compiler/com.sun.tools.javac.file=ALL-UNNAMED --add-exports jdk.compiler/com.sun.tools.javac.parser=ALL-UNNAMED --add-exports jdk.compiler/com.sun.tools.javac.tree=ALL-UNNAMED --add-exports jdk.compiler/com.sun.tools.javac.util=ALL-UNNAMED -jar "$JAVA_HOME/lib/google-java-format.jar" "$@"\n' > $JAVA_HOME/bin/google-java-format && chmod +x $JAVA_HOME/bin/google-java-format`},
				},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/usr/lib/jvm/temurin-{VERSION}-jdk-amd64/", InstallPath: "~/.ferry/runtimes/java-{VERSION}/"},
				{Container: "/opt/jdtls/", InstallPath: "~/.ferry/runtimes/jdtls/"},
			},
			MacOSDownloads: []MacOSDownload{
				{URL: "https://github.com/adoptium/temurin21-binaries/releases/download/jdk-21.0.5%2B11/OpenJDK21U-jdk_x64_mac_hotspot_21.0.5_11.tar.gz", Arch: "x86_64", ArchiveRoot: "jdk-21.0.5+11/Contents/Home/", InstallPath: "~/.ferry/runtimes/java-{VERSION}/"},
				{URL: "https://github.com/adoptium/temurin21-binaries/releases/download/jdk-21.0.5%2B11/OpenJDK21U-jdk_aarch64_mac_hotspot_21.0.5_11.tar.gz", Arch: "arm64", ArchiveRoot: "jdk-21.0.5+11/Contents/Home/", InstallPath: "~/.ferry/runtimes/java-{VERSION}/"},
			},
			ShellInit: []string{
				`export PATH="$HOME/.ferry/runtimes/java-{VERSION}/bin:$PATH"`,
				`export JAVA_HOME="$HOME/.ferry/runtimes/java-{VERSION}"`,
			},
			LSPBuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -fsSL "https://www.eclipse.org/downloads/download.php?file=/jdtls/milestones/1.31.0/jdt-language-server-1.31.0-202401111522.tar.gz" -o /tmp/jdtls.tar.gz && mkdir -p /opt/jdtls && tar -xzf /tmp/jdtls.tar.gz -C /opt/jdtls && rm /tmp/jdtls.tar.gz`},
			},
		},
		LSPOnlyRuntime: &Runtime{
			Manager:           "direct",
			DefaultVersion:    "21",
			AvailableVersions: []string{"21", "17", "11"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `wget -qO - https://packages.adoptium.net/artifactory/api/gpg/key/public | gpg --dearmor -o /etc/apt/keyrings/adoptium.gpg && echo "deb [signed-by=/etc/apt/keyrings/adoptium.gpg] https://packages.adoptium.net/artifactory/deb jammy main" > /etc/apt/sources.list.d/adoptium.list && apt-get update && apt-get install -y temurin-{VERSION}-jdk && rm -rf /var/lib/apt/lists/*`},
				{Kind: "env", Value: `PATH=/usr/lib/jvm/temurin-{VERSION}-jdk-amd64/bin:$PATH`},
				{Kind: "env", Value: `JAVA_HOME=/usr/lib/jvm/temurin-{VERSION}-jdk-amd64`},
				{Kind: "run", Value: `curl -fsSL "https://www.eclipse.org/downloads/download.php?file=/jdtls/milestones/1.31.0/jdt-language-server-1.31.0-202401111522.tar.gz" -o /tmp/jdtls.tar.gz && mkdir -p /opt/jdtls && tar -xzf /tmp/jdtls.tar.gz -C /opt/jdtls && rm /tmp/jdtls.tar.gz`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/usr/lib/jvm/temurin-{VERSION}-jdk-amd64/", InstallPath: "~/.ferry/runtimes/java-lsp-{VERSION}/"},
				{Container: "/opt/jdtls/", InstallPath: "~/.ferry/runtimes/jdtls/"},
			},
			ShellInit: []string{
				`export PATH="$HOME/.ferry/runtimes/java-lsp-{VERSION}/bin:$PATH"`,
				`export JAVA_HOME="$HOME/.ferry/runtimes/java-lsp-{VERSION}"`,
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
		Runtime:           llvmRuntime,
	},

	"cpp": {
		Name:              "cpp",
		TreesitterParsers: []string{"cpp", "c"},
		LSP:               "clangd",
		Formatters:        []string{"clang-format"},
		Linters:           []string{"clang-tidy"},
		ApproxSizeMB:      200, // shares LLVM runtime with "c" via Manager deduplication
		ApproxLSPOnlyMB:   200,
		MacOSSupported:    true,
		Runtime:           llvmRuntime, // same toolchain as "c"; deduplicated when both are selected
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
			Manager:           "direct",
			DefaultVersion:    "8.0",
			AvailableVersions: []string{"8.0", "7.0", "6.0"},
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
			// PHP is installed via phpenv so the runtime lives at a predictable path
			// (/root/.phpenv/versions/{VERSION}/) that can be cleanly extracted.
			// intelephense is installed via nvm (Manager: "nvm" dedup handles shared node).
			Manager:           "phpenv",
			DefaultVersion:    "8.3.0",
			AvailableVersions: []string{"8.3.0", "8.2.0", "8.1.0"},
			PackageManager:    "composer",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `apt-get install -y re2c bison libxml2-dev libcurl4-openssl-dev libssl-dev libonig-dev libsqlite3-dev libreadline-dev 2>/dev/null || true`},
				{Kind: "run", Value: `git clone https://github.com/phpenv/phpenv.git /root/.phpenv && git clone https://github.com/php-build/php-build.git /root/.phpenv/plugins/php-build`},
				{Kind: "env", Value: `PATH=/root/.phpenv/bin:/root/.phpenv/shims:$PATH`},
				{Kind: "run", Value: `phpenv install {VERSION} && phpenv global {VERSION}`},
				{Kind: "run", Value: `curl -sS https://getcomposer.org/installer | php -- --install-dir=/root/.phpenv/versions/{VERSION}/bin --filename=composer`},
			},
			// php-cs-fixer: phar downloaded into the phpenv versioned bin dir so it is
			// extracted with the PHP ContainerPath.
			FormatterBuildSteps: map[string][]BuildStep{
				"php-cs-fixer": {
					{Kind: "run", Value: `curl -fsSL https://cs.symfony.com/download/php-cs-fixer-v3.phar -o /root/.phpenv/versions/{VERSION}/bin/php-cs-fixer && chmod +x /root/.phpenv/versions/{VERSION}/bin/php-cs-fixer`},
				},
			},
			// intelephense installed into the shared nvm node via AddonBuildSteps
			AddonBuildSteps: []BuildStep{
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && npm install -g intelephense`},
			},
			ContainerPaths: []ContainerPath{
				// phpenv installs to a versioned directory — clean extraction boundary
				{Container: "/root/.phpenv/versions/{VERSION}/", InstallPath: "~/.ferry/runtimes/php-{VERSION}/"},
			},
			ShellInit: []string{
				`export PATH="$HOME/.ferry/runtimes/php-{VERSION}/bin:$HOME/.ferry/runtimes/node-{VERSION}/bin:$PATH"`,
			},
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
				{Kind: "run", Value: `curl -fsSL https://github.com/fwcd/kotlin-language-server/releases/download/1.3.13/server.zip -o /tmp/kotlin-ls.zip && unzip /tmp/kotlin-ls.zip -d /opt/kotlin-ls && rm /tmp/kotlin-ls.zip`},
			},
			// ktlint: standalone binary downloaded into the sdkman kotlin bin dir so it
			// is extracted with the Kotlin ContainerPath. Requires Java at runtime
			// (provided by the java-kotlin ContainerPath).
			FormatterBuildSteps: map[string][]BuildStep{
				"ktlint": {
					{Kind: "run", Value: `curl -fsSL https://github.com/pinterest/ktlint/releases/download/1.3.1/ktlint -o /root/.sdkman/candidates/kotlin/{VERSION}/bin/ktlint && chmod +x /root/.sdkman/candidates/kotlin/{VERSION}/bin/ktlint`},
				},
			},
			LinterBuildSteps: map[string][]BuildStep{
				"ktlint": {}, // installed by FormatterBuildSteps["ktlint"]
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
			Manager:           "direct",
			DefaultVersion:    "1.3.13",
			AvailableVersions: []string{"1.3.13", "1.3.0", "1.2.0"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -fsSL https://github.com/fwcd/kotlin-language-server/releases/download/{VERSION}/server.zip -o /tmp/kotlin-ls.zip && unzip /tmp/kotlin-ls.zip -d /opt/kotlin-ls && rm /tmp/kotlin-ls.zip`},
				{Kind: "run", Value: `wget -qO - https://packages.adoptium.net/artifactory/api/gpg/key/public | gpg --dearmor -o /etc/apt/keyrings/adoptium.gpg && echo "deb [signed-by=/etc/apt/keyrings/adoptium.gpg] https://packages.adoptium.net/artifactory/deb jammy main" > /etc/apt/sources.list.d/adoptium.list && apt-get update && apt-get install -y temurin-21-jdk && rm -rf /var/lib/apt/lists/*`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/opt/kotlin-ls/", InstallPath: "~/.ferry/runtimes/kotlin-ls/"},
				{Container: "/opt/java/21/", InstallPath: "~/.ferry/runtimes/java-kotlin/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/kotlin-ls/server/bin:$HOME/.ferry/runtimes/java-kotlin/bin:$PATH"`},
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
			DefaultVersion:    "0.14.0",
			AvailableVersions: []string{"0.14.0", "0.13.0"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -fsSL https://ziglang.org/download/{VERSION}/zig-linux-{ARCH_NATIVE}-{VERSION}.tar.xz -o /tmp/zig.tar.xz && mkdir -p /opt/zig && tar -xJf /tmp/zig.tar.xz --strip-components=1 -C /opt/zig && rm /tmp/zig.tar.xz`},
				{Kind: "env", Value: `PATH=/opt/zig:$PATH`},
				{Kind: "run", Value: `curl -fsSL https://github.com/zigtools/zls/releases/download/{VERSION}/zls-{ARCH_NATIVE}-linux.tar.xz -o /tmp/zls.tar.xz && mkdir -p /opt/zls && tar -xJf /tmp/zls.tar.xz -C /opt/zls && rm /tmp/zls.tar.xz`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/opt/zig/", InstallPath: "~/.ferry/runtimes/zig-{VERSION}/"},
				{Container: "/opt/zls/", InstallPath: "~/.ferry/runtimes/zls/"},
			},
			MacOSDownloads: []MacOSDownload{
				{URL: "https://ziglang.org/download/{VERSION}/zig-macos-x86_64-{VERSION}.tar.xz", Arch: "x86_64", ArchiveRoot: "zig-macos-x86_64-{VERSION}/", InstallPath: "~/.ferry/runtimes/zig-{VERSION}/"},
				{URL: "https://ziglang.org/download/{VERSION}/zig-macos-aarch64-{VERSION}.tar.xz", Arch: "arm64", ArchiveRoot: "zig-macos-aarch64-{VERSION}/", InstallPath: "~/.ferry/runtimes/zig-{VERSION}/"},
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
			Manager:           "direct",
			DefaultVersion:    "0.21.3",
			AvailableVersions: []string{"0.21.3", "0.20.0", "0.19.0"},
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


	"lua": {
		Name:              "lua",
		TreesitterParsers: []string{"lua", "luadoc", "luap"},
		LSP:               "lua-language-server",
		Formatters:        []string{"stylua"},
		ApproxSizeMB:      5,
		ApproxLSPOnlyMB:   5,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "system",
			DefaultVersion:    "3.18.2",
			AvailableVersions: []string{"3.18.2", "3.17.1", "3.16.0"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -fsSL https://github.com/LuaLS/lua-language-server/releases/download/{VERSION}/lua-language-server-{VERSION}-linux-x64.tar.gz -o /tmp/lua-ls.tar.gz && mkdir -p /opt/lua-ls && tar -xzf /tmp/lua-ls.tar.gz -C /opt/lua-ls && rm /tmp/lua-ls.tar.gz`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/opt/lua-ls/", InstallPath: "~/.ferry/runtimes/lua-ls/"},
			},
			MacOSDownloads: []MacOSDownload{
				{URL: "https://github.com/LuaLS/lua-language-server/releases/download/{VERSION}/lua-language-server-{VERSION}-darwin-{ARCH}.tar.gz", Arch: "x86_64", ArchiveRoot: "", InstallPath: "~/.ferry/runtimes/lua-ls/"},
				{URL: "https://github.com/LuaLS/lua-language-server/releases/download/{VERSION}/lua-language-server-{VERSION}-darwin-arm64.tar.gz", Arch: "arm64", ArchiveRoot: "", InstallPath: "~/.ferry/runtimes/lua-ls/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/lua-ls/bin:$PATH"`},
		},
	},

	// bash, yaml, json, markdown, dockerfile share the nvm runtime manager.
	// The first of these (or javascript/typescript) to appear in a profile
	// owns the ContainerPath. All subsequent ones use AddonBuildSteps to install
	// their tools into the shared node directory.
	// shfmt and shellcheck are static binaries shipped via CLIDownloads.
	// hadolint is a static binary shipped via CLIDownloads.
	// marksman is a static binary shipped via CLIDownloads.

	"bash": {
		Name:              "bash",
		TreesitterParsers: []string{"bash"},
		LSP:               "bash-language-server",
		Formatters:        []string{"shfmt"},     // static binary via CLIDownloads
		Linters:           []string{"shellcheck"}, // static binary via CLIDownloads
		ApproxSizeMB:      5,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "nvm",
			DefaultVersion:    "22",
			AvailableVersions: []string{"22", "20", "18"},
			// Full BuildSteps: run only if this is the first nvm language in the profile.
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash`},
				{Kind: "env", Value: `NVM_DIR=/root/.nvm`},
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && nvm install {VERSION} && nvm alias default {VERSION} && ln -sf $(. $NVM_DIR/nvm.sh && nvm which {VERSION} | xargs dirname | xargs dirname) /root/.nvm/versions/node/v{VERSION}`},
			},
			// AddonBuildSteps: always run, installs bash-ls into the shared node.
			AddonBuildSteps: []BuildStep{
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && npm install -g bash-language-server`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/root/.nvm/versions/node/v{VERSION}/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
			},
			MacOSDownloads: []MacOSDownload{
				{URL: "https://nodejs.org/dist/v{VERSION}/node-v{VERSION}-darwin-x64.tar.gz", Arch: "x86_64", ArchiveRoot: "node-v{VERSION}-darwin-x64/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
				{URL: "https://nodejs.org/dist/v{VERSION}/node-v{VERSION}-darwin-arm64.tar.gz", Arch: "arm64", ArchiveRoot: "node-v{VERSION}-darwin-arm64/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/node-{VERSION}/bin:$PATH"`},
		},
	},

	"yaml": {
		Name:              "yaml",
		TreesitterParsers: []string{"yaml"},
		LSP:               "yaml-language-server",
		Formatters:        []string{"prettier"},
		ApproxSizeMB:      5,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "nvm",
			DefaultVersion:    "22",
			AvailableVersions: []string{"22", "20", "18"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash`},
				{Kind: "env", Value: `NVM_DIR=/root/.nvm`},
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && nvm install {VERSION} && nvm alias default {VERSION} && ln -sf $(. $NVM_DIR/nvm.sh && nvm which {VERSION} | xargs dirname | xargs dirname) /root/.nvm/versions/node/v{VERSION}`},
			},
			AddonBuildSteps: []BuildStep{
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && npm install -g yaml-language-server prettier`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/root/.nvm/versions/node/v{VERSION}/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
			},
			MacOSDownloads: []MacOSDownload{
				{URL: "https://nodejs.org/dist/v{VERSION}/node-v{VERSION}-darwin-x64.tar.gz", Arch: "x86_64", ArchiveRoot: "node-v{VERSION}-darwin-x64/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
				{URL: "https://nodejs.org/dist/v{VERSION}/node-v{VERSION}-darwin-arm64.tar.gz", Arch: "arm64", ArchiveRoot: "node-v{VERSION}-darwin-arm64/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/node-{VERSION}/bin:$PATH"`},
		},
	},

	"json": {
		Name:              "json",
		TreesitterParsers: []string{"json", "jsonc"},
		LSP:               "json-language-server",
		// prettier is the standard; jq ships as a CLI binary so it's always available.
		Formatters:     []string{"prettier", "jq"},
		ApproxSizeMB:   5,
		MacOSSupported: true,
		Runtime: &Runtime{
			Manager:           "nvm",
			DefaultVersion:    "22",
			AvailableVersions: []string{"22", "20", "18"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash`},
				{Kind: "env", Value: `NVM_DIR=/root/.nvm`},
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && nvm install {VERSION} && nvm alias default {VERSION} && ln -sf $(. $NVM_DIR/nvm.sh && nvm which {VERSION} | xargs dirname | xargs dirname) /root/.nvm/versions/node/v{VERSION}`},
			},
			// vscode-langservers-extracted provides json-language-server + html/css servers.
			AddonBuildSteps: []BuildStep{
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && npm install -g vscode-langservers-extracted prettier`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/root/.nvm/versions/node/v{VERSION}/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
			},
			MacOSDownloads: []MacOSDownload{
				{URL: "https://nodejs.org/dist/v{VERSION}/node-v{VERSION}-darwin-x64.tar.gz", Arch: "x86_64", ArchiveRoot: "node-v{VERSION}-darwin-x64/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
				{URL: "https://nodejs.org/dist/v{VERSION}/node-v{VERSION}-darwin-arm64.tar.gz", Arch: "arm64", ArchiveRoot: "node-v{VERSION}-darwin-arm64/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/node-{VERSION}/bin:$PATH"`},
		},
	},

	"markdown": {
		Name:              "markdown",
		TreesitterParsers: []string{"markdown", "markdown_inline"},
		// marksman is a static binary shipped via CLIDownloads — no node needed for the LSP.
		LSP:            "marksman",
		Formatters:     []string{"prettier"},
		ApproxSizeMB:   5,
		MacOSSupported: true,
		Runtime: &Runtime{
			Manager:           "nvm",
			DefaultVersion:    "22",
			AvailableVersions: []string{"22", "20", "18"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash`},
				{Kind: "env", Value: `NVM_DIR=/root/.nvm`},
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && nvm install {VERSION} && nvm alias default {VERSION} && ln -sf $(. $NVM_DIR/nvm.sh && nvm which {VERSION} | xargs dirname | xargs dirname) /root/.nvm/versions/node/v{VERSION}`},
			},
			// prettier handles markdown formatting; marksman LSP is a static binary.
			AddonBuildSteps: []BuildStep{
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && npm install -g prettier`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/root/.nvm/versions/node/v{VERSION}/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
			},
			MacOSDownloads: []MacOSDownload{
				{URL: "https://nodejs.org/dist/v{VERSION}/node-v{VERSION}-darwin-x64.tar.gz", Arch: "x86_64", ArchiveRoot: "node-v{VERSION}-darwin-x64/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
				{URL: "https://nodejs.org/dist/v{VERSION}/node-v{VERSION}-darwin-arm64.tar.gz", Arch: "arm64", ArchiveRoot: "node-v{VERSION}-darwin-arm64/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/node-{VERSION}/bin:$PATH"`},
		},
	},

	// html, css share the nvm runtime manager with javascript/typescript/bash/yaml/json/markdown.
	// vscode-langservers-extracted bundles the HTML, CSS, and JSON language servers in one
	// npm package. Installing it here is idempotent if json, yaml, or markdown are also
	// selected — npm deduplicates the install automatically.

	"html": {
		Name:              "html",
		TreesitterParsers: []string{"html"},
		LSP:               "vscode-html-language-server",
		Formatters:        []string{"prettier"},
		ApproxSizeMB:      5,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "nvm",
			DefaultVersion:    "22",
			AvailableVersions: []string{"22", "20", "18"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash`},
				{Kind: "env", Value: `NVM_DIR=/root/.nvm`},
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && nvm install {VERSION} && nvm alias default {VERSION} && ln -sf $(. $NVM_DIR/nvm.sh && nvm which {VERSION} | xargs dirname | xargs dirname) /root/.nvm/versions/node/v{VERSION}`},
			},
			AddonBuildSteps: []BuildStep{
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && npm install -g vscode-langservers-extracted prettier`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/root/.nvm/versions/node/v{VERSION}/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
			},
			MacOSDownloads: []MacOSDownload{
				{URL: "https://nodejs.org/dist/v{VERSION}/node-v{VERSION}-darwin-x64.tar.gz", Arch: "x86_64", ArchiveRoot: "node-v{VERSION}-darwin-x64/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
				{URL: "https://nodejs.org/dist/v{VERSION}/node-v{VERSION}-darwin-arm64.tar.gz", Arch: "arm64", ArchiveRoot: "node-v{VERSION}-darwin-arm64/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/node-{VERSION}/bin:$PATH"`},
		},
	},

	"css": {
		Name:              "css",
		TreesitterParsers: []string{"css"},
		LSP:               "vscode-css-language-server",
		Formatters:        []string{"prettier"},
		Linters:           []string{"stylelint"},
		ApproxSizeMB:      5,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:           "nvm",
			DefaultVersion:    "22",
			AvailableVersions: []string{"22", "20", "18"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash`},
				{Kind: "env", Value: `NVM_DIR=/root/.nvm`},
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && nvm install {VERSION} && nvm alias default {VERSION} && ln -sf $(. $NVM_DIR/nvm.sh && nvm which {VERSION} | xargs dirname | xargs dirname) /root/.nvm/versions/node/v{VERSION}`},
			},
			// stylelint-config-standard is the baseline ruleset for stylelint.
			AddonBuildSteps: []BuildStep{
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && npm install -g vscode-langservers-extracted prettier stylelint stylelint-config-standard`},
			},
			LinterBuildSteps: map[string][]BuildStep{
				"stylelint": {}, // installed via AddonBuildSteps above
			},
			ContainerPaths: []ContainerPath{
				{Container: "/root/.nvm/versions/node/v{VERSION}/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
			},
			MacOSDownloads: []MacOSDownload{
				{URL: "https://nodejs.org/dist/v{VERSION}/node-v{VERSION}-darwin-x64.tar.gz", Arch: "x86_64", ArchiveRoot: "node-v{VERSION}-darwin-x64/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
				{URL: "https://nodejs.org/dist/v{VERSION}/node-v{VERSION}-darwin-arm64.tar.gz", Arch: "arm64", ArchiveRoot: "node-v{VERSION}-darwin-arm64/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/node-{VERSION}/bin:$PATH"`},
		},
	},

	// sql uses sqls — a self-contained Go binary with no runtime dependency.
	// The binary is placed in /opt/sqls/bin/ so it extracts as a proper directory
	// (matching the pattern used by Zig, lua-language-server, etc.) rather than a
	// single-file component that would install as a directory on the target.
	"sql": {
		Name:              "sql",
		TreesitterParsers: []string{"sql"},
		LSP:               "sqls",
		ApproxSizeMB:      5,
		MacOSSupported:    true,
		Runtime: &Runtime{
			Manager:        "none",
			DefaultVersion: "0.2.28",
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `mkdir -p /opt/sqls/bin && curl -fsSL https://github.com/lighttiger2505/sqls/releases/download/v{VERSION}/sqls_linux_{ARCH} -o /opt/sqls/bin/sqls && chmod +x /opt/sqls/bin/sqls`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/opt/sqls/", InstallPath: "~/.ferry/runtimes/sqls/"},
			},
			MacOSDownloads: []MacOSDownload{
				{URL: "https://github.com/lighttiger2505/sqls/releases/download/v{VERSION}/sqls_darwin_amd64", Arch: "x86_64", ArchiveRoot: "", InstallPath: "~/.ferry/runtimes/sqls/"},
				{URL: "https://github.com/lighttiger2505/sqls/releases/download/v{VERSION}/sqls_darwin_arm64", Arch: "arm64", ArchiveRoot: "", InstallPath: "~/.ferry/runtimes/sqls/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/sqls/bin:$PATH"`},
		},
	},

	"dockerfile": {
		Name:              "dockerfile",
		TreesitterParsers: []string{"dockerfile"},
		LSP:               "dockerfile-language-server",
		// hadolint ships as a static binary via CLIDownloads.
		// It is a linter only — conform.nvim has no hadolint formatter.
		Formatters:     []string{},
		Linters:        []string{"hadolint"},
		ApproxSizeMB:   5,
		MacOSSupported: true,
		Runtime: &Runtime{
			Manager:           "nvm",
			DefaultVersion:    "22",
			AvailableVersions: []string{"22", "20", "18"},
			BuildSteps: []BuildStep{
				{Kind: "run", Value: `curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash`},
				{Kind: "env", Value: `NVM_DIR=/root/.nvm`},
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && nvm install {VERSION} && nvm alias default {VERSION} && ln -sf $(. $NVM_DIR/nvm.sh && nvm which {VERSION} | xargs dirname | xargs dirname) /root/.nvm/versions/node/v{VERSION}`},
			},
			AddonBuildSteps: []BuildStep{
				{Kind: "run", Value: `. $NVM_DIR/nvm.sh && npm install -g dockerfile-language-server-nodejs`},
			},
			ContainerPaths: []ContainerPath{
				{Container: "/root/.nvm/versions/node/v{VERSION}/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
			},
			MacOSDownloads: []MacOSDownload{
				{URL: "https://nodejs.org/dist/v{VERSION}/node-v{VERSION}-darwin-x64.tar.gz", Arch: "x86_64", ArchiveRoot: "node-v{VERSION}-darwin-x64/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
				{URL: "https://nodejs.org/dist/v{VERSION}/node-v{VERSION}-darwin-arm64.tar.gz", Arch: "arm64", ArchiveRoot: "node-v{VERSION}-darwin-arm64/", InstallPath: "~/.ferry/runtimes/node-{VERSION}/"},
			},
			ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/node-{VERSION}/bin:$PATH"`},
		},
	},
}
