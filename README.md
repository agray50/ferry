# ferry

Ship your dev environment to remote machines.

Ferry bundles your neovim config, shell setup, CLI tools, and programming language runtimes into a portable archive, then deploys it to remote VMs over SSH — no internet access required on the target.

## How it works

1. **`ferry init`** — interactive wizard to configure one or more profiles (language runtimes, plugins, CLI tools)
2. **`ferry bundle`** — build component archives using Docker (Linux) or direct downloads (macOS)
3. **`ferry bootstrap <user@host>`** — upload and install the bundle on a remote machine
4. **`ferry update <user@host>`** — send delta updates when your config changes

## Features

- **Full runtime bundling** — Python, Node, Go, Rust, Ruby, Java, and 20+ more languages, with LSPs, formatters, and linters included
- **No internet on target** — everything ships in the bundle; the remote VM can be airgapped
- **Profile-based** — multiple independent profiles per bundle (e.g. `backend`, `frontend`, `data`)
- **Content-addressed store** — same runtime at the same version is built once and reused across profiles
- **SSH config integration** — reads and writes `~/.ssh/config`; ferry-managed hosts are tagged for easy identification
- **macOS support** — darwin/amd64 and darwin/arm64 tracks use direct binary downloads instead of Docker
- **Encrypted config files** — sensitive dotfiles are encrypted with age before bundling

## Installation

```bash
go install github.com/agray50/ferry@latest
```

Or build from source:

```bash
git clone https://github.com/agray50/ferry.git
cd ferry
go build -o ferry .
```

## Quick start

```bash
# Configure a profile interactively
ferry init

# Build bundles for all profiles
ferry bundle

# Deploy to a remote machine
ferry bootstrap ubuntu@my-vm

# Check deployment status
ferry ls

# Update after changing your config
ferry update ubuntu@my-vm
```

## Commands

| Command | Description |
|---|---|
| `ferry init` | Interactive profile manager — create, edit, delete profiles |
| `ferry init --profile <name>` | Jump directly to the wizard for a specific profile |
| `ferry bundle` | Build component bundles using Docker |
| `ferry bundle --profile <name>` | Build bundles for a specific profile only |
| `ferry bootstrap <user@host>` | Upload and install the bundle on a remote machine |
| `ferry update <user@host>` | Send delta updates to a bootstrapped target |
| `ferry ls` | List profiles and their deployment status |
| `ferry ls --interactive` | Open the interactive profile manager |
| `ferry connect <host>` | SSH into a target and attach to a tmux session |
| `ferry clean <host>` | Remove ferry installation from a target |

## Supported languages

Python, JavaScript, TypeScript, Go, Rust, Ruby, Java, C, C++, Lua, PHP, Swift, Kotlin, Scala, Haskell, Elixir, R, Zig, Dart, Julia, OCaml, Perl, Nim, Crystal, D

Each language can be included as:
- **full** — complete runtime + LSP + formatters + linters
- **lsp-only** — LSP server only (smaller bundle, no runtime on target)

## Configuration

Ferry stores configuration in `ferry.lock` (TOML) in the current directory:

```toml
[profiles.default]
description = "default profile"

[[profiles.default.languages]]
name = "go"
tier = "full"

[[profiles.default.languages]]
name = "python"
tier = "full"
runtime_version = "3.12"

[[profiles.default.languages]]
name = "rust"
tier = "lsp-only"
```

## Requirements

- Go 1.21+
- Docker (for building Linux bundles)
- SSH access to target machines
