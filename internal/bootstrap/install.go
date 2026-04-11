package bootstrap

import (
	"fmt"
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

backup_if_exists() {
  [ -e "$1" ] && cp -r "$1" "$1.ferry-backup-$(date +%s)" || true
}

extract_component() {
  local hash="$1"
  local dest="$2"
  local src="$STORE_DIR/${hash}.tar.zst"
  rm -rf "${dest:?}"
  mkdir -p "$dest"
  tar --zstd -xf "$src" -C "$dest"
}

`)

	// Encrypted component handling
	hasEncrypted := false
	for _, c := range m.Components {
		if c.Encrypted {
			hasEncrypted = true
			break
		}
	}

	b.WriteString(`# Read age key from stdin (blank line sent when no encrypted components).
IFS= read -r FERRY_AGE_KEY || true

`)

	if hasEncrypted {
		b.WriteString(`decrypt_component() {
  local src="$1"
  local dst="$2"
  printf '%s' "$FERRY_AGE_KEY" | age --decrypt --identity /dev/stdin "$src" > "$dst"
  rm -f "$src"
}

`)
	}

	// Extract each component
	for _, c := range m.Components {
		b.WriteString(fmt.Sprintf("# %s\n", c.ID))
		installPath := expandHome(c.InstallPath)
		b.WriteString(fmt.Sprintf("backup_if_exists %q\n", installPath))
		if c.Encrypted {
			b.WriteString(fmt.Sprintf("decrypt_component \"$STORE_DIR/%s.tar.zst.age\" \"$STORE_DIR/%s.tar.zst\"\n", c.Hash, c.Hash))
		}
		b.WriteString(fmt.Sprintf("extract_component %q %q\n", c.Hash, installPath))
		if c.BinSymlink != "" {
			binLink := expandHome(c.BinSymlink)
			b.WriteString(fmt.Sprintf("ln -sf %q/bin/nvim %q\n", installPath, binLink))
		}
		b.WriteString("\n")
	}

	// Shell PATH init — one idempotent block per runtime.
	// Points at ~/.ferry/runtimes/; no version manager setup.
	if len(langs) > 0 {
		b.WriteString(`# Runtime PATH setup.
# Each line is guarded by grep -q to prevent duplication on re-install.
`)
		shellrc := `"$HOME/.zshrc"`
		b.WriteString(fmt.Sprintf("SHELL_RC=%s\n", shellrc))
		b.WriteString(`[ -f "$HOME/.bashrc" ] && SHELL_RC="$HOME/.bashrc"
[ -n "$SHELL" ] && echo "$SHELL" | grep -q zsh && SHELL_RC="$HOME/.zshrc"

`)
		for _, rl := range langs {
			if rl.Runtime == nil || len(rl.Runtime.ShellInit) == 0 {
				continue
			}
			b.WriteString(fmt.Sprintf("# %s\n", rl.Language.Name))
			for _, line := range rl.Runtime.ShellInit {
				key := line
				if idx := strings.Index(line, ".ferry/runtimes/"); idx >= 0 {
					end := strings.Index(line[idx+16:], "/")
					if end > 0 {
						key = line[idx : idx+16+end]
					}
				}
				b.WriteString(fmt.Sprintf("grep -q %q \"$SHELL_RC\" 2>/dev/null || printf '\\n%s\\n' >> \"$SHELL_RC\"\n",
					key, line))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString(`# Set permissions.
chmod +x "$BIN_DIR"/* 2>/dev/null || true
chmod -R +x "$HOME/.ferry/runtimes/"*/bin/ 2>/dev/null || true

# Write manifest.
cp "$FERRY_DIR/incoming/manifest.json" "$FERRY_DIR/manifest.json" 2>/dev/null || true

`)

	b.WriteString("unset FERRY_AGE_KEY\n\n")
	b.WriteString(`echo "ferry: install complete"
`)

	return b.String(), nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		return "$HOME/" + path[2:]
	}
	return path
}
