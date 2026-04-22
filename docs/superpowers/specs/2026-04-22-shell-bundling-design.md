# Shell Bundling & Nvim Opt-out Design

**Goal:** Make nvim optional per-profile and add first-class shell bundling (framework + dotfiles + prompt theme) so a user who only wants oh-my-zsh + p10k (or starship) on a no-internet target gets a fully working shell with no manual steps.

**Architecture:** Shell follows the exact same component model as nvim plugins — framework directories and dotfiles are archived, content-addressed, and restored by the install script. Nvim is gated behind a per-profile `include_nvim` flag (default true). Shell bundling is driven by a nullable `ShellProfile` struct on `ProfileConfig`. Everything is opt-in and independently switchable.

**Tech Stack:** Go, existing `store.Component` / `ExtractComponents` pipeline, TOML lock file, POSIX sh install script.

**Scope:** zsh only. Zsh binary remains a host prerequisite (already in `BundleRequires.Packages`). Starship flows through the CLI tool path; only its config file is a shell component.

---

## Section 1: Config schema

### `ProfileConfig` changes (`internal/config/lock.go`)

- Remove `IncludeShell bool` (dead field).
- Add `IncludeNvim *bool` — pointer so nil (field absent in TOML) = unset = true. `DefaultLockFile` writes `true` explicitly on all preset profiles. A helper `func (p ProfileConfig) NvimEnabled() bool` returns `p.IncludeNvim == nil || *p.IncludeNvim`.
- Add `Shell *ShellProfile` — nil means no shell bundling.

```toml
[profiles.default]
include_nvim = true

[profiles.default.shell]
type             = "zsh"
framework        = "oh-my-zsh"          # "oh-my-zsh" | "zinit" | "zplug" | ""
framework_path   = "~/.oh-my-zsh"
rc_path          = "~/.zshrc"
theme            = "p10k"               # "p10k" | "starship" | "pure" | ""
theme_config_path = "~/.p10k.zsh"       # or ~/.config/starship.toml
```

### New `ShellProfile` struct

```go
type ShellProfile struct {
    Type            string `toml:"type"`              // "zsh"
    Framework       string `toml:"framework"`         // "oh-my-zsh" | "zinit" | "zplug" | ""
    FrameworkPath   string `toml:"framework_path"`
    RCPath          string `toml:"rc_path"`
    Theme           string `toml:"theme"`             // "p10k" | "starship" | "pure" | ""
    ThemeConfigPath string `toml:"theme_config_path"`
}
```

`IncludeShell bool` and `ShellConfig` are removed. `DefaultLockFile` sets `include_nvim = true` explicitly on all preset profiles.

---

## Section 2: Discovery (`internal/discovery/shell.go`)

Extend `DiscoverShell()` to also detect `pure` theme (`~/.zsh/pure/` or via zinit path). Return type stays `ShellDiscovery` — no change to callers. Add:

```go
case fileExists(filepath.Join(home, ".zsh", "pure", "pure.zsh")):
    d.ThemeDetected = "pure"
    d.ThemeConfigPath = ""  // pure has no separate config file
```

Add a method `ToShellProfile() *ShellProfile` on `ShellDiscovery` that maps detected fields to the lock file struct, returning nil if no shell was detected.

---

## Section 3: Bundle extraction (`internal/bundle/extract.go`)

### Nvim opt-out

`buildComponentSpecs` receives the `ProfileConfig`. The nvim block becomes conditional:

```go
if prof.NvimEnabled() {
    specs = append(specs,
        componentSpec{id: "nvim-binary", ...},
        componentSpec{id: "treesitter/parsers", ...},
    )
    for _, plugin := range prof.Plugins {
        specs = append(specs, componentSpec{id: "lazy/" + plugin, ...})
    }
}
```

### Shell components

When `prof.Shell != nil`:

```go
sp := prof.Shell
if sp.Framework != "" {
    specs = append(specs, componentSpec{
        id:            "shell/framework",
        containerPath: sp.FrameworkPath + "/",
        installPath:   sp.FrameworkPath + "/",
        localPath:     config.ExpandHome(sp.FrameworkPath),  // macOS: copy from local
    })
}
if sp.RCPath != "" {
    specs = append(specs, componentSpec{
        id:          "shell/rc",
        containerPath: sp.RCPath,
        installPath:   sp.RCPath,
    })
}
if sp.ThemeConfigPath != "" {
    specs = append(specs, componentSpec{
        id:          "shell/theme-config",
        containerPath: sp.ThemeConfigPath,
        installPath:   sp.ThemeConfigPath,
    })
}
```

`componentSpec` gains a `localPath string` field used by the macOS track, and `preserve bool` which maps to `store.Component.Preserve` — when true the install script skips restore if the target file already exists (dotfile non-destructive behaviour).

Shell `rc` and `theme-config` specs set `preserve: true`. The framework dir does not — it always gets the atomic swap.

### macOS extraction

`buildMacOSTrack` checks `spec.localPath`: if set, copies from local filesystem instead of extracting from container. The nvim build call is gated on `prof.NvimEnabled()`.

---

## Section 4: Dockerfile generation (`internal/bundle/dockerfile.go`)

`GenerateDockerfile` already receives `lock *config.LockFile` and `profile string`. The nvim download/install block is wrapped using the helper:

```go
prof := lock.Profiles[profile]
if prof.NvimEnabled() {
    // existing nvim RUN steps
}
```

No shell framework is built in Docker — shell components come from the local machine via `localPath`.

---

## Section 5: Install script (`internal/bootstrap/install.go`)

`GenerateInstallScript` signature gains `shellProf *config.ShellProfile`.

The nvim lua setup call is gated:

```go
if lock.Profiles[profile].IncludeNvim {
    b.WriteString(generateNvimLuaSetup(langs))
}
```

Shell components flow through the **existing component loop** in the install script. No separate shell restore block is needed. The loop already calls `install_component` (dir) or `install_file` (single file) based on the trailing slash of `installPath`.

To support non-destructive dotfile install, `store.Component` gains a `Preserve bool` field. The install script checks it:

```sh
install_file_preserve() {
  local hash="$1" dest="$2"
  [ -e "$dest" ] && return 0
  install_file "$hash" "$dest"
}
```

When `Preserve = true` the script calls `install_file_preserve` instead of `install_file`. Shell `rc` and `theme-config` components have `Preserve = true`; framework dir does not.

The nvim lua setup call is gated using the profile helper, with the profile name threaded through from the caller:

```go
if lock.Profiles[profile].NvimEnabled() {
    b.WriteString(generateNvimLuaSetup(langs))
}
```

---

## Section 6: Init wizard (`internal/tui/init_wizard.go`)

### Step 2: Neovim (modified)

Before the plugin list, a single toggle item: `◉ Include Neovim`. If deselected, the plugin sub-list is hidden and `include_nvim = false` is written to the profile.

### Step 4: Shell (new, replaces silent stub)

Uses `MultiSelectModel` with items built from `ShellDiscovery`:

```
Step 4: Shell

  detected: zsh  │  oh-my-zsh  │  p10k

  ❯ ◉ Bundle shell framework (~/.oh-my-zsh)   ~45MB
    ◉ Bundle .zshrc
    ◉ Bundle theme config (~/.p10k.zsh)
```

For a starship user only `.zshrc` and `~/.config/starship.toml` appear (no framework line). If nothing is detected the step is skipped silently. Result written to `prof.Shell`.

---

## Error behaviour

- Shell framework dir not found at `localPath` at bundle time → warning + skip (same as any missing component), bundle continues without shell components.
- `include_nvim = false` with nvim plugins still listed in profile → plugins are ignored silently (no error).
- `theme_config_path` set but file missing → warning + skip.

---

## What is NOT in scope

- Bash support (zsh only).
- Shipping the zsh binary (remains a host prerequisite in `BundleRequires.Packages`).
- Starship binary bundling via shell path (goes through CLI tools + `ferry-tools.yaml`).
- Bundling `~/.zsh_history` or other user state files.
- Multi-framework installs (only one framework per profile).
