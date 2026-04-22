package discovery

import (
	"os"
	"path/filepath"
)

// ShellDiscovery holds results of shell environment scanning.
type ShellDiscovery struct {
	Type              string
	ConfigPath        string
	ThemeDetected     string
	ThemeConfigPath   string
	PluginManager     string
	PluginManagerPath string
}

// DiscoverShell scans the local machine for shell configuration.
func DiscoverShell() ShellDiscovery {
	d := ShellDiscovery{}
	home, _ := os.UserHomeDir()

	// detect shell type
	zshrc := filepath.Join(home, ".zshrc")
	if _, err := os.Stat(zshrc); err == nil {
		d.Type = "zsh"
		d.ConfigPath = zshrc
	} else if _, err := os.Stat(filepath.Join(home, ".bashrc")); err == nil {
		d.Type = "bash"
		d.ConfigPath = filepath.Join(home, ".bashrc")
	}

	// detect theme
	p10k := filepath.Join(home, ".p10k.zsh")
	starship := filepath.Join(home, ".config", "starship.toml")
	pure := filepath.Join(home, ".zsh", "pure", "pure.zsh")
	switch {
	case fileExists(p10k):
		d.ThemeDetected = "p10k"
		d.ThemeConfigPath = p10k
	case fileExists(starship):
		d.ThemeDetected = "starship"
		d.ThemeConfigPath = starship
	case fileExists(pure):
		d.ThemeDetected = "pure"
		d.ThemeConfigPath = ""
	case fileExists(filepath.Join(home, ".oh-my-zsh")):
		d.ThemeDetected = "oh-my-zsh"
	}

	// detect plugin manager
	zinit := filepath.Join(home, ".local", "share", "zinit")
	omz := filepath.Join(home, ".oh-my-zsh")
	zplug := filepath.Join(home, ".zplug")
	switch {
	case fileExists(zinit):
		d.PluginManager = "zinit"
		d.PluginManagerPath = zinit
	case fileExists(omz):
		d.PluginManager = "oh-my-zsh"
		d.PluginManagerPath = omz
	case fileExists(zplug):
		d.PluginManager = "zplug"
		d.PluginManagerPath = zplug
	}

	return d
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
