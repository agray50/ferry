# Shell Bundling & Nvim Opt-out Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make nvim optional per-profile and add first-class shell bundling (zsh framework + dotfiles + prompt theme) so a user who only wants oh-my-zsh + p10k, zinit + starship, or any other zsh setup gets a fully working shell on a no-internet target with no manual steps.

**Architecture:** Shell follows the exact same component model as nvim plugins — framework directories and dotfiles are archived by `store.CompressDir`, content-addressed by `store.HashBytes`, restored by the install script. Nvim is gated by a per-profile `IncludeNvim *bool` flag (nil = unset = true, backward compatible). Shell bundling is driven by a new `ShellProfile` struct on `ProfileConfig`. A new `Preserve bool` field on `store.Component` gives the install script non-destructive dotfile restore semantics. Local shell assets (framework dir, dotfiles) are copied from the local filesystem via `github.com/otiai10/copy`, which is already in go.mod.

**Tech Stack:** Go 1.26, `github.com/otiai10/copy`, existing `store.CompressDir`/`store.HashBytes` pipeline, TOML lock file, POSIX sh install script, `charmbracelet/bubbletea` TUI.

---

## File map

| File | Change |
|---|---|
| `internal/config/lock.go` | Add `ShellProfile`, replace `IncludeShell`/`ShellConfig` with `IncludeNvim *bool` + `Shell *ShellProfile`, add `NvimEnabled()` + `boolPtr` |
| `internal/config/lock_test.go` | Remove `IncludeShell`, add `IncludeNvim`/`Shell` round-trip tests |
| `internal/discovery/shell.go` | Add `pure` theme detection |
| `internal/store/store.go` | Add `Preserve bool` to `Component` |
| `internal/bootstrap/install.go` | Add `install_file_preserve` shell fn, use `Preserve`, gate nvim lua setup on `NvimEnabled()` |
| `internal/bootstrap/install_test.go` | Add tests for nvim-disabled script + shell preserve |
| `internal/bundle/extract.go` | Add `localPath`/`preserve` to `componentSpec`, gate nvim, add shell specs, add `copyLocalToTmp` |
| `internal/bundle/dockerfile.go` | Gate nvim download/lazy/treesitter blocks on `NvimEnabled()` |
| `internal/bundle/dockerfile_test.go` | Add test that nvim-disabled profile omits nvim from Dockerfile |
| `internal/bundle/macos.go` | Gate `buildNvimMacOS`, add section 5 for local shell components |
| `internal/tui/init_wizard.go` | Replace silent shell stub with real Step 4; add nvim toggle to Step 2; add `shellDiscoveryToProfile` helper |

---

### Task 1: Config schema — `ShellProfile`, `IncludeNvim *bool`, remove dead fields

**Files:**
- Modify: `internal/config/lock.go`
- Modify: `internal/config/lock_test.go`

- [ ] **Step 1: Write failing tests**

Add to `internal/config/lock_test.go`:

```go
func TestNvimEnabled(t *testing.T) {
	trueVal := true
	falseVal := false

	// nil pointer = unset = enabled (backward compat)
	p := config.ProfileConfig{}
	if !p.NvimEnabled() {
		t.Error("nil IncludeNvim should mean nvim is enabled")
	}

	// explicit true
	p.IncludeNvim = &trueVal
	if !p.NvimEnabled() {
		t.Error("IncludeNvim=true should mean nvim is enabled")
	}

	// explicit false
	p.IncludeNvim = &falseVal
	if p.NvimEnabled() {
		t.Error("IncludeNvim=false should mean nvim is disabled")
	}
}

func TestShellProfileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	trueVal := true
	lf := &config.LockFile{
		Profiles: map[string]config.ProfileConfig{
			"shell-only": {
				Description: "shell only profile",
				IncludeNvim: &trueVal,
				Shell: &config.ShellProfile{
					Type:            "zsh",
					Framework:       "oh-my-zsh",
					FrameworkPath:   "~/.oh-my-zsh",
					RCPath:          "~/.zshrc",
					Theme:           "p10k",
					ThemeConfigPath: "~/.p10k.zsh",
				},
			},
		},
	}

	if err := config.WriteLockFile(lf); err != nil {
		t.Fatalf("WriteLockFile: %v", err)
	}
	got, err := config.ReadLockFile()
	if err != nil {
		t.Fatalf("ReadLockFile: %v", err)
	}
	prof := got.Profiles["shell-only"]
	if prof.Shell == nil {
		t.Fatal("Shell should not be nil after round-trip")
	}
	if prof.Shell.Framework != "oh-my-zsh" {
		t.Errorf("Shell.Framework = %q, want oh-my-zsh", prof.Shell.Framework)
	}
	if prof.Shell.Theme != "p10k" {
		t.Errorf("Shell.Theme = %q, want p10k", prof.Shell.Theme)
	}
	if prof.IncludeNvim == nil || !*prof.IncludeNvim {
		t.Error("IncludeNvim should be true after round-trip")
	}
}

func TestDefaultLockFileNvimEnabled(t *testing.T) {
	lf := config.DefaultLockFile()
	for name, prof := range lf.Profiles {
		if !prof.NvimEnabled() {
			t.Errorf("profile %q: DefaultLockFile should have nvim enabled", name)
		}
	}
}
```

- [ ] **Step 2: Run failing tests**

```bash
cd /home/claude/ferry && go test ./internal/config/... -run "TestNvimEnabled|TestShellProfileRoundTrip|TestDefaultLockFileNvimEnabled" -v
```

Expected: FAIL — `NvimEnabled`, `ShellProfile`, `IncludeNvim` undefined.

- [ ] **Step 3: Implement config changes**

Replace the body of `internal/config/lock.go` with the following (keep the existing `ReadLockFile`, `WriteLockFile`, `LockFileHash` functions unchanged; only the type definitions and `DefaultLockFile` change):

**Replace `ProfileConfig` struct:**
```go
// ProfileConfig describes a complete, independently deployable environment.
type ProfileConfig struct {
	Description string           `toml:"description"`
	Languages   []LanguageConfig `toml:"languages"`
	Plugins     []string         `toml:"plugins"`
	CLI         []string         `toml:"cli"`
	IncludeNvim *bool            `toml:"include_nvim,omitempty"`
	Shell       *ShellProfile    `toml:"shell,omitempty"`
}

// NvimEnabled reports whether nvim should be bundled for this profile.
// A nil pointer (field absent from ferry.lock) means true for backward compatibility.
func (p ProfileConfig) NvimEnabled() bool {
	return p.IncludeNvim == nil || *p.IncludeNvim
}
```

**Add `ShellProfile` struct** (after `ProfileConfig`):
```go
// ShellProfile describes the zsh setup to bundle for a profile.
// nil means no shell bundling.
type ShellProfile struct {
	Type            string `toml:"type"`             // always "zsh"
	Framework       string `toml:"framework"`        // "oh-my-zsh" | "zinit" | "zplug" | ""
	FrameworkPath   string `toml:"framework_path"`
	RCPath          string `toml:"rc_path"`
	Theme           string `toml:"theme"`            // "p10k" | "starship" | "pure" | ""
	ThemeConfigPath string `toml:"theme_config_path"`
}
```

**Remove `ShellConfig` struct entirely** (it was dead code).

**Remove `IncludeShell bool` from `ProfileConfig`** (replaced by `Shell *ShellProfile`).

**Add `boolPtr` helper** at the bottom of the file:
```go
func boolPtr(b bool) *bool { return &b }
```

**Update `DefaultLockFile`** — remove `IncludeShell: true` from all profiles, add `IncludeNvim: boolPtr(true)`:
```go
func DefaultLockFile() *LockFile {
	return &LockFile{
		Nvim: NvimConfig{Version: "0.10.4"},
		Bundle: BundleConfig{
			Compress: "zstd",
			Exclude: []string{
				"lazy/*/test/",
				"lazy/*/.git/",
				"lazy/*/doc/",
				"lazy/nvim-treesitter/parser/*.so",
			},
			Requires: BundleRequires{
				Packages:      []string{"git", "curl", "tmux", "zsh"},
				MinZshVersion: "5.8",
			},
		},
		Profiles: map[string]ProfileConfig{
			"default": {
				Description:  "full dev environment",
				Languages:    []LanguageConfig{},
				IncludeNvim:  boolPtr(true),
				CLI:          []string{"rg", "fzf", "zoxide", "jq"},
			},
			"minimal": {
				Description: "nvim + shell only",
				IncludeNvim: boolPtr(true),
			},
			"server": {
				Description: "shell and CLI tools, lightweight nvim",
				IncludeNvim: boolPtr(true),
				CLI:         []string{"rg", "fzf", "jq"},
			},
		},
	}
}
```

**Update `lock_test.go`** — remove `IncludeShell: true` from `TestLanguageConfigRoundTrip`:
```go
"backend": {
    Description: "backend profile",
    Languages: []LanguageConfig{
        {Name: "go", Tier: "full", RuntimeVersion: "1.22.5"},
        {Name: "python", Tier: "lsp-only", LSP: "pylsp"},
    },
    Plugins: []string{"nvim-lspconfig", "telescope.nvim"},
    CLI:     []string{"rg", "fzf"},
},
```

- [ ] **Step 4: Run tests**

```bash
cd /home/claude/ferry && go test ./internal/config/... -v
```

Expected: all PASS (including the 3 new tests).

- [ ] **Step 5: Verify compilation**

```bash
cd /home/claude/ferry && go build ./...
```

Expected: compile errors because `IncludeShell` is still referenced in `tui/init_wizard.go`. Note them — they will be fixed in Task 7.

- [ ] **Step 6: Commit**

```bash
git add internal/config/lock.go internal/config/lock_test.go
git commit -m "feat(config): ShellProfile struct, IncludeNvim *bool, remove dead ShellConfig"
```

---

### Task 2: Discovery — add `pure` theme detection

**Files:**
- Modify: `internal/discovery/shell.go`

- [ ] **Step 1: Write failing test**

Add to `internal/discovery/shell.go` a new file `internal/discovery/shell_test.go`:

```go
package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverShellPure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	pureDir := filepath.Join(home, ".zsh", "pure")
	os.MkdirAll(pureDir, 0755)
	os.WriteFile(filepath.Join(pureDir, "pure.zsh"), []byte("# pure"), 0644)

	d := DiscoverShell()
	if d.ThemeDetected != "pure" {
		t.Errorf("ThemeDetected = %q, want pure", d.ThemeDetected)
	}
	if d.ThemeConfigPath != "" {
		t.Errorf("pure has no config file, ThemeConfigPath should be empty, got %q", d.ThemeConfigPath)
	}
}

func TestDiscoverShellP10k(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	os.WriteFile(filepath.Join(home, ".p10k.zsh"), []byte("# p10k"), 0644)

	d := DiscoverShell()
	if d.ThemeDetected != "p10k" {
		t.Errorf("ThemeDetected = %q, want p10k", d.ThemeDetected)
	}
	if d.ThemeConfigPath != filepath.Join(home, ".p10k.zsh") {
		t.Errorf("ThemeConfigPath = %q, want ~/.p10k.zsh", d.ThemeConfigPath)
	}
}

func TestDiscoverShellStarship(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	starshipDir := filepath.Join(home, ".config")
	os.MkdirAll(starshipDir, 0755)
	os.WriteFile(filepath.Join(starshipDir, "starship.toml"), []byte("# starship"), 0644)

	d := DiscoverShell()
	if d.ThemeDetected != "starship" {
		t.Errorf("ThemeDetected = %q, want starship", d.ThemeDetected)
	}
}
```

- [ ] **Step 2: Run failing test**

```bash
cd /home/claude/ferry && go test ./internal/discovery/... -run "TestDiscoverShellPure" -v
```

Expected: FAIL — `pure` is not detected yet.

- [ ] **Step 3: Add `pure` detection to `DiscoverShell`**

In `internal/discovery/shell.go`, update the theme detection switch to add `pure` before `oh-my-zsh`:

```go
// detect theme
p10k := filepath.Join(home, ".p10k.zsh")
starship := filepath.Join(home, ".config", "starship.toml")
pure := filepath.Join(home, ".zsh", "pure", "pure.zsh")
switch {
case fileExists(p10k):
	d.ThemeDetected = "p10k"
	d.ThemeConfigPath = p10k
case fileExists(starship):
	d.ThemeDetected = "starship"
	d.ThemeConfigPath = starship
case fileExists(pure):
	d.ThemeDetected = "pure"
	d.ThemeConfigPath = ""
case fileExists(filepath.Join(home, ".oh-my-zsh")):
	d.ThemeDetected = "oh-my-zsh"
}
```

- [ ] **Step 4: Run tests**

```bash
cd /home/claude/ferry && go test ./internal/discovery/... -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/discovery/shell.go internal/discovery/shell_test.go
git commit -m "feat(discovery): add pure prompt theme detection"
```

---

### Task 3: `store.Component` — `Preserve` flag + install script non-destructive restore

**Files:**
- Modify: `internal/store/store.go`
- Modify: `internal/bootstrap/install.go`
- Modify: `internal/bootstrap/install_test.go`

- [ ] **Step 1: Write failing tests**

Add to `internal/bootstrap/install_test.go`:

```go
func TestGenerateInstallScriptPreservesDotfiles(t *testing.T) {
	m := &store.Manifest{
		Profile: "shell-only",
		Components: []store.Component{
			{ID: "shell/framework", InstallPath: "~/.oh-my-zsh/", Hash: "abc123"},
			{ID: "shell/rc", InstallPath: "~/.zshrc", Hash: "def456", Preserve: true},
			{ID: "shell/theme-config", InstallPath: "~/.p10k.zsh", Hash: "ghi789", Preserve: true},
		},
	}
	script, err := GenerateInstallScript(m, &config.LockFile{}, nil)
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	if !strings.Contains(script, "install_file_preserve") {
		t.Error("install script should define install_file_preserve for dotfile non-destructive restore")
	}
	// framework dir uses normal install_component (atomic swap, always overwrites)
	if !strings.Contains(script, `install_component "abc123"`) {
		t.Error("shell framework should use install_component")
	}
	// rc and theme-config must use install_file_preserve
	if !strings.Contains(script, `install_file_preserve "def456"`) {
		t.Error("shell/rc should use install_file_preserve")
	}
	if !strings.Contains(script, `install_file_preserve "ghi789"`) {
		t.Error("shell/theme-config should use install_file_preserve")
	}
}

func TestGenerateInstallScriptNvimDisabled(t *testing.T) {
	falseVal := false
	lf := &config.LockFile{
		Profiles: map[string]config.ProfileConfig{
			"shell-only": {
				IncludeNvim: &falseVal,
			},
		},
	}
	m := &store.Manifest{Profile: "shell-only"}
	script, err := GenerateInstallScript(m, lf, nil)
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	if strings.Contains(script, "plugin/ferry.lua") {
		t.Error("nvim-disabled profile should not write plugin/ferry.lua")
	}
	if strings.Contains(script, "lua/ferry.lua") {
		t.Error("nvim-disabled profile should not write lua/ferry.lua")
	}
}
```

- [ ] **Step 2: Run failing tests**

```bash
cd /home/claude/ferry && go test ./internal/bootstrap/... -run "TestGenerateInstallScriptPreservesDotfiles|TestGenerateInstallScriptNvimDisabled" -v
```

Expected: FAIL — `Preserve` field undefined, nvim lua always written.

- [ ] **Step 3: Add `Preserve bool` to `store.Component`**

In `internal/store/store.go`, update the `Component` struct:

```go
// Component is a single installable unit within a manifest.
type Component struct {
	ID             string `json:"id"`
	Version        string `json:"version"`
	Hash           string `json:"hash"`
	SizeCompressed int64  `json:"size_compressed"`
	InstallPath    string `json:"install_path"`
	BinSymlink     string `json:"bin_symlink,omitempty"`
	ArchSpecific   bool   `json:"arch_specific"`
	Preserve       bool   `json:"preserve,omitempty"`
}
```

- [ ] **Step 4: Update `GenerateInstallScript` in `internal/bootstrap/install.go`**

**4a.** Add `install_file_preserve` shell function to the preamble (after `install_file`):

```go
b.WriteString(`
# Installs a single-file component only if the destination does not already exist.
# Used for dotfiles so a user's remote customisations are not overwritten on update.
install_file_preserve() {
  local hash="$1" dest="$2"
  [ -e "$dest" ] && return 0
  install_file "$hash" "$dest"
}

`)
```

**4b.** Update the component install loop to call `install_file_preserve` when `c.Preserve`:

```go
for _, c := range m.Components {
	b.WriteString(fmt.Sprintf("# %s\n", c.ID))
	installPath := config.ExpandHome(c.InstallPath)
	if strings.HasSuffix(c.InstallPath, "/") {
		b.WriteString(fmt.Sprintf("install_component %q %q\n", c.Hash, installPath))
	} else if c.Preserve {
		b.WriteString(fmt.Sprintf("install_file_preserve %q %q\n", c.Hash, installPath))
	} else {
		b.WriteString(fmt.Sprintf("install_file %q %q\n", c.Hash, installPath))
	}
	if c.BinSymlink != "" {
		binLink := config.ExpandHome(c.BinSymlink)
		b.WriteString(fmt.Sprintf("ln -sf %q/bin/nvim %q\n", installPath, binLink))
	}
	b.WriteString("\n")
}
```

**4c.** Gate `generateNvimLuaSetup` call on `NvimEnabled()`. The manifest carries `Profile`; look it up in `lock`:

```go
// Gate nvim lua shims on IncludeNvim. When profile is absent from lock
// (e.g. legacy manifests with no ferry.lock), NvimEnabled() returns true.
if lock.Profiles[m.Profile].NvimEnabled() {
	b.WriteString(generateNvimLuaSetup(langs))
}
```

- [ ] **Step 5: Run tests**

```bash
cd /home/claude/ferry && go test ./internal/bootstrap/... -v
```

Expected: all PASS including the 2 new tests.

- [ ] **Step 6: Commit**

```bash
git add internal/store/store.go internal/bootstrap/install.go internal/bootstrap/install_test.go
git commit -m "feat(store,bootstrap): Preserve flag for dotfile non-destructive restore, gate nvim lua on IncludeNvim"
```

---

### Task 4: Bundle extraction — shell component specs + local filesystem copy

**Files:**
- Modify: `internal/bundle/extract.go`

- [ ] **Step 1: Write failing test**

Add to `internal/bundle/extract_test.go`:

```go
func TestBuildComponentSpecsNvimDisabled(t *testing.T) {
	falseVal := false
	lock := &config.LockFile{
		Profiles: map[string]config.ProfileConfig{
			"shell-only": {IncludeNvim: &falseVal},
		},
	}
	track := BuildTrack{Arch: "x86_64", OS: "linux"}
	specs := buildComponentSpecs(track, lock, "shell-only", nil)
	for _, s := range specs {
		if s.id == "nvim-binary" || strings.HasPrefix(s.id, "treesitter") || strings.HasPrefix(s.id, "lazy/") {
			t.Errorf("nvim-disabled profile should not produce spec %q", s.id)
		}
	}
}

func TestBuildComponentSpecsShellComponents(t *testing.T) {
	lock := &config.LockFile{
		Profiles: map[string]config.ProfileConfig{
			"shell-only": {
				Shell: &config.ShellProfile{
					Type:            "zsh",
					Framework:       "oh-my-zsh",
					FrameworkPath:   "~/.oh-my-zsh",
					RCPath:          "~/.zshrc",
					Theme:           "p10k",
					ThemeConfigPath: "~/.p10k.zsh",
				},
			},
		},
	}
	track := BuildTrack{Arch: "x86_64", OS: "linux"}
	specs := buildComponentSpecs(track, lock, "shell-only", nil)

	ids := make(map[string]componentSpec)
	for _, s := range specs {
		ids[s.id] = s
	}

	if _, ok := ids["shell/framework"]; !ok {
		t.Error("expected shell/framework component spec")
	}
	if _, ok := ids["shell/rc"]; !ok {
		t.Error("expected shell/rc component spec")
	}
	if rc := ids["shell/rc"]; !rc.preserve {
		t.Error("shell/rc should have preserve=true")
	}
	if _, ok := ids["shell/theme-config"]; !ok {
		t.Error("expected shell/theme-config component spec")
	}
	if tc := ids["shell/theme-config"]; !tc.preserve {
		t.Error("shell/theme-config should have preserve=true")
	}
}
```

- [ ] **Step 2: Run failing tests**

```bash
cd /home/claude/ferry && go test ./internal/bundle/... -run "TestBuildComponentSpecsNvimDisabled|TestBuildComponentSpecsShellComponents" -v
```

Expected: FAIL — `localPath`, `preserve` undefined on `componentSpec`.

- [ ] **Step 3: Update `componentSpec` struct**

In `internal/bundle/extract.go`, update the struct:

```go
type componentSpec struct {
	id            string
	containerPath string
	installPath   string
	binSymlink    string
	version       string
	localPath     string // if set, copy from local filesystem instead of docker cp
	preserve      bool   // if true, install script skips restore when dest already exists
}
```

- [ ] **Step 4: Update `buildComponentSpecs` — gate nvim + add shell specs**

Replace the body of `buildComponentSpecs`:

```go
func buildComponentSpecs(track BuildTrack, lock *config.LockFile, profile string, langs []registry.ResolvedLanguage) []componentSpec {
	prof := lock.Profiles[profile]
	var specs []componentSpec

	// Nvim — gated on IncludeNvim (nil = unset = true).
	if prof.NvimEnabled() {
		specs = append(specs,
			componentSpec{id: "nvim-binary", containerPath: "/opt/nvim/", installPath: "~/.local/share/nvim-dist/", binSymlink: "~/.local/bin/nvim"},
			componentSpec{id: "treesitter/parsers", containerPath: "/root/.local/share/nvim/lazy/nvim-treesitter/parser/", installPath: "~/.local/share/nvim/lazy/nvim-treesitter/parser/"},
		)
		for _, plugin := range prof.Plugins {
			specs = append(specs, componentSpec{
				id:            "lazy/" + plugin,
				containerPath: "/root/.local/share/nvim/lazy/" + plugin + "/",
				installPath:   "~/.local/share/nvim/lazy/" + plugin + "/",
			})
		}
	}

	// Shell — framework dir + dotfiles, always from local filesystem.
	if sp := prof.Shell; sp != nil {
		if sp.Framework != "" && sp.FrameworkPath != "" {
			fp := config.ExpandHome(sp.FrameworkPath)
			specs = append(specs, componentSpec{
				id:          "shell/framework",
				installPath: sp.FrameworkPath + "/",
				localPath:   fp,
			})
		}
		if sp.RCPath != "" {
			rc := config.ExpandHome(sp.RCPath)
			specs = append(specs, componentSpec{
				id:          "shell/rc",
				installPath: sp.RCPath,
				localPath:   rc,
				preserve:    true,
			})
		}
		if sp.ThemeConfigPath != "" {
			tc := config.ExpandHome(sp.ThemeConfigPath)
			specs = append(specs, componentSpec{
				id:          "shell/theme-config",
				installPath: sp.ThemeConfigPath,
				localPath:   tc,
				preserve:    true,
			})
		}
	}

	// Language runtimes.
	for _, rl := range langs {
		if rl.Runtime == nil {
			continue
		}
		version := rl.Runtime.DefaultVersion
		for _, cp := range rl.Runtime.ContainerPaths {
			containerPath := substituteVars(cp.Container, version, track.Arch)
			installPath := substituteVars(cp.InstallPath, version, track.Arch)
			trimmed := strings.TrimRight(containerPath, "/")
			base := trimmed[strings.LastIndex(trimmed, "/")+1:]
			specs = append(specs, componentSpec{
				id:            "runtime/" + rl.Language.Name + "/" + base,
				containerPath: containerPath,
				installPath:   installPath,
				version:       version,
			})
		}
	}

	// CLI tools.
	for _, name := range flattenCLI(lock, profile) {
		specs = append(specs, componentSpec{
			id:            "cli/" + name,
			containerPath: "/usr/local/bin/" + name,
			installPath:   "~/.local/bin/" + name,
		})
	}
	return specs
}
```

- [ ] **Step 5: Add `copyLocalToTmp` helper + update `ExtractComponents`**

Add `copyLocalToTmp` at the bottom of `internal/bundle/extract.go` (before `sanitizeID`):

```go
// copyLocalToTmp copies a local file or directory into tmpDir using otiai10/copy.
// The destination filename matches the source base name.
func copyLocalToTmp(src, tmpDir string) error {
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return err
	}
	dst := filepath.Join(tmpDir, filepath.Base(src))
	return cp.Copy(src, dst)
}
```

Add the import `cp "github.com/otiai10/copy"` and `"path/filepath"` to the import block (if not already present).

In `ExtractComponents`, update the extraction loop to handle `spec.localPath`:

```go
for _, spec := range buildComponentSpecs(track, lock, profile, langs) {
	tmpDir := fmt.Sprintf("/tmp/ferry-extract-%s-%s", containerID[:8], sanitizeID(spec.id))

	if spec.localPath != "" {
		// Copy from local filesystem (shell dotfiles/framework, not from container).
		if err := copyLocalToTmp(spec.localPath, tmpDir); err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping component %s: local copy failed: %v\n", spec.id, err)
			os.RemoveAll(tmpDir)
			continue
		}
	} else {
		// Copy from running container.
		isSingleFile := !strings.HasSuffix(spec.containerPath, "/")
		if isSingleFile {
			os.MkdirAll(tmpDir, 0755)
		}
		cpDest := tmpDir
		if isSingleFile {
			cpDest = tmpDir + "/"
		}
		cpCmd := exec.Command("docker", "cp", containerID+":"+spec.containerPath, cpDest)
		if out, err := cpCmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping component %s: docker cp failed: %s\n", spec.id, strings.TrimSpace(string(out)))
			os.RemoveAll(tmpDir)
			continue
		}
	}

	compressed, err := store.CompressDir(tmpDir, lock.Bundle.Exclude)
	os.RemoveAll(tmpDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: skipping component %s: compress failed: %v\n", spec.id, err)
		continue
	}

	hash := store.HashBytes(compressed)
	if err := s.Put(hash, compressed); err != nil {
		return nil, fmt.Errorf("storing component %s: %w", spec.id, err)
	}

	components = append(components, store.Component{
		ID:             spec.id,
		Version:        spec.version,
		Hash:           hash,
		SizeCompressed: int64(len(compressed)),
		InstallPath:    spec.installPath,
		BinSymlink:     spec.binSymlink,
		ArchSpecific:   true,
		Preserve:       spec.preserve,
	})
}
```

- [ ] **Step 6: Run tests**

```bash
cd /home/claude/ferry && go test ./internal/bundle/... -v
```

Expected: all PASS including the 2 new tests.

- [ ] **Step 7: Commit**

```bash
git add internal/bundle/extract.go internal/bundle/extract_test.go
git commit -m "feat(bundle): gate nvim components on NvimEnabled, add shell component extraction from local filesystem"
```

---

### Task 5: Dockerfile — gate nvim block on `NvimEnabled()`

**Files:**
- Modify: `internal/bundle/dockerfile.go`
- Modify: `internal/bundle/dockerfile_test.go`

- [ ] **Step 1: Write failing test**

Add to `internal/bundle/dockerfile_test.go`:

```go
func TestGenerateDockerfileNvimDisabled(t *testing.T) {
	falseVal := false
	track := BuildTracks[0] // linux/x86_64
	lf := &config.LockFile{
		Profiles: map[string]config.ProfileConfig{
			"shell-only": {IncludeNvim: &falseVal},
		},
	}
	df, err := GenerateDockerfile(track, lf, "shell-only", nil, nil)
	if err != nil {
		t.Fatalf("GenerateDockerfile: %v", err)
	}
	if strings.Contains(df, "neovim") || strings.Contains(df, "nvim") {
		t.Error("Dockerfile for nvim-disabled profile should not contain nvim")
	}
	if strings.Contains(df, "Lazy!") {
		t.Error("Dockerfile for nvim-disabled profile should not contain lazy.nvim bootstrap")
	}
	if strings.Contains(df, "TSInstall") {
		t.Error("Dockerfile for nvim-disabled profile should not contain TSInstall")
	}
}

func TestGenerateDockerfileNvimEnabled(t *testing.T) {
	trueVal := true
	track := BuildTracks[0]
	lf := &config.LockFile{
		Profiles: map[string]config.ProfileConfig{
			"default": {IncludeNvim: &trueVal},
		},
	}
	df, err := GenerateDockerfile(track, lf, "default", nil, nil)
	if err != nil {
		t.Fatalf("GenerateDockerfile: %v", err)
	}
	if !strings.Contains(df, "nvim") {
		t.Error("nvim-enabled Dockerfile should contain nvim")
	}
}
```

- [ ] **Step 2: Run failing test**

```bash
cd /home/claude/ferry && go test ./internal/bundle/... -run "TestGenerateDockerfileNvimDisabled" -v
```

Expected: FAIL — nvim is always present regardless of profile.

- [ ] **Step 3: Gate nvim block in `GenerateDockerfile`**

In `internal/bundle/dockerfile.go`, after `dockerArch` is determined and before the nvim download block, get the profile and wrap the nvim section:

```go
prof := lock.Profiles[profile]

if prof.NvimEnabled() {
	nvimVersion := lock.Nvim.Version
	if nvimVersion == "" {
		nvimVersion = "0.10.4"
	}
	nvimArch := "x86_64"
	if track.Arch == "arm64" {
		nvimArch = "arm64"
	}
	tarball := fmt.Sprintf("nvim-linux-%s.tar.gz", nvimArch)
	b.WriteString("# neovim\n")
	b.WriteString(fmt.Sprintf("RUN curl -fsSL https://github.com/neovim/neovim/releases/download/v%s/%s -o /tmp/%s \\\n", nvimVersion, tarball, tarball))
	b.WriteString(fmt.Sprintf("    && tar -xzf /tmp/%s -C /opt \\\n", tarball))
	b.WriteString(fmt.Sprintf("    && mv /opt/nvim-linux-%s /opt/nvim \\\n", nvimArch))
	b.WriteString("    && ln -s /opt/nvim/bin/nvim /usr/local/bin/nvim \\\n")
	b.WriteString(fmt.Sprintf("    && rm /tmp/%s\n\n", tarball))

	b.WriteString("# nvim config\n")
	b.WriteString("COPY nvim-config /root/.config/nvim\n\n")

	b.WriteString("# bootstrap lazy.nvim plugins\n")
	b.WriteString("RUN nvim --headless \"+Lazy! sync\" +qa || true\n\n")

	// Treesitter parsers (union across all languages).
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
}
```

Remove the old (now-enclosed) nvim and treesitter blocks from their original position in the function.

- [ ] **Step 4: Run tests**

```bash
cd /home/claude/ferry && go test ./internal/bundle/... -v
```

Expected: all PASS including the 2 new tests.

- [ ] **Step 5: Commit**

```bash
git add internal/bundle/dockerfile.go internal/bundle/dockerfile_test.go
git commit -m "feat(bundle): gate neovim Dockerfile steps on profile NvimEnabled()"
```

---

### Task 6: macOS track — gate nvim build + add local shell component section

**Files:**
- Modify: `internal/bundle/macos.go`

- [ ] **Step 1: Gate `buildNvimMacOS` on `NvimEnabled()`**

In `internal/bundle/macos.go`, update `buildMacOSTrack` section 1:

```go
// 1. Download neovim binary for macOS (only when nvim is enabled for this profile).
prof := opts.Lock.Profiles[opts.Profile]
if prof.NvimEnabled() {
	nvimComps, err := buildNvimMacOS(track, opts.Lock, s)
	if err != nil {
		r.Error = fmt.Errorf("nvim macOS: %w", err)
		return r
	}
	components = append(components, nvimComps...)
}
```

- [ ] **Step 2: Add section 5 for local shell components**

After section 4 (custom tools), add:

```go
// 5. Shell components — copied from local filesystem.
if sp := prof.Shell; sp != nil {
	type shellSpec struct {
		id          string
		localPath   string
		installPath string
		preserve    bool
	}
	var shellSpecs []shellSpec
	if sp.Framework != "" && sp.FrameworkPath != "" {
		shellSpecs = append(shellSpecs, shellSpec{
			id:          "shell/framework",
			localPath:   config.ExpandHome(sp.FrameworkPath),
			installPath: sp.FrameworkPath + "/",
		})
	}
	if sp.RCPath != "" {
		shellSpecs = append(shellSpecs, shellSpec{
			id:          "shell/rc",
			localPath:   config.ExpandHome(sp.RCPath),
			installPath: sp.RCPath,
			preserve:    true,
		})
	}
	if sp.ThemeConfigPath != "" {
		shellSpecs = append(shellSpecs, shellSpec{
			id:          "shell/theme-config",
			localPath:   config.ExpandHome(sp.ThemeConfigPath),
			installPath: sp.ThemeConfigPath,
			preserve:    true,
		})
	}

	for _, ss := range shellSpecs {
		tmpDir := fmt.Sprintf("/tmp/ferry-shell-%s-%s", track.Arch, sanitizeID(ss.id))
		if err := copyLocalToTmp(ss.localPath, tmpDir); err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping shell component %s: %v\n", ss.id, err)
			os.RemoveAll(tmpDir)
			continue
		}
		compressed, err := store.CompressDir(tmpDir, opts.Lock.Bundle.Exclude)
		os.RemoveAll(tmpDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping shell component %s: compress failed: %v\n", ss.id, err)
			continue
		}
		hash := store.HashBytes(compressed)
		if err := s.Put(hash, compressed); err != nil {
			r.Error = fmt.Errorf("storing shell component %s: %w", ss.id, err)
			return r
		}
		components = append(components, store.Component{
			ID:             ss.id,
			Hash:           hash,
			SizeCompressed: int64(len(compressed)),
			InstallPath:    ss.installPath,
			ArchSpecific:   false,
			Preserve:       ss.preserve,
		})
	}
}
```

- [ ] **Step 3: Verify compilation**

```bash
cd /home/claude/ferry && go build ./...
```

Expected: no errors.

- [ ] **Step 4: Run all tests**

```bash
cd /home/claude/ferry && go test ./...
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/bundle/macos.go
git commit -m "feat(bundle): gate nvim macOS build on NvimEnabled, add shell component bundling from local filesystem"
```

---

### Task 7: Init wizard — nvim toggle in Step 2 + real shell Step 4

**Files:**
- Modify: `internal/tui/init_wizard.go`

- [ ] **Step 1: Fix compile error from Task 1**

The `IncludeShell` field was removed from `ProfileConfig` in Task 1. In `init_wizard.go` at Step 4, the current code is:

```go
// Step 4: Shell
if shell.Type != "" {
    prof.IncludeShell = true
}
```

Replace with a temporary stub to fix the compile error:

```go
// Step 4: Shell — implemented below
_ = shell
```

Then run:

```bash
cd /home/claude/ferry && go build ./...
```

Expected: compiles cleanly.

- [ ] **Step 2: Add `shellDiscoveryToProfile` helper**

Add this function at the bottom of `internal/tui/init_wizard.go`:

```go
// shellDiscoveryToProfile converts discovery results to a ShellProfile,
// filtered by which items the user selected (by index: 0=framework, 1=rc, 2=theme).
func shellDiscoveryToProfile(d discovery.ShellDiscovery, selected map[int]bool) *config.ShellProfile {
	if d.Type == "" {
		return nil
	}
	sp := &config.ShellProfile{
		Type: d.Type,
	}
	idx := 0
	if d.PluginManager != "" && d.PluginManagerPath != "" {
		if selected[idx] {
			sp.Framework = d.PluginManager
			sp.FrameworkPath = d.PluginManagerPath
		}
		idx++
	}
	if d.ConfigPath != "" {
		if selected[idx] {
			sp.RCPath = d.ConfigPath
		}
		idx++
	}
	if d.ThemeConfigPath != "" {
		if selected[idx] {
			sp.Theme = d.ThemeDetected
			sp.ThemeConfigPath = d.ThemeConfigPath
		}
	}
	// If nothing was selected, return nil (user opted out of shell bundling entirely).
	if sp.Framework == "" && sp.RCPath == "" && sp.ThemeConfigPath == "" {
		return nil
	}
	return sp
}
```

- [ ] **Step 3: Update Step 2 — add nvim toggle before plugin list**

Replace the Step 2 block in `RunProfileWizard`:

```go
// Step 2: Neovim
{
	includeNvim := true
	nvimConfirmed, err := ConfirmPrompt("Step 2: Include Neovim?")
	if err != nil {
		return nil, false, err
	}
	includeNvim = nvimConfirmed
	prof.IncludeNvim = boolPtr(includeNvim)

	if includeNvim && nvim.Found {
		items := pluginItems(nvim.Plugins, prof.Plugins)
		m := NewMultiSelect("Step 2: Neovim plugins — select plugins to include", items)
		p := tea.NewProgram(m)
		final, err := p.Run()
		if err != nil {
			return nil, false, err
		}
		mm := final.(MultiSelectModel)
		if mm.aborted {
			return nil, true, nil
		}
		prof.Plugins = nil
		for _, item := range mm.Items {
			if item.Selected {
				prof.Plugins = append(prof.Plugins, item.Value)
			}
		}
	}
}
```

Add `boolPtr` as a local helper at the top of `init_wizard.go` (after imports):

```go
func boolPtr(b bool) *bool { return &b }
```

- [ ] **Step 4: Update Step 4 — replace stub with real shell MultiSelect**

Replace the shell step stub with:

```go
// Step 4: Shell
{
	if shell.Type != "" {
		// Build items from what was detected.
		var items []Item
		if shell.PluginManager != "" && shell.PluginManagerPath != "" {
			items = append(items, Item{
				Label:    fmt.Sprintf("Bundle shell framework (%s)", shell.PluginManagerPath),
				Value:    "framework",
				Selected: true,
			})
		}
		if shell.ConfigPath != "" {
			items = append(items, Item{
				Label:    fmt.Sprintf("Bundle %s", shell.ConfigPath),
				Value:    "rc",
				Selected: true,
			})
		}
		if shell.ThemeConfigPath != "" {
			items = append(items, Item{
				Label:    fmt.Sprintf("Bundle theme config (%s)", shell.ThemeConfigPath),
				Value:    "theme",
				Selected: true,
			})
		}

		if len(items) > 0 {
			fmt.Printf("\n  detected: %s", shell.Type)
			if shell.PluginManager != "" {
				fmt.Printf("  │  %s", shell.PluginManager)
			}
			if shell.ThemeDetected != "" {
				fmt.Printf("  │  %s", shell.ThemeDetected)
			}
			fmt.Println()

			m := NewMultiSelect("Step 4: Shell — select components to bundle", items)
			p := tea.NewProgram(m)
			final, err := p.Run()
			if err != nil {
				return nil, false, err
			}
			mm := final.(MultiSelectModel)
			if mm.aborted {
				return nil, true, nil
			}

			selected := make(map[int]bool)
			for i, item := range mm.Items {
				if item.Selected {
					selected[i] = true
				}
			}
			prof.Shell = shellDiscoveryToProfile(shell, selected)
		}
	}
}
```

- [ ] **Step 5: Verify compilation and tests**

```bash
cd /home/claude/ferry && go build ./... && go test ./...
```

Expected: all PASS, no compile errors.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/init_wizard.go
git commit -m "feat(tui): nvim opt-out toggle in Step 2, real shell bundling selection in Step 4"
```

---

### Task 8: Final wiring — verify full build compiles and all tests pass

**Files:**
- None new

- [ ] **Step 1: Run full test suite**

```bash
cd /home/claude/ferry && go test ./... -v 2>&1 | tail -40
```

Expected: all PASS, zero failures.

- [ ] **Step 2: Verify clean build**

```bash
cd /home/claude/ferry && go build ./... && echo "BUILD OK"
```

Expected: `BUILD OK`

- [ ] **Step 3: Smoke check the ferry binary**

```bash
cd /home/claude/ferry && go run . --help
```

Expected: ferry help output, no panics.

- [ ] **Step 4: Commit if any fixup changes were needed**

```bash
git add -A && git diff --cached --stat
```

Only commit if there are staged changes from fixups. Use:

```bash
git commit -m "fix: final wiring for shell bundling and nvim opt-out"
```
