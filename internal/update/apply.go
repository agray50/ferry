package update

import (
	"fmt"
	"strings"

	"github.com/anthropics/ferry/internal/store"
)

// GenerateApplyScript generates a sh script that applies a diff on the target.
func GenerateApplyScript(diff store.DiffResult) string {
	var b strings.Builder

	b.WriteString("#!/bin/sh\nset -e\n\n")
	b.WriteString("STORE=\"$HOME/.ferry/store\"\n\n")

	// extract new and changed components
	for _, c := range append(diff.New, diff.Changed...) {
		dest := expandHome(c.InstallPath)
		b.WriteString(fmt.Sprintf("# %s\n", c.ID))
		b.WriteString(fmt.Sprintf("mkdir -p %q\n", dest))
		b.WriteString(fmt.Sprintf("tar --zstd -xf \"$STORE/%s.tar.zst\" -C %q\n\n", c.Hash, dest))
	}

	// remove components no longer in profile
	for _, c := range diff.Removed {
		dest := expandHome(c.InstallPath)
		b.WriteString(fmt.Sprintf("# remove: %s\n", c.ID))
		b.WriteString(fmt.Sprintf("rm -rf %q\n\n", dest))
	}

	// fix permissions
	b.WriteString("chmod +x \"$HOME/.local/bin\"/* 2>/dev/null || true\n\n")

	// update manifest
	b.WriteString("cp \"$HOME/.ferry/incoming/manifest.json\" \"$HOME/.ferry/manifest.json\" 2>/dev/null || true\n")
	b.WriteString("echo \"ferry: update applied\"\n")

	return b.String()
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		return "$HOME/" + path[2:]
	}
	return path
}
