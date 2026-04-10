package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/crypto"
	"github.com/anthropics/ferry/internal/discovery"
)

// InitWizardResult is the result of running the init wizard.
type InitWizardResult struct {
	LockFile *config.LockFile
	Aborted  bool
}

// RunInitWizard runs the interactive ferry init TUI.
func RunInitWizard(existing *config.LockFile) (*InitWizardResult, error) {
	// Run discovery
	fmt.Println("  scanning environment...")
	nvim := discovery.DiscoverNvim()
	shell := discovery.DiscoverShell()
	langs := discovery.DiscoverLanguages()
	cliTools := discovery.DiscoverCLITools()
	configs := discovery.KnownConfigs()

	lf := config.DefaultLockFile()
	if existing != nil {
		lf = existing
	}

	// Step 1: Neovim plugins
	if nvim.Found {
		items := pluginItems(nvim.Plugins, existing)
		m := NewMultiSelect("Step 1: Neovim plugins — select plugins to include", items)
		p := tea.NewProgram(m)
		final, err := p.Run()
		if err != nil {
			return nil, err
		}
		mm := final.(MultiSelectModel)
		if mm.aborted {
			return &InitWizardResult{Aborted: true}, nil
		}
		lf.Nvim.Version = nvim.Version

		// build plugin list from selections
		var plugins []string
		for _, item := range mm.Items {
			if item.Selected {
				plugins = append(plugins, item.Value)
			}
		}
		if lf.Profiles == nil {
			lf.Profiles = config.DefaultLockFile().Profiles
		}
		for name, prof := range lf.Profiles {
			prof.Plugins = plugins
			lf.Profiles[name] = prof
		}
	}

	// Step 2: Languages
	{
		items := languageItems(langs, existing)
		m := NewMultiSelect("Step 2: Languages — select languages to support", items)
		p := tea.NewProgram(m)
		final, err := p.Run()
		if err != nil {
			return nil, err
		}
		mm := final.(MultiSelectModel)
		if mm.aborted {
			return &InitWizardResult{Aborted: true}, nil
		}
		var enabled []string
		for _, item := range mm.Items {
			if item.Selected {
				enabled = append(enabled, item.Value)
			}
		}
		lf.Languages.Enabled = enabled
	}

	// Step 3: Shell config
	if shell.Type != "" {
		lf.Shell.Type = shell.Type
		lf.Shell.Config = shell.ConfigPath
		lf.Shell.Theme = shell.ThemeDetected
		lf.Shell.ThemeConfig = shell.ThemeConfigPath
		lf.Shell.PluginManager = shell.PluginManager
		lf.Shell.PluginManagerPath = shell.PluginManagerPath
	}

	// Step 4: CLI tools
	{
		items := cliItems(cliTools, existing)
		m := NewMultiSelect("Step 4: CLI tools — select tools to bundle", items)
		p := tea.NewProgram(m)
		final, err := p.Run()
		if err != nil {
			return nil, err
		}
		mm := final.(MultiSelectModel)
		if mm.aborted {
			return &InitWizardResult{Aborted: true}, nil
		}
		lf.CLI = make(map[string]string)
		for _, item := range mm.Items {
			if item.Selected {
				lf.CLI[item.Value] = item.Extra
			}
		}
	}

	// Step 5: Config files
	{
		items := configItems(configs, existing)
		m := NewMultiSelect("Step 5: Config files — select configs to bundle", items)
		p := tea.NewProgram(m)
		final, err := p.Run()
		if err != nil {
			return nil, err
		}
		mm := final.(MultiSelectModel)
		if mm.aborted {
			return &InitWizardResult{Aborted: true}, nil
		}
		// Build a lookup from path → candidate so selection order doesn't matter.
		candidateByPath := make(map[string]discovery.ConfigCandidate, len(configs))
		for _, c := range configs {
			candidateByPath[c.Path] = c
		}
		lf.Configs = nil
		for _, item := range mm.Items {
			if !item.Selected {
				continue
			}
			cf, ok := candidateByPath[item.Value]
			if !ok {
				continue
			}
			home := getUserHome()
			target := "~/" + strings.TrimPrefix(cf.Path, home+"/")
			lf.Configs = append(lf.Configs, config.ConfigFile{
				Source:  cf.Path,
				Target:  target,
				Encrypt: cf.Sensitive,
			})
		}
	}

	// Step 6: Secret scanning
	var configPaths []string
	for _, cf := range lf.Configs {
		configPaths = append(configPaths, cf.Source)
	}
	if len(configPaths) > 0 {
		secrets := discovery.ScanForSecrets(configPaths)
		if len(secrets) > 0 {
			fmt.Printf("\n  %s\n", warningStyle.Render("⚠ potential secrets detected:"))
			for _, s := range secrets {
				fmt.Printf("    %s:%d  %s\n", s.FilePath, s.Line, s.Preview)
			}
			fmt.Println()
		}
	}

	// Step 7: Encryption setup
	hasSensitive := false
	for _, cf := range lf.Configs {
		if cf.Encrypt {
			hasSensitive = true
			break
		}
	}
	if hasSensitive && !crypto.KeyExists() {
		fmt.Println("  sensitive files selected — generating age keypair...")
		if err := crypto.GenerateKeypair(); err != nil {
			return nil, fmt.Errorf("generating keypair: %w", err)
		}
		fmt.Println()
		fmt.Println(warningStyle.Render("  ⚠ important: back up your age private key"))
		fmt.Println("    ~/.ferry/key.txt")
		fmt.Println("    this key decrypts sensitive components in your bundles.")
		fmt.Println("    losing it means losing access to encrypted configs.")
		fmt.Println("    suggested: store in a password manager or secure backup.")
		fmt.Println()
	}

	return &InitWizardResult{LockFile: lf}, nil
}

func getUserHome() string {
	h, _ := os.UserHomeDir()
	return h
}

func pluginItems(plugins []discovery.PluginInfo, existing *config.LockFile) []Item {
	existingSet := map[string]bool{}
	if existing != nil {
		for _, prof := range existing.Profiles {
			for _, p := range prof.Plugins {
				existingSet[p] = true
			}
		}
	}

	var items []Item
	for _, p := range plugins {
		extra := ""
		if p.SizeBytes > 0 {
			extra = formatBytes(p.SizeBytes)
		}
		isNew := existing != nil && !existingSet[p.Name]
		items = append(items, Item{
			Label:    p.Name,
			Value:    p.Name,
			Extra:    extra,
			IsNew:    isNew,
			Selected: existingSet[p.Name] || existing == nil,
		})
	}
	return items
}

func languageItems(langs []discovery.LanguageDiscovery, existing *config.LockFile) []Item {
	existingSet := map[string]bool{}
	if existing != nil {
		for _, l := range existing.Languages.Enabled {
			existingSet[l] = true
		}
	}

	var items []Item
	for _, l := range langs {
		isNew := existing != nil && !existingSet[l.Name]
		items = append(items, Item{
			Label:    l.Name,
			Value:    l.Name,
			IsNew:    isNew,
			Selected: existingSet[l.Name] || existing == nil,
		})
	}
	return items
}

func cliItems(tools []discovery.CLITool, existing *config.LockFile) []Item {
	existingSet := map[string]bool{}
	if existing != nil {
		for name := range existing.CLI {
			existingSet[name] = true
		}
	}

	var items []Item
	for _, t := range tools {
		isNew := existing != nil && !existingSet[t.Name]
		items = append(items, Item{
			Label:    t.Name,
			Value:    t.Name,
			Extra:    t.Version,
			IsNew:    isNew,
			Selected: existingSet[t.Name] || existing == nil,
		})
	}
	return items
}

func configItems(candidates []discovery.ConfigCandidate, existing *config.LockFile) []Item {
	existingSet := map[string]bool{}
	if existing != nil {
		for _, cf := range existing.Configs {
			existingSet[cf.Source] = true
		}
	}

	var items []Item
	for _, c := range candidates {
		label := c.Path
		if c.Sensitive {
			label += " 🔒"
		}
		isNew := existing != nil && !existingSet[c.Path]
		items = append(items, Item{
			Label:    label,
			Value:    c.Path,
			Extra:    c.Reason,
			IsNew:    isNew,
			Selected: existingSet[c.Path] || existing == nil,
		})
	}
	return items
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "KMGTPE"[exp])
}
