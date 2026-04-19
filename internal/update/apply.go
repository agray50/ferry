package update

import (
	"fmt"
	"strings"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/store"
)

// GenerateApplyScript generates a sh script that applies a diff on the target.
func GenerateApplyScript(diff store.DiffResult) string {
	var b strings.Builder

	b.WriteString("#!/bin/sh\nset -e\n\n")
	b.WriteString("STORE=\"$HOME/.ferry/store\"\n\n")

	// install_component: extract directory component to a temp dir, then swap into place.
	// install_file: extract single-binary component directly to dest path.
	b.WriteString(`install_component() {
  local hash="$1"
  local dest="$2"
  local tmp="${dest}.ferry-new"
  rm -rf "$tmp"
  mkdir -p "$tmp"
  tar --zstd -xf "$STORE/${hash}.tar.zst" -C "$tmp" || { rm -rf "$tmp"; return 1; }
  rm -rf "${dest:?}"
  mv "$tmp" "$dest"
}

install_file() {
  local hash="$1"
  local dest="$2"
  local tmpdir
  tmpdir="$(mktemp -d)"
  tar --zstd -xf "$STORE/${hash}.tar.zst" -C "$tmpdir" || { rm -rf "$tmpdir"; return 1; }
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

	// extract new and changed components
	for _, c := range append(diff.New, diff.Changed...) {
		dest := config.ExpandHome(c.InstallPath)
		b.WriteString(fmt.Sprintf("# %s\n", c.ID))
		if strings.HasSuffix(c.InstallPath, "/") {
			b.WriteString(fmt.Sprintf("install_component %q %q\n\n", c.Hash, dest))
		} else {
			b.WriteString(fmt.Sprintf("install_file %q %q\n\n", c.Hash, dest))
		}
	}

	// remove components no longer in profile
	for _, c := range diff.Removed {
		dest := config.ExpandHome(c.InstallPath)
		b.WriteString(fmt.Sprintf("# remove: %s\n", c.ID))
		b.WriteString(fmt.Sprintf("rm -rf %q\n\n", dest))
	}

	// fix permissions
	b.WriteString("chmod +x \"$HOME/.local/bin\"/* 2>/dev/null || true\n\n")

	// update env.sh (runtime PATH entries) and manifest atomically
	b.WriteString("cp \"$HOME/.ferry/incoming/env.sh\" \"$HOME/.ferry/env.sh\" 2>/dev/null || true\n")
	b.WriteString("cp \"$HOME/.ferry/incoming/manifest.json\" \"$HOME/.ferry/manifest.json\" 2>/dev/null || true\n")
	b.WriteString("echo \"ferry: update applied\"\n")

	return b.String()
}

