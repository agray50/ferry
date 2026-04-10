package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnsureFerryDir creates ~/.ferry/ and subdirectories if they do not exist.
func EnsureFerryDir() error {
	base := FerryDir()
	if err := os.MkdirAll(base, 0700); err != nil {
		return fmt.Errorf("creating ~/.ferry: %w", err)
	}
	for _, sub := range []string{"store", "bundles", "profiles", "logs"} {
		if err := os.MkdirAll(filepath.Join(base, sub), 0700); err != nil {
			return fmt.Errorf("creating ~/.ferry/%s: %w", sub, err)
		}
	}
	return nil
}
