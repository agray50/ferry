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
	p := func(parts ...string) string {
		return filepath.Join(append([]string{home}, parts...)...)
	}

	candidates := []ConfigCandidate{
		// ── Shell ─────────────────────────────────────────────────────────
		{Path: p(".zshrc"), Reason: "zsh config"},
		{Path: p(".bashrc"), Reason: "bash config"},
		{Path: p(".bash_profile"), Reason: "bash login config"},
		{Path: p(".profile"), Reason: "POSIX shell profile"},
		{Path: p(".config", "fish", "config.fish"), Reason: "fish shell config"},
		{Path: p(".zsh_aliases"), Reason: "zsh aliases"},
		{Path: p(".aliases"), Reason: "shell aliases"},
		{Path: p(".exports"), Reason: "shell exports"},

		// ── Shell prompts & themes ─────────────────────────────────────────
		{Path: p(".p10k.zsh"), Reason: "powerlevel10k theme"},
		{Path: p(".config", "starship.toml"), Reason: "starship prompt"},
		{Path: p(".oh-my-zsh", "custom", "themes"), Reason: "oh-my-zsh custom themes"},

		// ── Terminal multiplexers ──────────────────────────────────────────
		{Path: p(".tmux.conf"), Reason: "tmux config"},
		{Path: p(".config", "tmux", "tmux.conf"), Reason: "tmux config (XDG)"},
		{Path: p(".config", "zellij", "config.kdl"), Reason: "zellij config"},

		// ── Terminal emulators ─────────────────────────────────────────────
		{Path: p(".config", "alacritty", "alacritty.toml"), Reason: "alacritty config"},
		{Path: p(".config", "alacritty", "alacritty.yml"), Reason: "alacritty config (legacy)"},
		{Path: p(".config", "kitty", "kitty.conf"), Reason: "kitty config"},
		{Path: p(".config", "wezterm", "wezterm.lua"), Reason: "wezterm config"},
		{Path: p(".config", "ghostty", "config"), Reason: "ghostty config"},

		// ── Git ────────────────────────────────────────────────────────────
		{Path: p(".gitconfig"), Reason: "git config"},
		{Path: p(".gitignore_global"), Reason: "global gitignore"},
		{Path: p(".gitignore"), Reason: "global gitignore (home)"},
		{
			Path:      p(".config", "gh", "hosts.yml"),
			Sensitive: true,
			Reason:    "GitHub CLI auth token",
		},

		// ── Editor ─────────────────────────────────────────────────────────
		{Path: p(".editorconfig"), Reason: "editor config"},
		{Path: p(".config", "lazygit", "config.yml"), Reason: "lazygit config"},

		// ── CLI tools ─────────────────────────────────────────────────────
		{Path: p(".config", "bat", "config"), Reason: "bat config"},
		{Path: p(".config", "btop", "btop.conf"), Reason: "btop config"},
		{Path: p(".config", "delta"), Reason: "delta (git diff) config"},
		{Path: p(".config", "lf", "lfrc"), Reason: "lf file manager config"},
		{Path: p(".config", "yazi"), Reason: "yazi file manager config"},
		{Path: p(".config", "navi", "config.yaml"), Reason: "navi cheatsheet config"},
		{Path: p(".config", "direnv", "direnv.toml"), Reason: "direnv config"},

		// ── Language / build tools ─────────────────────────────────────────
		{Path: p(".npmrc"), Reason: "npm config"},
		{Path: p(".config", "pypoetry", "config.toml"), Reason: "poetry config"},
		{Path: p(".cargo", "config.toml"), Reason: "cargo (Rust) config"},
		{Path: p("go", "env"), Reason: "go env overrides"},

		// ── Sensitive / credentials ────────────────────────────────────────
		{
			Path:      p(".netrc"),
			Sensitive: true,
			Reason:    "plaintext credentials",
		},
		{
			Path:      p(".aws", "credentials"),
			Sensitive: true,
			Reason:    "AWS access keys",
		},
		{
			Path:      p(".aws", "config"),
			Reason:    "AWS CLI config (regions, profiles)",
		},
		{
			Path:      p(".kube", "config"),
			Sensitive: true,
			Reason:    "Kubernetes cluster credentials",
		},
		{
			Path:      p(".docker", "config.json"),
			Sensitive: true,
			Reason:    "Docker registry auth",
		},
		{
			Path:      p(".ssh", "config"),
			Sensitive: true,
			Reason:    "SSH host configuration",
		},
		{
			Path:      p(".cargo", "credentials.toml"),
			Sensitive: true,
			Reason:    "crates.io publish token",
		},
		{
			Path:      p(".config", "pypoetry", "auth.toml"),
			Sensitive: true,
			Reason:    "poetry package registry auth",
		},
	}

	// filter to only existing files/dirs
	var found []ConfigCandidate
	for _, c := range candidates {
		if _, err := os.Stat(c.Path); err == nil {
			found = append(found, c)
		}
	}
	return found
}
