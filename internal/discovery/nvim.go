package discovery

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// NvimDiscovery holds results of nvim environment scanning.
type NvimDiscovery struct {
	Found      bool
	Version    string
	ConfigPath string
	LazyLock   string
	Plugins    []PluginInfo
}

// PluginInfo holds metadata about a single plugin.
type PluginInfo struct {
	Name      string
	Version   string
	HasBuild  bool
	BuildCmd  string
	SizeBytes int64
}

// DiscoverNvim scans the local machine for neovim and its configuration.
func DiscoverNvim() NvimDiscovery {
	d := NvimDiscovery{}

	path, err := exec.LookPath("nvim")
	if err != nil {
		return d
	}

	d.Found = true

	out, err := exec.Command(path, "--version").Output()
	if err == nil {
		lines := strings.SplitN(string(out), "\n", 2)
		if len(lines) > 0 {
			d.Version = strings.TrimPrefix(strings.TrimSpace(lines[0]), "NVIM ")
		}
	}

	home, _ := os.UserHomeDir()
	d.ConfigPath = filepath.Join(home, ".config", "nvim")

	seen := make(map[string]bool)
	var plugins []PluginInfo

	// ── lazy.nvim ─────────────────────────────────────────────────────────
	lazyLock := filepath.Join(d.ConfigPath, "lazy-lock.json")
	if _, err := os.Stat(lazyLock); err == nil {
		d.LazyLock = lazyLock
		for _, p := range parseLazyPlugins(lazyLock, home) {
			if !seen[p.Name] {
				seen[p.Name] = true
				plugins = append(plugins, p)
			}
		}
	}

	// ── packer.nvim ───────────────────────────────────────────────────────
	packerDir := filepath.Join(home, ".local", "share", "nvim", "site", "pack", "packer", "start")
	if entries, err := os.ReadDir(packerDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() || seen[e.Name()] {
				continue
			}
			seen[e.Name()] = true
			size, _ := dirSize(filepath.Join(packerDir, e.Name()))
			plugins = append(plugins, PluginInfo{Name: e.Name(), SizeBytes: size})
		}
	}

	// ── vim-plug ──────────────────────────────────────────────────────────
	plugDir := filepath.Join(home, ".local", "share", "nvim", "plugged")
	if entries, err := os.ReadDir(plugDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() || seen[e.Name()] {
				continue
			}
			seen[e.Name()] = true
			size, _ := dirSize(filepath.Join(plugDir, e.Name()))
			plugins = append(plugins, PluginInfo{Name: e.Name(), SizeBytes: size})
		}
	}

	// ── native packages (pack/*/start/* and pack/*/opt/*) ─────────────────
	packRoot := filepath.Join(home, ".local", "share", "nvim", "site", "pack")
	if groups, err := os.ReadDir(packRoot); err == nil {
		for _, group := range groups {
			if group.Name() == "packer" {
				continue // already handled above
			}
			for _, subdir := range []string{"start", "opt"} {
				dir := filepath.Join(packRoot, group.Name(), subdir)
				entries, err := os.ReadDir(dir)
				if err != nil {
					continue
				}
				for _, e := range entries {
					if !e.IsDir() || seen[e.Name()] {
						continue
					}
					seen[e.Name()] = true
					size, _ := dirSize(filepath.Join(dir, e.Name()))
					plugins = append(plugins, PluginInfo{Name: e.Name(), SizeBytes: size})
				}
			}
		}
	}

	sort.Slice(plugins, func(i, j int) bool { return plugins[i].Name < plugins[j].Name })
	d.Plugins = plugins

	return d
}

type lazyLockEntry struct {
	Branch string `json:"branch"`
	Commit string `json:"commit"`
	Build  string `json:"build"`
}

func parseLazyPlugins(lockPath string, home string) []PluginInfo {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return nil
	}

	var entries map[string]lazyLockEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil
	}

	var plugins []PluginInfo
	for name, entry := range entries {
		info := PluginInfo{
			Name:    name,
			Version: entry.Commit,
		}
		if entry.Build != "" {
			info.HasBuild = true
			info.BuildCmd = entry.Build
		}
		pluginDir := filepath.Join(home, ".local", "share", "nvim", "lazy", name)
		if size, err := dirSize(pluginDir); err == nil {
			info.SizeBytes = size
		}
		plugins = append(plugins, info)
	}
	return plugins
}

func dirSize(path string) (int64, error) {
	var total int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}
