package bootstrap

import (
	"fmt"
	"sort"
	"strings"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/registry"
	"github.com/anthropics/ferry/internal/store"
)

// GenerateInstallScript generates a POSIX sh install.sh for the given manifest.
// langs carries the ShellInit lines for each bundled runtime.
// Version managers are never referenced — PATH points directly at ~/.ferry/runtimes/.
func GenerateInstallScript(m *store.Manifest, lock *config.LockFile, langs []registry.ResolvedLanguage) (string, error) {
	var b strings.Builder

	b.WriteString(`#!/bin/sh
# ferry install.sh — generated, do not edit
set -e

FERRY_DIR="$HOME/.ferry"
STORE_DIR="$FERRY_DIR/store"
BIN_DIR="$HOME/.local/bin"
NVIM_DATA="$HOME/.local/share/nvim"
NVIM_DIST="$HOME/.local/share/nvim-dist"

mkdir -p "$BIN_DIR" "$NVIM_DATA" "$NVIM_DIST" "$FERRY_DIR"

# Atomically installs a directory component: extracts to a temp dir, then swaps into place.
# If extraction fails the destination is untouched.
install_component() {
  local hash="$1"
  local dest="$2"
  local src="$STORE_DIR/${hash}.tar.zst"
  local tmp="${dest}.ferry-new"
  rm -rf "$tmp"
  mkdir -p "$tmp"
  tar --zstd -xf "$src" -C "$tmp" || { rm -rf "$tmp"; return 1; }
  rm -rf "${dest:?}"
  mv "$tmp" "$dest"
}

# Installs a single-file component (binary) directly to dest.
install_file() {
  local hash="$1"
  local dest="$2"
  local src="$STORE_DIR/${hash}.tar.zst"
  local tmpdir
  tmpdir="$(mktemp -d)"
  tar --zstd -xf "$src" -C "$tmpdir" || { rm -rf "$tmpdir"; return 1; }
  local f
  f="$(ls "$tmpdir" | head -1)"
  [ -n "$f" ] || { rm -rf "$tmpdir"; return 1; }
  mkdir -p "$(dirname "$dest")"
  rm -f "$dest"
  mv "$tmpdir/$f" "$dest"
  rm -rf "$tmpdir"
  chmod +x "$dest"
}

`)

	// Read one blank line from stdin so the script doesn't block if the caller
	// pipes something (kept for forward compatibility; currently unused).
	b.WriteString("IFS= read -r _ || true\n\n")

	// Extract each component
	for _, c := range m.Components {
		b.WriteString(fmt.Sprintf("# %s\n", c.ID))
		installPath := config.ExpandHome(c.InstallPath)
		if strings.HasSuffix(c.InstallPath, "/") {
			b.WriteString(fmt.Sprintf("install_component %q %q\n", c.Hash, installPath))
		} else {
			b.WriteString(fmt.Sprintf("install_file %q %q\n", c.Hash, installPath))
		}
		if c.BinSymlink != "" {
			binLink := config.ExpandHome(c.BinSymlink)
			b.WriteString(fmt.Sprintf("ln -sf %q/bin/nvim %q\n", installPath, binLink))
		}
		b.WriteString("\n")
	}

	// Shell init — one source line added once to the shell RC.
	// All runtime PATH entries live in ~/.ferry/env.sh, which ferry owns
	// completely and rewrites on every install/update. This means version
	// upgrades automatically update PATH with no stale entries left behind,
	// and ferry clean can remove everything by deleting one file.
	b.WriteString(`# Runtime PATH setup.
# Add a single source line to the shell RC; env.sh contains the actual entries.
# The source line is idempotent — only added if not already present.
SHELL_RC="$HOME/.zshrc"
[ -f "$HOME/.bashrc" ] && SHELL_RC="$HOME/.bashrc"
[ -n "$SHELL" ] && echo "$SHELL" | grep -q zsh && SHELL_RC="$HOME/.zshrc"

grep -q '.ferry/env.sh' "$SHELL_RC" 2>/dev/null || \
  printf '\n# ferry runtime PATH\n[ -f "$HOME/.ferry/env.sh" ] && . "$HOME/.ferry/env.sh"\n' >> "$SHELL_RC"

# Install env.sh (runtime PATH entries) from incoming/.
cp "$FERRY_DIR/incoming/env.sh" "$FERRY_DIR/env.sh" 2>/dev/null || true

`)

	// Nvim LSP shims: two lua files written to ~/.config/nvim/ on the target.
	// ferry-lsp-guard.lua patches vim.lsp.start to silently skip servers whose
	// executables aren't installed — requires no changes to the user's nvim config.
	// ferry.lua exposes the bundled server list for optional user integration.
	b.WriteString(generateNvimLuaSetup(langs))

	b.WriteString(`# Set permissions.
chmod +x "$BIN_DIR"/* 2>/dev/null || true
chmod -R +x "$HOME/.ferry/runtimes/"*/bin/ 2>/dev/null || true

# Write manifest.
cp "$FERRY_DIR/incoming/manifest.json" "$FERRY_DIR/manifest.json" 2>/dev/null || true

`)

	b.WriteString(`echo "ferry: install complete"
`)

	return b.String(), nil
}

// generateNvimLuaSetup returns the shell snippet that writes the ferry lua
// files into ~/.config/nvim/ on the target.
//
// Two files are generated:
//
//   plugin/ferry.lua   — auto-sourced early (before plugin configs run).
//                        Guards vim.lsp.start + vim.lsp.start_client, neutralises
//                        mason/nvim-treesitter/lazy.nvim network calls, and provides
//                        a LSP format-on-save fallback for users without conform.nvim.
//
//   lua/ferry.lua      — optional user module. Exposes server/parser/formatter/linter
//                        lists and setup helpers for conform, nvim-lint, mason, treesitter,
//                        and lazy.nvim.
func generateNvimLuaSetup(langs []registry.ResolvedLanguage) string {
	// Build lua table of LSP server names from the bundled languages.
	var serverEntries []string
	seenServers := map[string]bool{}
	for _, rl := range langs {
		if rl.EffectiveLSP != "" && !seenServers[rl.EffectiveLSP] {
			serverEntries = append(serverEntries, fmt.Sprintf("  %q,", rl.EffectiveLSP))
			seenServers[rl.EffectiveLSP] = true
		}
	}
	serverList := strings.Join(serverEntries, "\n")

	// Build lua table of treesitter parser names from the bundled languages.
	var parserEntries []string
	seenParsers := map[string]bool{}
	for _, rl := range langs {
		for _, p := range rl.Language.TreesitterParsers {
			if !seenParsers[p] {
				parserEntries = append(parserEntries, fmt.Sprintf("  %q,", p))
				seenParsers[p] = true
			}
		}
	}
	parserList := strings.Join(parserEntries, "\n")

	// Build per-filetype formatter and linter tables for conform.nvim / nvim-lint.
	type ftEntry struct{ formatters, linters []string }
	ftMap := map[string]*ftEntry{}
	for _, rl := range langs {
		ft := nvimFiletype(rl.Language.Name)
		if ftMap[ft] == nil {
			ftMap[ft] = &ftEntry{}
		}
		for _, f := range rl.EffectiveFormatters {
			ftMap[ft].formatters = append(ftMap[ft].formatters, conformName(f))
		}
		for _, l := range rl.EffectiveLinters {
			ftMap[ft].linters = append(ftMap[ft].linters, nvimLintName(l))
		}
	}
	var sortedFTs []string
	for ft := range ftMap {
		sortedFTs = append(sortedFTs, ft)
	}
	sort.Strings(sortedFTs)

	var fmtLines, lintLines []string
	for _, ft := range sortedFTs {
		e := ftMap[ft]
		if len(e.formatters) > 0 {
			quoted := make([]string, len(e.formatters))
			for i, f := range e.formatters {
				quoted[i] = fmt.Sprintf("%q", f)
			}
			fmtLines = append(fmtLines, fmt.Sprintf("  %-14s = { %s },", ft, strings.Join(quoted, ", ")))
		}
		if len(e.linters) > 0 {
			quoted := make([]string, len(e.linters))
			for i, l := range e.linters {
				quoted[i] = fmt.Sprintf("%q", l)
			}
			lintLines = append(lintLines, fmt.Sprintf("  %-14s = { %s },", ft, strings.Join(quoted, ", ")))
		}
	}
	formatterTable := strings.Join(fmtLines, "\n")
	linterTable := strings.Join(lintLines, "\n")

	return fmt.Sprintf(`# Nvim configuration for no-internet targets.
# plugin/ferry.lua:  early guard — sourced before plugin configs, patches vim.lsp,
#                    neutralises mason/treesitter/lazy/vim.pack/mini.deps network calls.
# lua/ferry.lua:     optional user module with server/parser lists + setup helpers.
NVIM_PLUGIN_DIR="$HOME/.config/nvim/plugin"
NVIM_LUA_DIR="$HOME/.config/nvim/lua"
mkdir -p "$NVIM_PLUGIN_DIR" "$NVIM_LUA_DIR"

cat > "$NVIM_PLUGIN_DIR/ferry.lua" << 'FERRY_PLUGIN_EOF'
-- Generated by ferry — do not edit.
-- Sourced automatically from plugin/ before lazy.nvim runs plugin configs.

-- ---------------------------------------------------------------------------
-- 1. LSP executable guard
-- Wraps vim.lsp.start (nvim 0.9+) and vim.lsp.start_client (older API) so
-- any server whose executable is not in PATH is silently skipped.
-- No changes to your nvim config required.
-- ---------------------------------------------------------------------------
local function cmd_is_executable(cmd)
  if type(cmd) == "table" then cmd = cmd[1] end
  return type(cmd) ~= "string" or vim.fn.executable(cmd) == 1
end

if vim.lsp then
  if vim.lsp.start then
    local _orig = vim.lsp.start
    vim.lsp.start = function(config, opts)
      if config and not cmd_is_executable(config.cmd) then return nil end
      return _orig(config, opts)
    end
  end
  -- Older lspconfig used start_client directly
  if vim.lsp.start_client then
    local _orig = vim.lsp.start_client
    vim.lsp.start_client = function(config)
      if config and not cmd_is_executable(config.cmd) then return nil end
      return _orig(config)
    end
  end
end

-- ---------------------------------------------------------------------------
-- 2. Mason network neutralisation
-- On a no-internet target, mason must not attempt to download packages.
-- Uses package.preload hooks so patches apply regardless of load timing —
-- whether mason loads at startup, on VimEnter, or on first buffer open.
-- ---------------------------------------------------------------------------
local function patch_mason_registry(M)
  -- refresh() is the entry point for all mason network activity.
  -- Making it a synchronous no-op prevents registry downloads and
  -- stops ensure_installed from queuing install jobs.
  M.refresh = function(callback)
    if type(callback) == "function" then callback(nil) end
  end
  return M
end

local function patch_mason_package(M)
  -- Package:install() is the method mason calls to download a tool.
  -- Replace it with a quiet no-op so missing tools produce no errors.
  if M.Package and type(M.Package.install) == "function" then
    M.Package.install = function(self)
      return nil
    end
  end
  return M
end

-- Installs a preload hook: if the module is already loaded, patch it now;
-- otherwise intercept its first require() and patch the result.
local function hook_module(modname, patcher)
  if package.loaded[modname] then
    patcher(package.loaded[modname])
    return
  end
  local prev_preload = package.preload[modname]
  package.preload[modname] = function(name)
    -- Restore the original loader before requiring to avoid infinite recursion
    package.preload[modname] = prev_preload
    local ok, M = pcall(require, name)
    if ok and M then
      patcher(M)
      package.loaded[modname] = M
    end
    return package.loaded[modname]
  end
end

hook_module("mason-registry",     patch_mason_registry)
hook_module("mason-core.package", patch_mason_package)

-- ---------------------------------------------------------------------------
-- 3. nvim-treesitter download neutralisation
-- On a no-internet target, nvim-treesitter must not attempt to download or
-- update parsers. Parsers are pre-bundled by ferry; all install/update entry
-- points in nvim-treesitter.install are replaced with silent no-ops via the
-- same preload hook mechanism, so the patch applies regardless of when
-- nvim-treesitter loads or whether ensure_installed / auto_install is set.
-- ---------------------------------------------------------------------------
local function patch_treesitter_install(M)
  local noop = function() end
  -- Primary entry points called by :TSInstall, :TSUpdate, ensure_installed,
  -- and auto_install. Replace all with no-ops to prevent network calls.
  if type(M.install) == "function"             then M.install = noop end
  if type(M.update) == "function"              then M.update = noop end
  if type(M.install_all_parsers) == "function" then M.install_all_parsers = noop end
  if type(M.update_all_installed) == "function" then M.update_all_installed = noop end
  -- nvim-treesitter 0.12 installs via an async job scheduler; neutralise it too.
  if type(M.cancel_jobs) == "function"         then M.cancel_jobs = noop end
  if type(M.schedule_install) == "function"    then M.schedule_install = noop end
  return M
end

hook_module("nvim-treesitter.install", patch_treesitter_install)

-- ---------------------------------------------------------------------------
-- 4. lazy.nvim update checker neutralisation
-- lazy.nvim has two network-active behaviours on a no-internet target:
--   a) checker — background timer that runs git fetch to find plugin updates.
--      Neutralised by patching lazy.manage.checker so start() is a no-op.
--   b) install.missing = true (default) — clones plugins absent from disk.
--      Neutralised via setup_lazy() in lua/ferry.lua, which forces
--      install.missing = false before calling require('lazy').setup().
-- The checker patch is a hard stop applied regardless of user config;
-- setup_lazy() is the integration point for the install.missing default.
-- ---------------------------------------------------------------------------
local function patch_lazy_checker(M)
  local noop = function() end
  if type(M.start) == "function"  then M.start = noop end
  if type(M.check) == "function"  then M.check = noop end
  if type(M.update) == "function" then M.update = noop end
  return M
end

hook_module("lazy.manage.checker", patch_lazy_checker)

-- ---------------------------------------------------------------------------
-- 5. vim.pack neutralisation (Neovim 0.12+)
-- vim.pack is Neovim 0.12's built-in package manager. It is a global table
-- in the vim namespace, not a requireable Lua module, so hook_module cannot
-- intercept it. We patch it directly at startup instead.
-- update() is replaced with a no-op to prevent git-fetch operations.
-- add() is intentionally left intact: for already-installed plugins it only
-- extends runtimepath (no network); it only attempts a git clone when the
-- plugin directory is absent — which should not occur on a ferry-deployed
-- machine where plugins are already in place.
-- ---------------------------------------------------------------------------
if vim.pack then
  if type(vim.pack.update) == "function" then
    vim.pack.update = function() end
  end
end

-- ---------------------------------------------------------------------------
-- 6. mini.deps neutralisation
-- mini.deps (the mini.nvim package manager) does not fire network calls at
-- startup automatically. update() performs explicit git-fetches and is
-- replaced with a no-op. add() is left intact for the same reason as
-- vim.pack.add(): it only clones absent plugins; for installed plugins it
-- simply extends runtimepath, which must be allowed for the user's config
-- to work correctly.
-- ---------------------------------------------------------------------------
local function patch_mini_deps(M)
  if type(M.update) == "function" then M.update = function() end end
  return M
end

hook_module("mini.deps", patch_mini_deps)

-- ---------------------------------------------------------------------------
-- 7. LSP format-on-save fallback
-- If conform.nvim is not loaded at write time, fall back to vim.lsp.buf.format().
-- This means formatting works out of the box for any LSP-capable buffer with
-- zero plugin config — no conform, no extra setup.
-- When the user calls require('ferry').setup_conform(), that wires conform's
-- own BufWritePre autocmd and this fallback steps aside automatically.
-- ---------------------------------------------------------------------------
vim.api.nvim_create_autocmd("BufWritePre", {
  group = vim.api.nvim_create_augroup("ferry_lsp_format_fallback", { clear = true }),
  callback = function()
    if not package.loaded["conform"] then
      vim.lsp.buf.format({ async = false, timeout_ms = 500 })
    end
  end,
})
FERRY_PLUGIN_EOF

cat > "$NVIM_LUA_DIR/ferry.lua" << 'FERRY_MODULE_EOF'
-- Generated by ferry — do not edit.
-- Optional integration module. plugin/ferry.lua already handles silent
-- skipping of missing LSPs, prevents network calls from mason, nvim-treesitter,
-- lazy.nvim, vim.pack, and mini.deps, and provides LSP format-on-save.
-- Use this module when you want explicit ferry-aware setup.
--
-- Formatters + conform.nvim (replaces your conform.setup call):
--   require('ferry').setup_conform({ -- any extra conform opts -- })
--
-- Linters + nvim-lint (replaces your nvim-lint config):
--   require('ferry').setup_lint()
--
-- lazy.nvim (replaces your require('lazy').setup() call):
--   require('ferry').setup_lazy(plugins, { -- your lazy opts -- })
--
-- LSP — Usage A (mason-lspconfig):
--   require('ferry').setup_mason({ on_attach = ..., capabilities = ... })
--
-- LSP — Usage B (direct lspconfig):
--   for _, server in ipairs(require('ferry').servers) do
--     require('lspconfig')[server].setup({ on_attach = ..., capabilities = ... })
--   end
--
-- Treesitter (replaces your nvim-treesitter.configs.setup call):
--   require('ferry').setup_treesitter({ highlight = { enable = true } })
local M = {}

-- LSP servers bundled by ferry for this target.
M.servers = {
%s
}

-- Treesitter parsers bundled by ferry for this target.
M.parsers = {
%s
}

-- Formatters bundled by ferry, keyed by neovim filetype.
-- Suitable for direct use as conform.nvim's formatters_by_ft table.
M.formatters = {
%s
}

-- Linters bundled by ferry, keyed by neovim filetype.
-- Suitable for direct use as nvim-lint's linters_by_ft table.
M.linters = {
%s
}

-- setup_conform(opts?)
-- Configures conform.nvim with ferry's bundled formatters and wires
-- format-on-save (BufWritePre). Only fills in filetypes not already
-- present in opts.formatters_by_ft, so user overrides are preserved.
-- lsp_format = "fallback" is set so filetypes without a dedicated formatter
-- (e.g. SQL) still format via their LSP server.
-- Safe to call unconditionally — silently no-ops if conform is not installed.
function M.setup_conform(opts)
  local ok, conform = pcall(require, "conform")
  if not ok then return end
  local cfg = {}
  for k, v in pairs(opts or {}) do cfg[k] = v end
  local by_ft = cfg.formatters_by_ft or {}
  for ft, fmts in pairs(M.formatters) do
    if not by_ft[ft] then
      by_ft[ft] = fmts
    end
  end
  cfg.formatters_by_ft = by_ft
  conform.setup(cfg)
  -- Wire format-on-save. Use lsp_format (conform v8+) with lsp_fallback as
  -- a compatibility alias for older conform versions that predated the rename.
  vim.api.nvim_create_autocmd("BufWritePre", {
    group = vim.api.nvim_create_augroup("ferry_conform_format", { clear = true }),
    callback = function(args)
      conform.format({ bufnr = args.buf, async = false, lsp_format = "fallback", lsp_fallback = true })
    end,
  })
end

-- setup_lint(opts?)
-- Configures nvim-lint with ferry's bundled linters and registers
-- BufWritePost / BufReadPost autocmds to trigger linting.
-- Only fills in filetypes not already present in lint.linters_by_ft,
-- so user overrides are preserved.
-- Safe to call unconditionally — silently no-ops if nvim-lint is not installed.
function M.setup_lint()
  local ok, lint = pcall(require, "nvim-lint")
  if not ok then return end
  lint.linters_by_ft = lint.linters_by_ft or {}
  for ft, linters in pairs(M.linters) do
    if not lint.linters_by_ft[ft] then
      lint.linters_by_ft[ft] = linters
    end
  end
  vim.api.nvim_create_autocmd({ "BufWritePost", "BufReadPost" }, {
    group = vim.api.nvim_create_augroup("ferry_nvim_lint", { clear = true }),
    callback = function() lint.try_lint() end,
  })
end

-- setup_mason(server_config?)
-- Configures mason-lspconfig to never install packages (no internet), then
-- calls lspconfig.setup() for each bundled server using server_config.
-- The vim.lsp.start guard in plugin/ferry.lua ensures servers not in PATH
-- are silently skipped even if lspconfig.setup() is called for them.
function M.setup_mason(server_config)
  local mlsp_ok, mlsp = pcall(require, "mason-lspconfig")
  if mlsp_ok then
    mlsp.setup({
      ensure_installed       = {},
      automatic_installation = false,
    })
  end
  local lsp_ok, lspconfig = pcall(require, "lspconfig")
  if not lsp_ok then return end
  for _, name in ipairs(M.servers) do
    local server = lspconfig[name]
    if server then
      server.setup(server_config or {})
    end
  end
end

-- setup_treesitter(ts_config?)
-- Configures nvim-treesitter for a no-internet target. Merges ts_config with
-- ferry's safe defaults: auto_install disabled, ensure_installed cleared.
function M.setup_treesitter(ts_config)
  local ok, configs = pcall(require, "nvim-treesitter.configs")
  if not ok then return end
  local cfg = {}
  for k, v in pairs(ts_config or {}) do cfg[k] = v end
  cfg.auto_install     = false
  cfg.ensure_installed = {}
  configs.setup(cfg)
end

-- setup_lazy(plugins, opts?)
-- Wraps require('lazy').setup() with no-internet safe defaults.
-- Forces checker.enabled = false and install.missing = false.
function M.setup_lazy(plugins, opts)
  local ok, lazy = pcall(require, "lazy")
  if not ok then return end
  local cfg = {}
  for k, v in pairs(opts or {}) do cfg[k] = v end
  cfg.checker = vim.tbl_deep_extend("force", cfg.checker or {}, { enabled = false })
  cfg.install = vim.tbl_deep_extend("force", cfg.install or {}, { missing = false })
  lazy.setup(plugins, cfg)
end

return M
FERRY_MODULE_EOF

`, serverList, parserList, formatterTable, linterTable)
}

// nvimFiletype maps a registry language name to the neovim filetype string.
// Most names are identical; only the exceptions are listed.
func nvimFiletype(lang string) string {
	m := map[string]string{
		"bash":       "sh",
		"csharp":     "cs",
		"dockerfile": "dockerfile",
	}
	if ft, ok := m[lang]; ok {
		return ft
	}
	return lang
}

// conformName maps a formatter binary name to the name conform.nvim uses.
// Conform uses underscores and sometimes shortened names for some tools.
func conformName(name string) string {
	m := map[string]string{
		"clang-format":       "clang_format",
		"php-cs-fixer":       "php_cs_fixer",
		"dotnet-csharpier":   "csharpier",
		"google-java-format": "google_java_format",
		"mix format":         "mix",
		"zig fmt":            "zig",
	}
	if v, ok := m[name]; ok {
		return v
	}
	return name
}

// nvimLintName maps a linter binary name to the name nvim-lint uses.
func nvimLintName(name string) string {
	m := map[string]string{
		"clang-tidy": "clangtidy",
	}
	if v, ok := m[name]; ok {
		return v
	}
	return name
}
