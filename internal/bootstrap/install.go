package bootstrap

import (
	"fmt"
	"strings"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/store"
)

// GenerateInstallScript generates a POSIX sh install.sh for the given manifest and lock.
// The age private key (if needed) is read from the first line of stdin — it is never
// written to disk on the target.
func GenerateInstallScript(m *store.Manifest, lock *config.LockFile) (string, error) {
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
  # Remove existing content so stale files from old versions don't survive.
  rm -rf "${dest:?}"
  mkdir -p "$dest"
  tar --zstd -xf "$src" -C "$dest"
}

`)

	// Determine if any components are encrypted.
	hasEncrypted := false
	for _, c := range m.Components {
		if c.Encrypted {
			hasEncrypted = true
			break
		}
	}

	// Always read one line from stdin. When encrypted components are present
	// that line is the age private key (delivered securely over the SSH
	// connection, never stored to disk). When there are no encrypted
	// components the caller sends a blank line so this read does not block.
	b.WriteString(`# Read age key from stdin (delivered via SSH, never written to disk).
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

	// Inject PATH.
	b.WriteString(`# Ensure ~/.local/bin is in PATH.
grep -q '\.local/bin' "$HOME/.zshrc" 2>/dev/null || \
  printf '\nexport PATH="$HOME/.local/bin:$PATH"\n' >> "$HOME/.zshrc" || true

`)

	// Extract each component.
	for _, c := range m.Components {
		b.WriteString(fmt.Sprintf("# %s\n", c.ID))
		installPath := expandHome(c.InstallPath)
		b.WriteString(fmt.Sprintf("backup_if_exists %q\n", installPath))
		if c.Encrypted {
			b.WriteString(fmt.Sprintf("decrypt_component \"$STORE_DIR/%s.tar.zst.age\" \"$STORE_DIR/%s.tar.zst\"\n",
				c.Hash, c.Hash))
		}
		b.WriteString(fmt.Sprintf("extract_component %q %q\n", c.Hash, installPath))
		if c.BinSymlink != "" {
			binLink := expandHome(c.BinSymlink)
			b.WriteString(fmt.Sprintf("ln -sf %q/bin/nvim %q\n", installPath, binLink))
		}
		b.WriteString("\n")
	}

	// Set permissions.
	b.WriteString(`# Set permissions.
chmod +x "$BIN_DIR"/* 2>/dev/null || true
chmod -R +x "$HOME/.local/share/nvim/mason/bin/" 2>/dev/null || true

`)

	// Write manifest.
	b.WriteString(`# Write manifest.
cp "$FERRY_DIR/incoming/manifest.json" "$FERRY_DIR/manifest.json" 2>/dev/null || true

`)

	// Clear age key from memory.
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
