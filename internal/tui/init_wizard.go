package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/discovery"
	"github.com/anthropics/ferry/internal/format"
	"github.com/anthropics/ferry/internal/registry"
)

func boolPtr(b bool) *bool { return &b }

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
	if seconds == 0 {
		return "< 1s"
	}
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

// RunProfileWizard runs the profile editor for a single profile.
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

	// Step 2: Neovim
	{
		nvimConfirmed, err := ConfirmPrompt("Step 2: Include Neovim?")
		if err != nil {
			return nil, false, err
		}
		prof.IncludeNvim = boolPtr(nvimConfirmed)

		if nvimConfirmed && nvim.Found {
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

	// Step 4: Shell
	{
		if shell.Type != "" {
			var items []Item
			if shell.PluginManager != "" && shell.PluginManagerPath != "" {
				items = append(items, Item{
					Label:    fmt.Sprintf("Bundle shell framework (%s)", shell.PluginManagerPath),
					Value:    "framework",
					Selected: true,
				})
			}
			if shell.ConfigPath != "" {
				items = append(items, Item{
					Label:    fmt.Sprintf("Bundle %s", shell.ConfigPath),
					Value:    "rc",
					Selected: true,
				})
			}
			if shell.ThemeConfigPath != "" {
				items = append(items, Item{
					Label:    fmt.Sprintf("Bundle theme config (%s)", shell.ThemeConfigPath),
					Value:    "theme",
					Selected: true,
				})
			}

			if len(items) > 0 {
				fmt.Printf("\n  detected: %s", shell.Type)
				if shell.PluginManager != "" {
					fmt.Printf("  │  %s", shell.PluginManager)
				}
				if shell.ThemeDetected != "" {
					fmt.Printf("  │  %s", shell.ThemeDetected)
				}
				fmt.Println()

				m := NewMultiSelect("Step 4: Shell — select components to bundle", items)
				p := tea.NewProgram(m)
				final, err := p.Run()
				if err != nil {
					return nil, false, err
				}
				mm := final.(MultiSelectModel)
				if mm.aborted {
					return nil, true, nil
				}

				selected := make(map[int]bool)
				for i, item := range mm.Items {
					if item.Selected {
						selected[i] = true
					}
				}
				prof.Shell = shellDiscoveryToProfile(shell, selected)
			}
		}
	}

	// Step 5: Size summary + confirm
	fmt.Println(renderSizeSummary(profileName, prof))
	confirmed, err := ConfirmPrompt("Write ferry.lock with this profile?")
	if err != nil {
		return nil, false, err
	}
	if !confirmed {
		return nil, true, nil
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
			extra = format.Bytes(p.SizeBytes)
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

// shellDiscoveryToProfile converts discovery results to a ShellProfile,
// filtered by which items the user selected (by index: 0=framework, 1=rc, 2=theme).
func shellDiscoveryToProfile(d discovery.ShellDiscovery, selected map[int]bool) *config.ShellProfile {
	if d.Type == "" {
		return nil
	}
	sp := &config.ShellProfile{
		Type: d.Type,
	}
	idx := 0
	if d.PluginManager != "" && d.PluginManagerPath != "" {
		if selected[idx] {
			sp.Framework = d.PluginManager
			sp.FrameworkPath = d.PluginManagerPath
		}
		idx++
	}
	if d.ConfigPath != "" {
		if selected[idx] {
			sp.RCPath = d.ConfigPath
		}
		idx++
	}
	if d.ThemeConfigPath != "" {
		if selected[idx] {
			sp.Theme = d.ThemeDetected
			sp.ThemeConfigPath = d.ThemeConfigPath
		}
	}
	// If nothing was selected, return nil (user opted out of shell bundling entirely).
	if sp.Framework == "" && sp.RCPath == "" && sp.ThemeConfigPath == "" {
		return nil
	}
	return sp
}

