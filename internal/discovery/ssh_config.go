package discovery

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// sshConfigPathOverride is used by tests to redirect reads/writes to a temp file.
var sshConfigPathOverride string

// SSHHost is a single entry from ~/.ssh/config.
type SSHHost struct {
	Alias        string
	Hostname     string
	User         string
	Port         int
	IdentityFile string
	FerryManaged bool
}

func resolvedSSHConfigPath() (string, error) {
	if sshConfigPathOverride != "" {
		return sshConfigPathOverride, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ssh", "config"), nil
}

// ParseSSHConfig reads and parses ~/.ssh/config.
// Returns nil slice (not error) if the file does not exist.
func ParseSSHConfig() ([]SSHHost, error) {
	path, err := resolvedSSHConfigPath()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening ssh config: %w", err)
	}
	defer f.Close()

	var hosts []SSHHost
	var current *SSHHost
	pendingFerryManaged := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}
		if trimmed == "# ferry managed" {
			pendingFerryManaged = true
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			pendingFerryManaged = false
			continue
		}

		parts := strings.Fields(trimmed)
		if len(parts) < 2 {
			continue
		}
		key := strings.ToLower(parts[0])
		val := strings.Join(parts[1:], " ")

		switch key {
		case "host":
			if current != nil {
				hosts = append(hosts, *current)
			}
			current = &SSHHost{
				Alias:        val,
				FerryManaged: pendingFerryManaged,
			}
			pendingFerryManaged = false
		case "hostname":
			if current != nil {
				current.Hostname = val
			}
		case "user":
			if current != nil {
				current.User = val
			}
		case "port":
			if current != nil {
				if p, err := strconv.Atoi(val); err == nil {
					current.Port = p
				}
			}
		case "identityfile":
			if current != nil {
				current.IdentityFile = val
			}
		}
	}
	if current != nil {
		hosts = append(hosts, *current)
	}
	return hosts, scanner.Err()
}

// AppendSSHHost appends a ferry-managed Host entry to ~/.ssh/config.
// Creates ~/.ssh/config (and ~/.ssh/) if they do not exist.
func AppendSSHHost(h SSHHost) error {
	path, err := resolvedSSHConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating .ssh dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("opening ssh config for append: %w", err)
	}
	defer f.Close()

	port := h.Port
	if port == 0 {
		port = 22
	}
	ident := h.IdentityFile
	if ident == "" {
		ident = "~/.ssh/id_ed25519"
	}
	_, err = fmt.Fprintf(f,
		"\n# ferry managed\nHost %s\n    HostName %s\n    User %s\n    Port %d\n    IdentityFile %s\n",
		h.Alias, h.Hostname, h.User, port, ident,
	)
	return err
}

// RemoveSSHHost removes the ferry-managed Host block for alias from ~/.ssh/config.
// Returns nil if the alias is not found (idempotent).
// Returns an error if the alias exists but is NOT ferry managed.
func RemoveSSHHost(alias string) error {
	path, err := resolvedSSHConfigPath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading ssh config: %w", err)
	}

	lines := strings.Split(string(data), "\n")

	targetLine := -1
	isFerryManaged := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		parts := strings.Fields(trimmed)
		if len(parts) >= 2 && strings.EqualFold(parts[0], "host") && parts[1] == alias {
			targetLine = i
			for j := i - 1; j >= 0; j-- {
				t := strings.TrimSpace(lines[j])
				if t == "" {
					continue
				}
				if t == "# ferry managed" {
					isFerryManaged = true
				}
				break
			}
			break
		}
	}

	if targetLine == -1 {
		return nil
	}
	if !isFerryManaged {
		return fmt.Errorf("host %q exists in ssh config but is not ferry managed — refusing to remove", alias)
	}

	blockStart := targetLine
	for blockStart > 0 {
		t := strings.TrimSpace(lines[blockStart-1])
		if t == "" {
			blockStart--
			continue
		}
		if t == "# ferry managed" {
			blockStart--
			break
		}
		break
	}

	blockEnd := targetLine + 1
	for blockEnd < len(lines) {
		t := strings.TrimSpace(lines[blockEnd])
		parts := strings.Fields(t)
		if len(parts) >= 1 && strings.EqualFold(parts[0], "host") {
			break
		}
		blockEnd++
	}

	result := make([]string, 0, len(lines)-(blockEnd-blockStart))
	result = append(result, lines[:blockStart]...)
	result = append(result, lines[blockEnd:]...)
	return os.WriteFile(path, []byte(strings.Join(result, "\n")), 0600)
}
