package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/crypto"
	"github.com/anthropics/ferry/internal/discovery"
	"github.com/anthropics/ferry/internal/registry"
)

// registryAllFunc is a variable so tests can inject a stub registry.
var registryAllFunc = func() []registry.Language { return registry.All() }

// ── Preset picker ─────────────────────────────────────────────────────────

type presetPickerModel struct {
	presets []registry.Preset
	cursor  int
	blank   bool // user chose "blank"
	done    bool
	aborted bool
}

func newPresetPickerModel(presets []registry.Preset) presetPickerModel {
	return presetPickerModel{presets: presets}
}

func (m presetPickerModel) Init() tea.Cmd { return nil }

func (m presetPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	total := len(m.presets) + 1 // +1 for blank
	switch km.String() {
	case "ctrl+c":
		m.aborted = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < total-1 {
			m.cursor++
		}
	case "enter":
		if m.cursor == len(m.presets) {
			m.blank = true
		}
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m presetPickerModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Create profile from preset?") + "\n\n")
	for i, p := range m.presets {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("❯ ")
		}
		b.WriteString(fmt.Sprintf("%s%-14s %s  %s\n",
			cursor,
			selectedStyle.Render(p.Name),
			subtleStyle.Render(p.Description),
			subtleStyle.Render(fmt.Sprintf("~%dMB", p.EstimatedMB))))
	}
	// blank option
	blankCursor := "  "
	if m.cursor == len(m.presets) {
		blankCursor = cursorStyle.Render("❯ ")
	}
	b.WriteString(fmt.Sprintf("%s%-14s %s\n", blankCursor, "blank", subtleStyle.Render("start from scratch")))
	b.WriteString(subtleStyle.Render("\n  ↑↓: navigate   enter: choose\n"))
	return b.String()
}

// RunPresetPicker shows the preset picker and returns the chosen preset name, or "" for blank.
// Returns ("", false, nil) for blank; ("", true, nil) if aborted.
func RunPresetPicker() (string, bool, error) {
	presets := registry.Presets()
	m := newPresetPickerModel(presets)
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return "", false, err
	}
	fm := final.(presetPickerModel)
	if fm.aborted {
		return "", true, nil
	}
	if fm.blank || fm.cursor >= len(fm.presets) {
		return "", false, nil
	}
	return fm.presets[fm.cursor].Name, false, nil
}

// ── Size summary ──────────────────────────────────────────────────────────

// SizeLine is one row in the size summary table.
type SizeLine struct {
	Label  string
	SizeMB int
}

// SizeSummary holds computed size breakdown for a profile.
type SizeSummary struct {
	Lines   []SizeLine
	TotalMB int
}

func buildSizeSummary(prof config.ProfileConfig) SizeSummary {
	langMap := make(map[string]registry.Language)
	for _, l := range registryAllFunc() {
		langMap[l.Name] = l
	}

	var s SizeSummary
	for _, lc := range prof.Languages {
		lang, ok := langMap[lc.Name]
		if !ok {
			continue
		}
		size := lang.ApproxSizeMB
		label := lang.Name
		if lc.Tier == "lsp-only" {
			size = lang.ApproxLSPOnlyMB
			label += " (LSP only)"
		}
		s.Lines = append(s.Lines, SizeLine{Label: label, SizeMB: size})
		s.TotalMB += size
	}
	return s
}

func transferTime(totalMB, mbps int) string {
	if mbps <= 0 || totalMB <= 0 {
		return "n/a"
	}
	totalMb := totalMB * 8 // MB to Mb
	seconds := totalMb / mbps
	if seconds < 60 {
		return fmt.Sprintf("~%ds", seconds)
	}
	return fmt.Sprintf("~%dm %ds", seconds/60, seconds%60)
}

func renderSizeSummary(name string, prof config.ProfileConfig) string {
	s := buildSizeSummary(prof)
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Profile: %s — size summary", name)) + "\n\n")
	b.WriteString(fmt.Sprintf("  %-32s %s\n", "COMPONENT", "SIZE"))
	b.WriteString("  " + strings.Repeat("─", 44) + "\n")
	for _, line := range s.Lines {
		b.WriteString(fmt.Sprintf("  %-32s ~%dMB\n", line.Label, line.SizeMB))
	}
	b.WriteString("  " + strings.Repeat("─", 44) + "\n")
	b.WriteString(fmt.Sprintf("  %-32s ~%dMB\n\n", "total (estimated)", s.TotalMB))
	b.WriteString(fmt.Sprintf("  transfer at 10 Mbps:   %s\n", transferTime(s.TotalMB, 10)))
	b.WriteString(fmt.Sprintf("  transfer at 50 Mbps:   %s\n", transferTime(s.TotalMB, 50)))
	b.WriteString(fmt.Sprintf("  transfer at 100 Mbps:  %s\n", transferTime(s.TotalMB, 100)))
	return b.String()
}

// ── Profile wizard ────────────────────────────────────────────────────────

// RunProfileWizard runs the 6-step profile editor for a single profile.
// existing is nil when creating a new profile.
// Returns the completed ProfileConfig, or (nil, true, nil) if aborted.
func RunProfileWizard(profileName string, existing *config.ProfileConfig) (*config.ProfileConfig, bool, error) {
	prof := config.ProfileConfig{}
	if existing != nil {
		prof = *existing
	}

	// Run discovery once
	fmt.Println("  scanning environment...")
	nvim := discovery.DiscoverNvim()
	shell := discovery.DiscoverShell()
	cliTools := discovery.DiscoverCLITools()
	configs := discovery.KnownConfigs()
	allLangs := registryAllFunc()

	// Step 1: Languages
	{
		cfgs, aborted, err := RunLangStep(allLangs, prof.Languages)
		if err != nil {
			return nil, false, err
		}
		if aborted {
			return nil, true, nil
		}
		prof.Languages = cfgs
	}

	// Step 2: Neovim plugins
	if nvim.Found {
		items := pluginItems(nvim.Plugins, prof.Plugins)
		m := NewMultiSelect("Step 2: Neovim plugins — select plugins to include", items)
		p := tea.NewProgram(m)
		final, err := p.Run()
		if err != nil {
			return nil, false, err
		}
		mm := final.(MultiSelectModel)
		if mm.aborted {
			return nil, true, nil
		}
		prof.Plugins = nil
		for _, item := range mm.Items {
			if item.Selected {
				prof.Plugins = append(prof.Plugins, item.Value)
			}
		}
	}

	// Step 3: CLI tools
	{
		items := cliItemsForProfile(cliTools, prof.CLI)
		m := NewMultiSelect("Step 3: CLI tools — select tools to bundle", items)
		p := tea.NewProgram(m)
		final, err := p.Run()
		if err != nil {
			return nil, false, err
		}
		mm := final.(MultiSelectModel)
		if mm.aborted {
			return nil, true, nil
		}
		prof.CLI = nil
		for _, item := range mm.Items {
			if item.Selected {
				prof.CLI = append(prof.CLI, item.Value)
			}
		}
	}

	// Step 4: Shell & config files
	if shell.Type != "" {
		// apply shell discovery to lock file shell section
		// (shell config is global, not per-profile — wizard caller handles it)
		prof.IncludeShell = true
	}
	{
		cfgItems := configItemsForProfile(configs)
		m := NewMultiSelect("Step 4: Config files — select configs to bundle", cfgItems)
		p := tea.NewProgram(m)
		final, err := p.Run()
		if err != nil {
			return nil, false, err
		}
		mm := final.(MultiSelectModel)
		if mm.aborted {
			return nil, true, nil
		}
		_ = mm // config file selection stored in LockFile.Configs (global); caller handles
	}

	// Step 5: SSH hosts
	{
		result, err := RunSSHHostsStep(nil) // no pre-selected for new profiles
		if err != nil {
			return nil, false, err
		}
		if result.Aborted {
			return nil, true, nil
		}
		// SSH host selection is informational here; stored as target aliases in targets.json
		// by the caller (ferry bootstrap) — we just display it
		_ = result
	}

	// Step 6: Size summary + confirm
	fmt.Println(renderSizeSummary(profileName, prof))
	confirmed, err := ConfirmPrompt("Write ferry.lock with this profile?")
	if err != nil {
		return nil, false, err
	}
	if !confirmed {
		return nil, true, nil
	}

	// Step 7: Secret scanning + encryption setup (global, run after all profiles)
	hasSensitive := false
	for _, cf := range configs {
		if cf.Sensitive {
			hasSensitive = true
			break
		}
	}
	if hasSensitive && !crypto.KeyExists() {
		fmt.Println("  sensitive files detected — generating age keypair...")
		if err := crypto.GenerateKeypair(); err != nil {
			return nil, false, fmt.Errorf("generating keypair: %w", err)
		}
		fmt.Println()
		fmt.Println(warningStyle.Render("  ⚠ important: back up your age private key"))
		fmt.Println("    ~/.ferry/key.txt")
		fmt.Println()
	}

	return &prof, false, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────

func pluginItems(plugins []discovery.PluginInfo, existing []string) []Item {
	existingSet := make(map[string]bool, len(existing))
	for _, p := range existing {
		existingSet[p] = true
	}
	var items []Item
	for _, p := range plugins {
		extra := ""
		if p.SizeBytes > 0 {
			extra = formatBytes(p.SizeBytes)
		}
		isNew := len(existing) > 0 && !existingSet[p.Name]
		items = append(items, Item{
			Label:    p.Name,
			Value:    p.Name,
			Extra:    extra,
			IsNew:    isNew,
			Selected: existingSet[p.Name] || len(existing) == 0,
		})
	}
	return items
}

func cliItemsForProfile(tools []discovery.CLITool, existing []string) []Item {
	existingSet := make(map[string]bool, len(existing))
	for _, t := range existing {
		existingSet[t] = true
	}
	var items []Item
	for _, t := range tools {
		isNew := len(existing) > 0 && !existingSet[t.Name]
		items = append(items, Item{
			Label:    t.Name,
			Value:    t.Name,
			Extra:    t.Version,
			IsNew:    isNew,
			Selected: existingSet[t.Name] || len(existing) == 0,
		})
	}
	return items
}

func configItemsForProfile(candidates []discovery.ConfigCandidate) []Item {
	var items []Item
	for _, c := range candidates {
		label := c.Path
		if c.Sensitive {
			label += " 🔒"
		}
		items = append(items, Item{
			Label:    label,
			Value:    c.Path,
			Extra:    c.Reason,
			Selected: true,
		})
	}
	return items
}

func getUserHome() string {
	h, _ := os.UserHomeDir()
	return h
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
