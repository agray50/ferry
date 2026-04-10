package ssh

import (
	"fmt"
	"strconv"
	"strings"
)

// RemoteEnv holds detected properties of a remote machine.
type RemoteEnv struct {
	Arch              string
	Libc              string // glibc | musl
	OS                string // linux
	DiskFree          int64  // bytes available in home directory
	ZshVersion        string
	InstalledPackages []string
}

// DetectRemoteEnv runs detection commands over SSH and returns results.
func DetectRemoteEnv(c *Client) (RemoteEnv, error) {
	script := `
echo "ARCH=$(uname -m)"
echo "LIBC=$(ldd --version 2>&1 | head -1)"
echo "DISKFREE=$(df -k "$HOME" 2>/dev/null | tail -1 | awk '{print $4}')"
echo "ZSH=$(zsh --version 2>/dev/null | head -1)"
for pkg in git curl tmux zsh; do
  which "$pkg" >/dev/null 2>&1 && echo "PKG=$pkg"
done
`
	stdout, _, code, err := c.Run(script)
	if err != nil {
		return RemoteEnv{}, fmt.Errorf("DetectRemoteEnv: %w", err)
	}
	if code != 0 {
		return RemoteEnv{}, fmt.Errorf("DetectRemoteEnv: non-zero exit %d", code)
	}

	env := RemoteEnv{OS: "linux"}
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch k {
		case "ARCH":
			env.Arch = normaliseArch(v)
		case "LIBC":
			env.Libc = detectLibc(v)
		case "DISKFREE":
			kb, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			env.DiskFree = kb * 1024
		case "ZSH":
			env.ZshVersion = parseZshVersion(v)
		case "PKG":
			env.InstalledPackages = append(env.InstalledPackages, strings.TrimSpace(v))
		}
	}
	return env, nil
}

func normaliseArch(uname string) string {
	switch strings.TrimSpace(uname) {
	case "x86_64", "amd64":
		return "x86_64"
	case "aarch64", "arm64":
		return "arm64"
	default:
		return strings.TrimSpace(uname)
	}
}

func detectLibc(lddOutput string) string {
	lower := strings.ToLower(lddOutput)
	if strings.Contains(lower, "musl") {
		return "musl"
	}
	return "glibc"
}

func parseZshVersion(zshOutput string) string {
	// "zsh 5.9 (x86_64-ubuntu-linux-gnu)"
	parts := strings.Fields(zshOutput)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
