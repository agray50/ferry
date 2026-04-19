package config

import (
	"os"
	"path/filepath"
	"strings"
)

func FerryDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ferry")
}

func TargetsFilePath() string {
	return filepath.Join(FerryDir(), "targets.json")
}

// ExpandHome replaces a leading ~/ with $HOME/ so the result is suitable for
// embedding in shell scripts where variable expansion will occur.
func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		return "$HOME/" + path[2:]
	}
	return path
}

func StoreDir() string {
	return filepath.Join(FerryDir(), "store")
}

func BundlesDir() string {
	return filepath.Join(FerryDir(), "bundles")
}

func LockFilePath() string {
	return "ferry.lock"
}
