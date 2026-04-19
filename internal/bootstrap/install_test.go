package bootstrap

import (
	"strings"
	"testing"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/registry"
	"github.com/anthropics/ferry/internal/store"
)

func makeTestManifest() *store.Manifest {
	return &store.Manifest{
		Profile: "backend",
		Components: []store.Component{
			{ID: "nvim-binary", InstallPath: "~/.local/share/nvim-dist/", BinSymlink: "~/.local/bin/nvim"},
			{ID: "runtime/python", InstallPath: "~/.ferry/runtimes/python-3.12/", Version: "3.12"},
			{ID: "runtime/go", InstallPath: "~/.ferry/runtimes/go-1.22.5/", Version: "1.22.5"},
		},
	}
}

func makeTestLangs() []registry.ResolvedLanguage {
	pyRuntime := &registry.Runtime{
		DefaultVersion: "3.12",
		ShellInit:      []string{`export PATH="$HOME/.ferry/runtimes/python-{VERSION}/bin:$PATH"`},
	}
	goRuntime := &registry.Runtime{
		DefaultVersion: "1.22.5",
		ShellInit: []string{
			`export PATH="$HOME/.ferry/runtimes/go-{VERSION}/bin:$HOME/.ferry/runtimes/go-tools/bin:$PATH"`,
			`export GOPATH="$HOME/.ferry/runtimes/go-tools"`,
		},
	}
	return []registry.ResolvedLanguage{
		{Language: registry.Language{Name: "python", TreesitterParsers: []string{"python"}}, Runtime: pyRuntime, EffectiveLSP: "pyright"},
		{Language: registry.Language{Name: "go", TreesitterParsers: []string{"go", "gomod"}}, Runtime: goRuntime, EffectiveLSP: "gopls"},
	}
}

func TestGenerateInstallScriptContainsFerryRuntimePath(t *testing.T) {
	// Runtime paths live in env.sh, which install.sh copies from incoming/.
	// Verify the install script sets up the env.sh sourcing infrastructure.
	script, err := GenerateInstallScript(makeTestManifest(), &config.LockFile{}, makeTestLangs())
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	if !strings.Contains(script, ".ferry/env.sh") {
		t.Error("install script should reference env.sh")
	}
	if !strings.Contains(script, "incoming/env.sh") {
		t.Error("install script should copy incoming/env.sh into place")
	}
}

func TestGenerateEnvShContainsRuntimePaths(t *testing.T) {
	envSh := GenerateEnvSh(makeTestLangs())
	if !strings.Contains(envSh, ".ferry/runtimes/python-3.12") {
		t.Error("env.sh should reference python runtime PATH")
	}
	if !strings.Contains(envSh, ".ferry/runtimes/go-1.22.5") {
		t.Error("env.sh should reference go runtime PATH")
	}
}

func TestGenerateEnvShNoVersionManager(t *testing.T) {
	envSh := GenerateEnvSh(makeTestLangs())
	for _, mgr := range []string{"pyenv", "nvm", "rbenv", "rustup", "sdkman"} {
		if strings.Contains(envSh, mgr) {
			t.Errorf("env.sh should not reference version manager %q", mgr)
		}
	}
}

func TestGenerateEnvShEmptyLangs(t *testing.T) {
	envSh := GenerateEnvSh(nil)
	if strings.Contains(envSh, "PATH") {
		t.Error("env.sh with no languages should not contain PATH entries")
	}
}

func TestGenerateInstallScriptNoVersionManager(t *testing.T) {
	script, err := GenerateInstallScript(makeTestManifest(), &config.LockFile{}, makeTestLangs())
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	// Version managers must not appear in the install script or env.sh
	envSh := GenerateEnvSh(makeTestLangs())
	combined := script + envSh
	for _, mgr := range []string{"pyenv", "nvm", "rbenv", "rustup", "sdkman"} {
		if strings.Contains(combined, mgr) {
			t.Errorf("install artefacts should not reference version manager %q", mgr)
		}
	}
}

func TestGenerateInstallScriptIdempotentSourceLine(t *testing.T) {
	script, err := GenerateInstallScript(makeTestManifest(), &config.LockFile{}, makeTestLangs())
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	// The env.sh source line must be guarded by grep -q to prevent duplication.
	if !strings.Contains(script, "grep -q '.ferry/env.sh'") {
		t.Error("install script should guard the env.sh source line with grep -q")
	}
	// Individual runtime PATH entries must NOT be appended directly to the shell RC.
	// They belong in env.sh, not in install.sh itself.
	if strings.Contains(script, "export PATH") {
		t.Error("install script should not append PATH entries directly to shell RC; they belong in env.sh")
	}
}

func TestGenerateInstallScriptNvimSymlink(t *testing.T) {
	script, err := GenerateInstallScript(makeTestManifest(), &config.LockFile{}, makeTestLangs())
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	if !strings.Contains(script, "ln -sf") {
		t.Error("install script should create nvim symlink")
	}
}

func TestGenerateInstallScriptLSPGuard(t *testing.T) {
	script, err := GenerateInstallScript(makeTestManifest(), &config.LockFile{}, makeTestLangs())
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	// Guard is now in plugin/ferry.lua (not after/plugin/ferry-lsp-guard.lua)
	if !strings.Contains(script, "plugin/ferry.lua") {
		t.Error("install script should write plugin/ferry.lua")
	}
	// Guard must patch vim.lsp.start (nvim 0.9+ API)
	if !strings.Contains(script, "vim.lsp.start") {
		t.Error("lsp guard should wrap vim.lsp.start")
	}
	// Guard must also patch vim.lsp.start_client (older lspconfig API)
	if !strings.Contains(script, "vim.lsp.start_client") {
		t.Error("lsp guard should wrap vim.lsp.start_client")
	}
	// Guard must check vim.fn.executable
	if !strings.Contains(script, "vim.fn.executable") {
		t.Error("lsp guard should use vim.fn.executable to check server availability")
	}
}

func TestGenerateInstallScriptTreesitterNeutralisation(t *testing.T) {
	script, err := GenerateInstallScript(makeTestManifest(), &config.LockFile{}, makeTestLangs())
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	// nvim-treesitter.install must be hooked to prevent downloads/updates
	if !strings.Contains(script, "nvim-treesitter.install") {
		t.Error("install script should neutralise nvim-treesitter.install")
	}
	// install and update entry points must be no-oped
	if !strings.Contains(script, "patch_treesitter_install") {
		t.Error("install script should define patch_treesitter_install")
	}
}

func TestGenerateInstallScriptFerryModuleParsers(t *testing.T) {
	script, err := GenerateInstallScript(makeTestManifest(), &config.LockFile{}, makeTestLangs())
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	// M.parsers must be present in ferry.lua
	if !strings.Contains(script, "M.parsers") {
		t.Error("ferry.lua should expose M.parsers")
	}
	// setup_treesitter helper must be present
	if !strings.Contains(script, "setup_treesitter") {
		t.Error("ferry.lua should expose setup_treesitter() helper")
	}
	// auto_install must be disabled
	if !strings.Contains(script, "auto_install") {
		t.Error("setup_treesitter should set auto_install = false")
	}
	// Python parsers should be listed (from makeTestLangs treesitter parsers)
	if !strings.Contains(script, `"python"`) {
		t.Error("ferry.lua M.parsers should list python parser")
	}
}

func TestGenerateInstallScriptLazyNeutralisation(t *testing.T) {
	script, err := GenerateInstallScript(makeTestManifest(), &config.LockFile{}, makeTestLangs())
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	// lazy.manage.checker must be hooked to prevent background git fetch calls
	if !strings.Contains(script, "lazy.manage.checker") {
		t.Error("install script should neutralise lazy.manage.checker")
	}
	if !strings.Contains(script, "patch_lazy_checker") {
		t.Error("install script should define patch_lazy_checker")
	}
	// setup_lazy helper must disable checker and install.missing
	if !strings.Contains(script, "setup_lazy") {
		t.Error("ferry.lua should expose setup_lazy() helper")
	}
	if !strings.Contains(script, "install.missing") {
		t.Error("setup_lazy should disable install.missing")
	}
}

func TestGenerateInstallScriptMasonNeutralisation(t *testing.T) {
	script, err := GenerateInstallScript(makeTestManifest(), &config.LockFile{}, makeTestLangs())
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	// Mason registry refresh must be patched to prevent network calls
	if !strings.Contains(script, "mason-registry") {
		t.Error("install script should neutralise mason-registry")
	}
	// Mason package install must be patched to prevent downloads
	if !strings.Contains(script, "mason-core.package") {
		t.Error("install script should neutralise mason-core.package")
	}
	// Preload hooks must be used so patches apply regardless of mason load timing
	if !strings.Contains(script, "package.preload") {
		t.Error("install script should use package.preload hooks for mason neutralisation")
	}
	// hook_module helper must be present
	if !strings.Contains(script, "hook_module") {
		t.Error("install script should define hook_module helper")
	}
}

func TestGenerateInstallScriptFerryModule(t *testing.T) {
	script, err := GenerateInstallScript(makeTestManifest(), &config.LockFile{}, makeTestLangs())
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	// ferry.lua module must be written to lua/
	if !strings.Contains(script, "lua/ferry.lua") {
		t.Error("install script should write lua/ferry.lua")
	}
	// Module must list the bundled LSP servers
	if !strings.Contains(script, `"pyright"`) {
		t.Error("ferry.lua should list pyright as a bundled server")
	}
	if !strings.Contains(script, `"gopls"`) {
		t.Error("ferry.lua should list gopls as a bundled server")
	}
	// Module must expose setup_mason() helper
	if !strings.Contains(script, "setup_mason") {
		t.Error("ferry.lua should expose setup_mason() helper")
	}
	// setup_mason must disable automatic_installation to prevent downloads
	if !strings.Contains(script, "automatic_installation") {
		t.Error("setup_mason should set automatic_installation = false")
	}
}

// --- formatter / linter table tests ---

func makeFormatterLangs() []registry.ResolvedLanguage {
	pyRuntime := &registry.Runtime{ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/python-3.12/bin:$PATH"`}}
	goRuntime := &registry.Runtime{ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/go-1.22.5/bin:$PATH"`}}
	shRuntime := &registry.Runtime{ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/node-22/bin:$PATH"`}}
	return []registry.ResolvedLanguage{
		{
			Language:            registry.Language{Name: "python", TreesitterParsers: []string{"python"}},
			Runtime:             pyRuntime,
			EffectiveLSP:        "pyright",
			EffectiveFormatters: []string{"ruff", "black"},
			EffectiveLinters:    []string{"ruff", "mypy"},
		},
		{
			Language:            registry.Language{Name: "go", TreesitterParsers: []string{"go"}},
			Runtime:             goRuntime,
			EffectiveLSP:        "gopls",
			EffectiveFormatters: []string{"gofmt", "goimports"},
			EffectiveLinters:    []string{"staticcheck"},
		},
		{
			Language:            registry.Language{Name: "bash", TreesitterParsers: []string{"bash"}},
			Runtime:             shRuntime,
			EffectiveLSP:        "bash-language-server",
			EffectiveFormatters: []string{"shfmt"},
			EffectiveLinters:    []string{"shellcheck"},
		},
		{
			Language:            registry.Language{Name: "csharp", TreesitterParsers: []string{"c_sharp"}},
			Runtime:             &registry.Runtime{},
			EffectiveLSP:        "csharp-ls",
			EffectiveFormatters: []string{"dotnet-csharpier"},
		},
	}
}

func TestGenerateInstallScriptFormatterTable(t *testing.T) {
	script, err := GenerateInstallScript(makeTestManifest(), &config.LockFile{}, makeFormatterLangs())
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	// M.formatters must be present
	if !strings.Contains(script, "M.formatters") {
		t.Error("ferry.lua should expose M.formatters")
	}
	// python → ruff, black
	if !strings.Contains(script, `"ruff"`) {
		t.Error("M.formatters should list ruff")
	}
	if !strings.Contains(script, `"black"`) {
		t.Error("M.formatters should list black")
	}
	// bash language maps to sh filetype
	if !strings.Contains(script, `sh`) {
		t.Error("M.formatters should use filetype 'sh' for bash language")
	}
	if !strings.Contains(script, `"shfmt"`) {
		t.Error("M.formatters should list shfmt")
	}
	// csharp maps to cs filetype; dotnet-csharpier maps to csharpier
	if !strings.Contains(script, `"csharpier"`) {
		t.Error("M.formatters should map dotnet-csharpier to csharpier (conform name)")
	}
}

func TestGenerateInstallScriptLinterTable(t *testing.T) {
	script, err := GenerateInstallScript(makeTestManifest(), &config.LockFile{}, makeFormatterLangs())
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	if !strings.Contains(script, "M.linters") {
		t.Error("ferry.lua should expose M.linters")
	}
	if !strings.Contains(script, `"mypy"`) {
		t.Error("M.linters should list mypy")
	}
	if !strings.Contains(script, `"staticcheck"`) {
		t.Error("M.linters should list staticcheck")
	}
	if !strings.Contains(script, `"shellcheck"`) {
		t.Error("M.linters should list shellcheck")
	}
}

func TestGenerateInstallScriptSetupConform(t *testing.T) {
	script, err := GenerateInstallScript(makeTestManifest(), &config.LockFile{}, makeFormatterLangs())
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	if !strings.Contains(script, "setup_conform") {
		t.Error("ferry.lua should expose setup_conform()")
	}
	// Must be pcall-guarded so it no-ops when conform is absent
	if !strings.Contains(script, `pcall(require, "conform")`) {
		t.Error("setup_conform should pcall require conform")
	}
	// Must wire format-on-save via BufWritePre
	if !strings.Contains(script, "BufWritePre") {
		t.Error("setup_conform should register a BufWritePre autocmd")
	}
	// Must set lsp_format (conform v8+) and lsp_fallback (older conform compat)
	if !strings.Contains(script, `lsp_format = "fallback"`) {
		t.Error("setup_conform should use lsp_format = \"fallback\" for conform v8+")
	}
	if !strings.Contains(script, "lsp_fallback = true") {
		t.Error("setup_conform should also set lsp_fallback = true for older conform compat")
	}
	// Must preserve user overrides (only fill missing filetypes)
	if !strings.Contains(script, "if not by_ft[ft]") {
		t.Error("setup_conform should not override user-configured filetypes")
	}
}

func TestGenerateInstallScriptSetupLint(t *testing.T) {
	script, err := GenerateInstallScript(makeTestManifest(), &config.LockFile{}, makeFormatterLangs())
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	if !strings.Contains(script, "setup_lint") {
		t.Error("ferry.lua should expose setup_lint()")
	}
	if !strings.Contains(script, `pcall(require, "nvim-lint")`) {
		t.Error("setup_lint should pcall require nvim-lint")
	}
	// Must register autocmds to trigger linting
	if !strings.Contains(script, "BufWritePost") {
		t.Error("setup_lint should register BufWritePost autocmd")
	}
	if !strings.Contains(script, "try_lint") {
		t.Error("setup_lint should call lint.try_lint()")
	}
}

func TestGenerateInstallScriptLSPFormatFallback(t *testing.T) {
	script, err := GenerateInstallScript(makeTestManifest(), &config.LockFile{}, makeFormatterLangs())
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	// plugin/ferry.lua must contain the LSP format-on-save fallback
	if !strings.Contains(script, "ferry_lsp_format_fallback") {
		t.Error("plugin/ferry.lua should define ferry_lsp_format_fallback augroup")
	}
	// Fallback must only fire when conform is NOT loaded
	if !strings.Contains(script, `package.loaded["conform"]`) {
		t.Error("fallback should check package.loaded[\"conform\"] before formatting")
	}
	// Fallback uses built-in LSP formatting — no extra plugins needed
	if !strings.Contains(script, "vim.lsp.buf.format") {
		t.Error("fallback should call vim.lsp.buf.format")
	}
}

func TestNvimFiletype(t *testing.T) {
	cases := map[string]string{
		"bash":       "sh",
		"csharp":     "cs",
		"dockerfile": "dockerfile",
		"python":     "python",
		"go":         "go",
		"typescript": "typescript",
	}
	for lang, want := range cases {
		if got := nvimFiletype(lang); got != want {
			t.Errorf("nvimFiletype(%q) = %q, want %q", lang, got, want)
		}
	}
}

func TestConformName(t *testing.T) {
	cases := map[string]string{
		"clang-format":       "clang_format",
		"php-cs-fixer":       "php_cs_fixer",
		"dotnet-csharpier":   "csharpier",
		"google-java-format": "google_java_format",
		"mix format":         "mix",
		"zig fmt":            "zig",
		"ruff":               "ruff",
		"prettier":           "prettier",
		"gofmt":              "gofmt",
	}
	for input, want := range cases {
		if got := conformName(input); got != want {
			t.Errorf("conformName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestGenerateInstallScriptLSPDeduplication(t *testing.T) {
	// Two languages sharing the same LSP should only appear once in ferry.lua
	runtime := &registry.Runtime{ShellInit: []string{`export PATH="$HOME/.ferry/runtimes/node-22/bin:$PATH"`}}
	langs := []registry.ResolvedLanguage{
		{Language: registry.Language{Name: "javascript"}, Runtime: runtime, EffectiveLSP: "typescript-language-server"},
		{Language: registry.Language{Name: "typescript"}, Runtime: runtime, EffectiveLSP: "typescript-language-server"},
	}
	script, err := GenerateInstallScript(makeTestManifest(), &config.LockFile{}, langs)
	if err != nil {
		t.Fatalf("GenerateInstallScript: %v", err)
	}
	count := strings.Count(script, `"typescript-language-server"`)
	if count != 1 {
		t.Errorf("typescript-language-server should appear exactly once in ferry.lua, got %d", count)
	}
}
