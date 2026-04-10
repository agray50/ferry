package discovery

import (
	"os"
	"path/filepath"
)

// ConfigCandidate is a config file to suggest including.
type ConfigCandidate struct {
	Path      string
	Sensitive bool
	Reason    string
}

// KnownConfigs returns a list of common config file candidates.
func KnownConfigs() []ConfigCandidate {
	home, _ := os.UserHomeDir()
	candidates := []ConfigCandidate{
		{Path: filepath.Join(home, ".tmux.conf")},
		{Path: filepath.Join(home, ".config", "starship.toml")},
		{Path: filepath.Join(home, ".gitconfig")},
		{
			Path:      filepath.Join(home, ".config", "gh", "hosts.yml"),
			Sensitive: true,
			Reason:    "contains GitHub authentication tokens",
		},
		{
			Path:      filepath.Join(home, ".netrc"),
			Sensitive: true,
			Reason:    "contains plaintext credentials",
		},
		{
			Path:      filepath.Join(home, ".aws", "credentials"),
			Sensitive: true,
			Reason:    "contains AWS access keys — suggest encrypt",
		},
		{
			Path:      filepath.Join(home, ".ssh", "config"),
			Sensitive: true,
			Reason:    "contains SSH host configuration",
		},
	}

	// filter to only existing files
	var found []ConfigCandidate
	for _, c := range candidates {
		if _, err := os.Stat(c.Path); err == nil {
			found = append(found, c)
		}
	}
	return found
}
