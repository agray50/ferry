# ferry

Ship your dev environment to remote machines.

Ferry bundles your neovim config, shell setup, CLI tools, and programming language runtimes into a portable archive and deploys it to remote VMs over SSH. The target machine needs no internet access — everything is pre-built and shipped.

## The problem

Setting up a dev environment on a remote VM is tedious. Package managers differ, version managers need bootstrapping, LSP servers have to be compiled or downloaded, plugins need syncing. On airgapped machines it's worse — nothing can be downloaded at all.

Ferry solves this by building everything on your local machine (in Docker for Linux targets) and shipping only the compiled artifacts. The target gets a working neovim with LSPs, treesitter parsers, shell config, language runtimes, and CLI tools — extracted from tarballs with no network activity required.

## How it works

```
ferry init       → configure profiles (languages, plugins, CLI tools)
ferry bundle     → build component archives (Docker for Linux, direct downloads for macOS)
ferry bootstrap  → upload archives to target and run install script
ferry update     → ship only changed components on subsequent deploys
```

Components are content-addressed: the same runtime at the same version is built once and reused across profiles and targets. A delta update ships only what changed.

## Requirements

**On your machine:**
- Go 1.21+
- Docker (required only for Linux bundles; macOS bundles use direct downloads)
- SSH access to target machines

**On target machines:**
- zsh 5.8+ (bash supported, zsh is primary)
- git, curl, tmux (or whatever packages you configure in `ferry.lock`)
- No internet access required

## Installation

```bash
go install github.com/anthropics/ferry@latest
```

Or build from source:

```bash
git clone https://github.com/anthropics/ferry.git
cd ferry
go build -o ferry .
```

## Quick start

```bash
# 1. Create a ferry.lock in your working directory
ferry init

# 2. Build bundles (requires Docker for Linux targets)
ferry bundle

# 3. Deploy to a remote machine
ferry bootstrap ubuntu@my-vm

# 4. SSH in
ssh ubuntu@my-vm

# 5. When your config changes, update only what changed
ferry update ubuntu@my-vm
```

---

## Commands

### `ferry init`

Interactive TUI wizard for creating and editing profiles. Walks through language selection (with tier and version picker), neovim plugin configuration, CLI tool selection, and shell setup.

```bash
ferry init                    # open profile manager
ferry init --profile backend  # jump directly to a specific profile
```

### `ferry bundle`

Builds component archives for all configured profiles. Linux targets use Docker (ubuntu:22.04 base); macOS targets use direct binary downloads without Docker.

```bash
ferry bundle                          # build all profiles, all architectures
ferry bundle --profile backend        # one profile only
ferry bundle --arch x86_64            # one architecture only
ferry bundle --os linux               # one OS only
ferry bundle --force                  # rebuild even if up to date
```

Build tracks:

| Track | Build method | Base |
|---|---|---|
| `x86_64-linux` | Docker | ubuntu:22.04 |
| `arm64-linux` | Docker | ubuntu:22.04 |
| `x86_64-darwin` | Direct downloads | — |
| `arm64-darwin` | Direct downloads | — |

Bundles are stored in `~/.ferry/bundles/` as content-addressed `.tar.zst` files.

### `ferry bootstrap <user@host>`

First-time deployment to a new target. Connects over SSH, detects remote architecture and libc, uploads all components, runs the install script, and validates the result.

```bash
ferry bootstrap ubuntu@10.0.0.5
ferry bootstrap ubuntu@10.0.0.5 --profile minimal
```

Pre-flight checks before uploading:
- Remote OS, arch, and libc detected automatically (`uname -s`, `uname -m`, `ldd --version` on Linux)
- Disk space: requires ~2× the bundle size free
- zsh version: must meet configured minimum
- Required packages: git, curl, tmux, zsh (configurable)

Transfer time is estimated at 10/50/100 Mbps and shown before confirming.

### `ferry update <user@host>`

Sends a delta update to an already-bootstrapped target. Downloads the remote manifest, diffs it against the local build, and ships only new or changed components. An interactive TUI lets you review and deselect individual components before applying.

```bash
ferry update ubuntu@10.0.0.5
ferry update ubuntu@10.0.0.5 --dry-run       # show what would change
ferry update ubuntu@10.0.0.5 --force          # update even if hash unchanged
ferry update ubuntu@10.0.0.5 --profile server # switch to a different profile
```

Update safety:
- Warns if nvim is currently running on the target (offers to wait)
- Uses an atomic POSIX lock (`~/.ferry/.update.lock`) to prevent concurrent updates
- The previous environment stays intact until the apply step completes

### `ferry ls`

Lists all profiles defined in `ferry.lock` and the deployment status of known targets.

```bash
ferry ls                # static table
ferry ls --interactive  # open the interactive profile manager
```

### `ferry clean <user@host>`

Removes ferry data from a target.

```bash
ferry clean ubuntu@10.0.0.5         # remove component store only (~/.ferry/store/)
ferry clean ubuntu@10.0.0.5 --full  # full teardown (requires typing "yes")
```

`--full` removes: `~/.ferry/`, `~/.local/bin/nvim` and all ferry-installed binaries, `~/.local/share/nvim/`, `~/.local/share/nvim-dist/`, `~/.config/nvim/plugin/ferry.lua`, `~/.config/nvim/lua/ferry.lua`, and the ferry PATH line from shell RC files. Only the two files ferry wrote are removed from `~/.config/nvim/` — the rest of your neovim config is untouched.

---

## Custom tools (`ferry-tools.yaml`)

Ferry ships a curated set of formatters, linters, and LSP servers for each language. If you need a tool that isn't in the built-in registry — or want to override a default — create a `ferry-tools.yaml` in your project directory (or `~/.ferry/tools.yaml` for a global default). Project-level wins on conflicts.

### Format

```yaml
formatters:
  <name>:
    build: "<shell command to install on Linux>"
    macos_downloads:          # optional — required for macOS bundles
      - arch: x86_64
        url: "https://..."
        inner_path: ""        # path to binary inside archive; omit for direct binary downloads
        archive_root: ""      # prefix to strip from archive entries; omit if archive is flat
      - arch: arm64
        url: "https://..."

linters:
  <name>:
    build: "<shell command>"
    macos_downloads: [...]

lsps:
  <name>:
    build: "<shell command>"
    macos_downloads: [...]
```

The `build` command runs inside the Docker container during `ferry bundle` (Linux tracks). For macOS tracks, ferry downloads the binary from `macos_downloads` instead. If `macos_downloads` is absent for a custom tool, it is silently skipped on macOS builds.

Once defined in `ferry-tools.yaml`, reference the tool by name in `ferry.lock` the same way as any built-in tool:

```toml
[[profiles.default.languages]]
name = "typescript"
tier = "full"
formatters = ["biome"]        # custom formatter from ferry-tools.yaml
linters    = ["oxlint"]       # custom linter from ferry-tools.yaml

[[profiles.default.languages]]
name = "python"
tier = "full"
lsp = "pylsp"                 # override with a different built-in LSP (no tools file needed)
```

### Example: biome (formatter) + oxlint (linter)

```yaml
# ferry-tools.yaml
formatters:
  biome:
    build: ". $NVM_DIR/nvm.sh && npm install -g @biomejs/biome"
    macos_downloads:
      - arch: x86_64
        url: "https://github.com/biomejs/biome/releases/download/cli%2Fv1.9.4/biome-darwin-x64"
      - arch: arm64
        url: "https://github.com/biomejs/biome/releases/download/cli%2Fv1.9.4/biome-darwin-arm64"

linters:
  oxlint:
    build: ". $NVM_DIR/nvm.sh && npm install -g oxlint"
    macos_downloads:
      - arch: x86_64
        url: "https://github.com/oxc-project/oxc/releases/download/oxlint_v0.15.0/oxlint-darwin-x64"
      - arch: arm64
        url: "https://github.com/oxc-project/oxc/releases/download/oxlint_v0.15.0/oxlint-darwin-arm64"
```

### Example: vtsls (custom LSP for TypeScript)

```yaml
# ferry-tools.yaml
lsps:
  vtsls:
    build: ". $NVM_DIR/nvm.sh && npm install -g @vtsls/language-server"
```

```toml
# ferry.lock
[[profiles.default.languages]]
name = "typescript"
tier = "full"
lsp  = "vtsls"
```

### Error behaviour

If you specify a formatter or linter in `ferry.lock` that is not in the built-in registry **and** not in `ferry-tools.yaml`, `ferry bundle` will fail with a clear message:

```
formatter "biome" for language "typescript" is not in the built-in registry;
add it to ferry-tools.yaml under formatters:
```

Tools from the language's default list (e.g. `gofmt` for Go, which ships with the runtime) are always available without a `build` entry.

---

## Configuration (`ferry.lock`)

Ferry reads `ferry.lock` (TOML) from the current directory. Run `ferry init` to create one interactively, or write it by hand.

### Complete annotated example

```toml
# Neovim version to bundle.
[nvim]
version = "0.10.4"

# Shell environment to bundle (optional — only matters if include_shell = true in a profile).
[shell]
type = "zsh"
config = "~/.zshrc"                    # your local shell config to copy
theme = "p10k"                         # p10k | starship | oh-my-zsh
theme_config = "~/.p10k.zsh"
plugin_manager = "zinit"
plugin_manager_path = "~/.local/share/zinit"

# Bundle settings.
[bundle]
compress = "zstd"
exclude = [
  "lazy/*/test/",
  "lazy/*/.git/",
  "lazy/*/doc/",
  "lazy/nvim-treesitter/parser/*.so",  # parsers bundled separately
]

[bundle.requires]
packages       = ["git", "curl", "tmux", "zsh"]
min_zsh_version = "5.8"

# Profiles — each is independently deployable.
[profiles.default]
description   = "full dev environment"
include_shell = true
plugins = [
  "nvim-treesitter/nvim-treesitter",
  "neovim/nvim-lspconfig",
  "williamboman/mason.nvim",
  "williamboman/mason-lspconfig.nvim",
  "hrsh7th/nvim-cmp",
  "nvim-telescope/telescope.nvim",
]
cli = ["rg", "fzf", "zoxide", "jq"]

[[profiles.default.languages]]
name = "python"
tier = "full"
runtime_version = "3.12"              # override default version

[[profiles.default.languages]]
name = "go"
tier = "full"

[[profiles.default.languages]]
name = "typescript"
tier = "full"
extra_packages = ["prettier"]         # installed via npm after runtime

[[profiles.default.languages]]
name = "rust"
tier = "lsp-only"                     # LSP only, no full toolchain

[[profiles.default.languages]]
name = "python"
tier = "full"
lsp = "pylsp"                         # override default LSP (pyright)
formatters = ["black", "isort"]       # override default formatters
linters = ["mypy"]                    # override default linters

[profiles.minimal]
description = "nvim + shell only"
include_shell = true

[profiles.server]
description   = "shell and CLI tools, lightweight nvim"
include_shell = true
cli = ["rg", "fzf", "jq"]
```

### Language tiers

| Tier | What's included | Use when |
|---|---|---|
| `full` | Complete runtime + LSP + all formatters/linters | Active development on target |
| `lsp-only` | LSP binary only, no full runtime | IDE features only, smaller bundle |

### Language configuration fields

| Field | Default | Description |
|---|---|---|
| `name` | required | Language identifier (see supported languages below) |
| `tier` | `full` | `full` or `lsp-only` |
| `lsp` | registry default | Override the default LSP server |
| `runtime_version` | registry default | Override the runtime version |
| `formatters` | registry default | Override the formatter list |
| `linters` | registry default | Override the linter list |
| `extra_packages` | `[]` | Additional packages installed via the language's package manager after the runtime (e.g. pip, npm, cargo) |

---

## Supported languages

| Language | Default LSP | Alternate LSPs | Default version | macOS |
|---|---|---|---|---|
| Python | pyright | pylsp | 3.12 | ✓ |
| JavaScript | typescript-language-server | — | Node 22 | ✓ |
| TypeScript | typescript-language-server | — | Node 22 | ✓ |
| Go | gopls | — | 1.22.5 | ✓ |
| Rust | rust-analyzer | — | stable | ✓ |
| Ruby | ruby-lsp | — | 3.3 | ✓ |
| Java | jdtls | — | JDK 21 | ✓ |
| C | clangd | — | LLVM 17 | ✓ |
| C++ | clangd | — | LLVM 17 | ✓ |
| C# | csharp-ls | — | .NET 8.0 | ✓ |
| PHP | intelephense | — | 8.3 | ✓ |
| Kotlin | kotlin-language-server | — | 2.0 | — |
| Lua | lua-language-server | — | 3.18 | ✓ |
| Bash | bash-language-server | — | Node 22 | ✓ |
| Elixir | elixir-ls | — | 0.21 | ✓ |
| Zig | zls | — | 0.14 | ✓ |
| YAML | yaml-language-server | — | Node 22 | ✓ |
| JSON | vscode-json-language-server | — | Node 22 | ✓ |
| Markdown | marksman | — | Node 22 | ✓ |
| Dockerfile | dockerfile-language-server | — | Node 22 | ✓ |
| HTML | vscode-html-language-server | — | Node 22 | ✓ |
| CSS | vscode-css-language-server | — | Node 22 | ✓ |
| SQL | sqls | — | 0.2.28 | ✓ |

Languages without their own runtime (YAML, JSON, Markdown, Dockerfile, Bash, HTML, CSS) share the Node.js nvm runtime. SQL uses a self-contained Go binary with no runtime dependency.

JavaScript, TypeScript, and all Node-based tool languages share a single nvm Node runtime — if multiple are included in a profile, Node is only bundled once.

C and C++ share the same LLVM toolchain — selecting both bundles LLVM once and deduplicates automatically.

---

## Neovim integration

Ferry writes two Lua files to the target's neovim config during `bootstrap`. They handle the fact that the target has no internet and may have only a subset of the tools present in your host config.

### `~/.config/nvim/plugin/ferry.lua`

Sourced automatically before lazy.nvim and plugin configs run. Contains seven hardened patches that apply regardless of what the user's config does:

**1. LSP executable guard**

Wraps `vim.lsp.start` (nvim 0.9+) and `vim.lsp.start_client` (older API) to check `vim.fn.executable()` before spawning a server. LSP servers not present in PATH are silently skipped with no error. No changes to your neovim config required.

**2. Mason neutralisation**

On a no-internet target, mason must not attempt to download packages. Two `package.preload` hooks intercept `mason-registry` and `mason-core.package` regardless of when mason loads:
- `mason-registry.refresh` → silent no-op (stops all registry network activity and `ensure_installed` install jobs)
- `mason-core.package.Package.install` → silent no-op (stops individual tool downloads)

This covers everything built on mason: `mason-lspconfig`, `mason-nvim-dap`, `mason-conform`, `mason-tool-installer`, etc.

**3. nvim-treesitter download neutralisation**

Patches `nvim-treesitter.install` via preload hook to no-op all install and update entry points. Treesitter parsers are pre-bundled by ferry and present on disk; `ensure_installed` and `auto_install` are intercepted before they attempt any downloads. Also covers the async job scheduler introduced in nvim-treesitter 0.12.

**4. lazy.nvim update checker neutralisation**

Patches `lazy.manage.checker` to disable the background `git fetch` that checks for plugin updates. All plugins are pre-bundled; no git operations should run on a no-internet target.

**5. vim.pack neutralisation (Neovim 0.12+)**

`vim.pack` is Neovim 0.12's built-in package manager. Its `update()` function is replaced with a no-op at startup. `add()` is left intact because for already-installed plugins it only extends `runtimepath` — no network calls occur unless a plugin directory is absent.

**6. mini.deps neutralisation**

Patches the `mini.deps` module (mini.nvim's package manager) via preload hook. `update()` is replaced with a no-op to prevent git-fetch operations. `add()` is left intact for the same reason as vim.pack.

**7. LSP format-on-save fallback**

Registers a `BufWritePre` autocmd that calls `vim.lsp.buf.format()` whenever conform.nvim is not loaded. Formatting works out of the box with zero plugin config — the fallback steps aside automatically when `require('ferry').setup_conform()` is called.

### `~/.config/nvim/lua/ferry.lua`

Optional integration module. The `plugin/ferry.lua` patches above work with zero config changes. Use `lua/ferry.lua` when you want explicit ferry-aware setup — for example, to replace an `ensure_installed` list with exactly what's bundled.

```lua
local ferry = require('ferry')

-- ferry.servers: list of LSP servers bundled for this target
-- ferry.parsers: list of treesitter parsers bundled for this target
```

**`setup_lazy(plugins, opts?)`** — wraps `require('lazy').setup()` with safe defaults:

```lua
-- In your init.lua, replace:
require('lazy').setup(plugins, opts)
-- With:
require('ferry').setup_lazy(plugins, opts)
-- Adds: checker.enabled = false, install.missing = false
```

**`setup_mason(server_config?)`** — configures mason-lspconfig to never install anything, then calls `lspconfig.setup()` for each bundled server:

```lua
-- Replaces your mason-lspconfig.setup() + lspconfig loop:
require('ferry').setup_mason({
  on_attach    = my_on_attach,
  capabilities = my_capabilities,
})
```

**`setup_treesitter(ts_config?)`** — wraps `nvim-treesitter.configs.setup()` with `auto_install = false` and `ensure_installed = {}`:

```lua
-- Replaces your nvim-treesitter.configs.setup() call:
require('ferry').setup_treesitter({
  highlight = { enable = true },
  indent    = { enable = true },
})
```

**`setup_conform(opts?)`** — configures conform.nvim with ferry's bundled formatters and wires `BufWritePre` format-on-save. Only fills in filetypes not already present in `opts.formatters_by_ft`, so user overrides are preserved. Uses `lsp_format = "fallback"` (conform v8+, with `lsp_fallback = true` as a compatibility alias for older versions) so filetypes without a dedicated formatter still format via their LSP:

```lua
-- Replaces your conform.setup() call:
require('ferry').setup_conform({
  -- any extra conform opts
})
```

**`setup_lint()`** — configures nvim-lint with ferry's bundled linters and wires `BufWritePost`/`BufReadPost` autocmds. Only fills in filetypes not already configured:

```lua
require('ferry').setup_lint()
```

### What's covered automatically vs. what uses helpers

| Plugin | Auto (plugin/ferry.lua) | Helper (lua/ferry.lua) |
|---|---|---|
| vim.lsp | Executable guard (silent skip) | — |
| mason | Install/download no-oped | `setup_mason()` for explicit server setup |
| nvim-treesitter | Download/update no-oped | `setup_treesitter()` for clean config |
| lazy.nvim | Checker no-oped | `setup_lazy()` for `install.missing = false` |
| vim.pack (nvim 0.12+) | `update()` no-oped | — |
| mini.deps | `update()` no-oped | — |
| conform.nvim | LSP fallback format-on-save | `setup_conform()` for full formatter config |
| nvim-lint | — | `setup_lint()` for linter config |

---

## Where ferry stores things

**On your machine (`~/.ferry/`):**

```
~/.ferry/
  bundles/        manifests: {profile}-{arch}-{libc}-{hash}.json
  store/          component archives: {sha256}.tar.zst (content-addressed)
  targets.json    known targets with arch, libc, profile, last update time
  logs/           build logs: bundle-{arch}-{os}.log
```

**On the target (`~/`):**

```
~/.ferry/
  manifest.json   active component manifest
  store/          uploaded component archives (cleared by ferry clean)
  incoming/       staging area during install/update
  .update.lock    atomic update lock (POSIX noclobber)

~/.local/bin/           nvim symlink and ferry-installed CLI tools
~/.local/share/nvim-dist/   neovim binary
~/.local/share/nvim/    neovim data directory (plugins, parser cache)
~/.config/nvim/         neovim config (not managed by ferry — yours)
~/.config/nvim/plugin/ferry.lua   written by ferry bootstrap
~/.config/nvim/lua/ferry.lua      written by ferry bootstrap
~/.ferry/runtimes/      language runtimes: python-3.12/, go-1.22.5/, etc.
```

---

## Architecture notes

**No version managers on target.** pyenv, nvm, rustup, rbenv, and sdkman are Docker-only build tools. Ferry extracts the versioned runtime binaries directly to `~/.ferry/runtimes/{lang}-{version}/` and adds them to PATH via `~/.zshrc`. The target gets a working runtime with no version manager overhead.

**Content-addressed component store.** Each component is a `tar.zst` archive named by the SHA256 of its contents. The same Python 3.12 runtime built twice produces the same hash — it's stored and shipped once, reused across all profiles and targets.

**POSIX shell install script.** `install.sh` is plain `sh` (not bash), uses only POSIX-standard commands, and works on minimal targets including Alpine/BusyBox. The age decryption key is delivered via stdin, never as a command-line argument.

**Deduplication for shared runtimes.** Languages sharing a runtime manager (JavaScript and TypeScript both use nvm, for example) only emit Docker `BuildSteps` once. Subsequent languages use `AddonBuildSteps` to install their tools into the shared runtime without rebuilding it.

**Atomic updates.** The update lock uses POSIX `set -C` (noclobber), which is atomic on all POSIX filesystems. The active manifest (`~/.ferry/manifest.json`) is only replaced after all components are successfully extracted.

---

## Troubleshooting

**`ferry bundle` fails with Docker errors**

Check that Docker is running (`docker ps`) and that you have permission to use it. On Linux, your user may need to be in the `docker` group.

**`ferry bootstrap` fails with "no bundle found"**

You need to run `ferry bundle` before bootstrapping. The bundle must match the target's architecture and libc. If the target is x86_64 Linux, run `ferry bundle --arch x86_64 --os linux` first.

**`ferry bootstrap` fails disk space check**

The check requires ~2× the bundle size. Either free disk space on the target or reduce the bundle by switching some languages to `lsp-only` tier.

**LSP servers not starting in neovim**

`plugin/ferry.lua` silently skips LSP servers whose executables are not in PATH — it should not produce errors. If a server you expect is not starting, verify it was included in the bundle (`ferry ls` shows the profile; check which languages were bundled). Also check that the runtime PATH entries were added to `~/.zshrc` by looking for `export PATH="$HOME/.ferry/runtimes/..."` lines.

**mason trying to download on the target**

If mason is configured with `ensure_installed`, the `plugin/ferry.lua` hooks neutralise all download attempts automatically. If you still see mason errors, use `require('ferry').setup_mason(...)` as a direct replacement for your `mason-lspconfig.setup()` call.

**nvim-treesitter errors about missing parsers**

Parsers are pre-bundled to `~/.local/share/nvim/lazy/nvim-treesitter/parser/`. If treesitter reports parsers as missing, verify the `treesitter/parsers` component was included in the bundle (`ferry ls`). The `plugin/ferry.lua` hooks prevent auto-install attempts but do not create parsers that weren't bundled.

**Stale update lock**

If a previous update was interrupted, the lock may remain:

```bash
ssh ubuntu@my-vm "rm ~/.ferry/.update.lock"
```

**SSH says target is not in known_hosts**

Add the host key first:

```bash
ssh-keyscan -H ubuntu@my-vm >> ~/.ssh/known_hosts
```
