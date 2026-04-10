package discovery

import (
	"github.com/anthropics/ferry/internal/crypto"
)

// SecretMatch re-exports from crypto package for discovery consumers.
type SecretMatch = crypto.SecretMatch

// ScanForSecrets scans a list of file paths for secret patterns.
func ScanForSecrets(paths []string) []SecretMatch {
	results, _ := crypto.ScanFiles(paths)
	var all []SecretMatch
	for _, matches := range results {
		all = append(all, matches...)
	}
	return all
}
