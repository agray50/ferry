package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/registry"
)

type langMode int

const (
	langModeList   langMode = iota
	langModeConfig
)

type langItem struct {
	lang     registry.Language
	selected bool
	cfg      config.LanguageConfig
}

type langListModel struct {
	items        []langItem
	cursor       int
	mode         langMode
	configurator langConfiguratorModel
	done         bool
	aborted      bool
}

func newLangListModel(langs []registry.Language, existing []config.LanguageConfig) langListModel {
	existingMap := make(map[string]config.LanguageConfig, len(existing))
	for _, lc := range existing {
		existingMap[lc.Name] = lc
	}
	items := make([]langItem, len(langs))
	for i, l := range langs {
		cfg, ok := existingMap[l.Name]
		if !ok {
			cfg = config.LanguageConfig{Name: l.Name, Tier: "full"}
		}
		items[i] = langItem{lang: l, selected: ok, cfg: cfg}
	}
	return langListModel{items: items}
}

func (m langListModel) Init() tea.Cmd { return nil }

func (m langListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.mode == langModeConfig {
		return m.updateConfigurator(msg)
	}
	return m.updateList(msg)
}

func (m langListModel) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch km.String() {
	case "ctrl+c":
		m.aborted = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case " ", "tab":
		m.items[m.cursor].selected = !m.items[m.cursor].selected
		if m.items[m.cursor].selected && m.items[m.cursor].cfg.Name == "" {
			m.items[m.cursor].cfg = config.LanguageConfig{
				Name: m.items[m.cursor].lang.Name,
				Tier: "full",
			}
		}
	case "enter":
		if m.items[m.cursor].selected {
			m.configurator = newLangConfiguratorModel(m.items[m.cursor].lang, m.items[m.cursor].cfg)
			m.mode = langModeConfig
		}
		// do nothing on unselected - use q to finish
	case "q":
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m langListModel) updateConfigurator(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if km.String() == "esc" || km.String() == "enter" {
		m.items[m.cursor].cfg = m.configurator.result()
		m.mode = langModeList
		return m, nil
	}
	next, cmd := m.configurator.Update(msg)
	m.configurator = next.(langConfiguratorModel)
	return m, cmd
}

func (m langListModel) View() string {
	if m.mode == langModeConfig {
		return m.configurator.View()
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("Languages — select to include, enter to configure") + "\n\n")
	for i, item := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("❯ ")
		}
		checkbox := "◯"
		if item.selected {
			checkbox = selectedStyle.Render("◉")
		}
		tierLabel := ""
		if item.selected {
			if item.cfg.Tier == "lsp-only" {
				tierLabel = subtleStyle.Render("  LSP ~" + fmt.Sprintf("%dMB", item.lang.ApproxLSPOnlyMB))
			} else {
				tierLabel = subtleStyle.Render(fmt.Sprintf("  ~%dMB", item.lang.ApproxSizeMB))
			}
		}
		configHint := ""
		if item.selected {
			configHint = highlightStyle.Render("  [configure ▸]")
		}
		b.WriteString(fmt.Sprintf("%s%s %s%s%s\n", cursor, checkbox, item.lang.Name, tierLabel, configHint))
	}
	b.WriteString(subtleStyle.Render("\n  space: toggle   enter: configure   q: done   ↑↓: navigate   ctrl+c: abort\n"))
	return b.String()
}

func (m langListModel) selectedConfigs() []config.LanguageConfig {
	var out []config.LanguageConfig
	for _, item := range m.items {
		if item.selected {
			out = append(out, item.cfg)
		}
	}
	return out
}

func RunLangStep(allLangs []registry.Language, existing []config.LanguageConfig) ([]config.LanguageConfig, bool, error) {
	m := newLangListModel(allLangs, existing)
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return nil, false, err
	}
	fm := final.(langListModel)
	if fm.aborted {
		return nil, true, nil
	}
	return fm.selectedConfigs(), false, nil
}

// ── configurator ────────────────────────────────────────────────────────────

type cfgField int

const (
	cfgFieldTier       cfgField = iota
	cfgFieldVersion
	cfgFieldLSP
	cfgFieldFormatters
	cfgFieldLinters
	cfgFieldPackages
)

type langConfiguratorModel struct {
	lang       registry.Language
	initial    config.LanguageConfig
	field      cfgField
	tierIdx    int
	versionIdx int
	lspIdx     int
	fmtSel     []bool
	lintSel    []bool
	pkgInput   string
}

func newLangConfiguratorModel(lang registry.Language, cfg config.LanguageConfig) langConfiguratorModel {
	m := langConfiguratorModel{lang: lang, initial: cfg}
	if cfg.Tier == "lsp-only" {
		m.tierIdx = 1
	}
	if lang.Runtime != nil {
		for i, v := range lang.Runtime.AvailableVersions {
			if cfg.RuntimeVersion != "" && v == cfg.RuntimeVersion {
				m.versionIdx = i
				break
			}
		}
	}
	for i, alt := range lang.AlternateLSPs {
		if cfg.LSP == alt.Name {
			m.lspIdx = i + 1
			break
		}
	}
	fmtSet := make(map[string]bool, len(cfg.Formatters))
	for _, f := range cfg.Formatters {
		fmtSet[f] = true
	}
	m.fmtSel = make([]bool, len(lang.Formatters))
	for i, f := range lang.Formatters {
		m.fmtSel[i] = fmtSet[f] || len(cfg.Formatters) == 0
	}
	lintSet := make(map[string]bool, len(cfg.Linters))
	for _, l := range cfg.Linters {
		lintSet[l] = true
	}
	m.lintSel = make([]bool, len(lang.Linters))
	for i, l := range lang.Linters {
		m.lintSel[i] = lintSet[l] || len(cfg.Linters) == 0
	}
	m.pkgInput = strings.Join(cfg.ExtraPackages, " ")
	return m
}

func (m langConfiguratorModel) result() config.LanguageConfig {
	cfg := config.LanguageConfig{Name: m.lang.Name, Tier: "full"}
	if m.tierIdx == 1 {
		cfg.Tier = "lsp-only"
	}
	if m.lang.Runtime != nil && m.versionIdx < len(m.lang.Runtime.AvailableVersions) {
		cfg.RuntimeVersion = m.lang.Runtime.AvailableVersions[m.versionIdx]
	}
	if m.lspIdx > 0 && m.lspIdx-1 < len(m.lang.AlternateLSPs) {
		cfg.LSP = m.lang.AlternateLSPs[m.lspIdx-1].Name
	}
	for i, f := range m.lang.Formatters {
		if i < len(m.fmtSel) && m.fmtSel[i] {
			cfg.Formatters = append(cfg.Formatters, f)
		}
	}
	for i, l := range m.lang.Linters {
		if i < len(m.lintSel) && m.lintSel[i] {
			cfg.Linters = append(cfg.Linters, l)
		}
	}
	for _, pkg := range strings.Fields(m.pkgInput) {
		cfg.ExtraPackages = append(cfg.ExtraPackages, pkg)
	}
	return cfg
}

func (m langConfiguratorModel) Init() tea.Cmd { return nil }

func (m langConfiguratorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch km.String() {
	case "up", "k":
		if m.field > 0 {
			m.field--
		}
	case "down", "j", "tab":
		if int(m.field) < int(cfgFieldPackages) {
			m.field++
		}
	case "left", "h":
		m = m.adjustFieldLeft()
	case "right", "l":
		m = m.adjustFieldRight()
	case " ":
		m = m.toggleField()
	case "backspace":
		if m.field == cfgFieldPackages && len(m.pkgInput) > 0 {
			m.pkgInput = m.pkgInput[:len(m.pkgInput)-1]
		}
	default:
		if m.field == cfgFieldPackages && len(km.Runes) > 0 {
			m.pkgInput += string(km.Runes)
		}
	}
	return m, nil
}

func (m langConfiguratorModel) adjustFieldLeft() langConfiguratorModel {
	switch m.field {
	case cfgFieldTier:
		if m.tierIdx > 0 {
			m.tierIdx--
		}
	case cfgFieldVersion:
		if m.versionIdx > 0 {
			m.versionIdx--
		}
	case cfgFieldLSP:
		if m.lspIdx > 0 {
			m.lspIdx--
		}
	}
	return m
}

func (m langConfiguratorModel) adjustFieldRight() langConfiguratorModel {
	switch m.field {
	case cfgFieldTier:
		if m.tierIdx < 1 {
			m.tierIdx++
		}
	case cfgFieldVersion:
		if m.lang.Runtime != nil && m.versionIdx < len(m.lang.Runtime.AvailableVersions)-1 {
			m.versionIdx++
		}
	case cfgFieldLSP:
		if m.lspIdx < len(m.lang.AlternateLSPs) {
			m.lspIdx++
		}
	}
	return m
}

func (m langConfiguratorModel) toggleField() langConfiguratorModel {
	switch m.field {
	case cfgFieldFormatters:
		allOn := true
		for _, sel := range m.fmtSel {
			if !sel {
				allOn = false
				break
			}
		}
		for i := range m.fmtSel {
			m.fmtSel[i] = !allOn
		}
	case cfgFieldLinters:
		allOn := true
		for _, sel := range m.lintSel {
			if !sel {
				allOn = false
				break
			}
		}
		for i := range m.lintSel {
			m.lintSel[i] = !allOn
		}
	}
	return m
}

func (m langConfiguratorModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Configure: %s", m.lang.Name)) + "\n\n")
	pkgDisplay := m.pkgInput
	if m.field == cfgFieldPackages {
		pkgDisplay += "_"
	}
	fields := []struct {
		label   string
		field   cfgField
		content string
	}{
		{"Tier    ", cfgFieldTier, m.renderTier()},
		{"Version ", cfgFieldVersion, m.renderVersion()},
		{"LSP     ", cfgFieldLSP, m.renderLSP()},
		{"Formats ", cfgFieldFormatters, m.renderFormatters()},
		{"Linters ", cfgFieldLinters, m.renderLinters()},
		{"Packages", cfgFieldPackages, pkgDisplay},
	}
	for _, f := range fields {
		prefix := "  "
		if f.field == m.field {
			prefix = cursorStyle.Render("❯ ")
		}
		b.WriteString(fmt.Sprintf("%s%s  %s\n", prefix, subtleStyle.Render(f.label), f.content))
	}
	b.WriteString(subtleStyle.Render("\n  ↑↓: field   ←→: change   space: toggle   esc: save\n"))
	return b.String()
}

func (m langConfiguratorModel) renderTier() string {
	tiers := []string{"Full runtime", "LSP only"}
	var parts []string
	for i, t := range tiers {
		if i == m.tierIdx {
			parts = append(parts, selectedStyle.Render("❯ "+t))
		} else {
			parts = append(parts, subtleStyle.Render(t))
		}
	}
	return strings.Join(parts, "   ")
}

func (m langConfiguratorModel) renderVersion() string {
	if m.lang.Runtime == nil || len(m.lang.Runtime.AvailableVersions) == 0 {
		return subtleStyle.Render("(n/a)")
	}
	var parts []string
	for i, v := range m.lang.Runtime.AvailableVersions {
		if i == m.versionIdx {
			parts = append(parts, selectedStyle.Render("❯ "+v))
		} else {
			parts = append(parts, subtleStyle.Render(v))
		}
	}
	return strings.Join(parts, "  ")
}

func (m langConfiguratorModel) renderLSP() string {
	options := []string{m.lang.LSP}
	for _, alt := range m.lang.AlternateLSPs {
		options = append(options, alt.Name)
	}
	var parts []string
	for i, opt := range options {
		if i == m.lspIdx {
			parts = append(parts, selectedStyle.Render("❯ "+opt))
		} else {
			parts = append(parts, subtleStyle.Render(opt))
		}
	}
	return strings.Join(parts, "  ")
}

func (m langConfiguratorModel) renderFormatters() string {
	if len(m.lang.Formatters) == 0 {
		return subtleStyle.Render("(none)")
	}
	var parts []string
	for i, f := range m.lang.Formatters {
		sel := i < len(m.fmtSel) && m.fmtSel[i]
		if sel {
			parts = append(parts, selectedStyle.Render("[x] "+f))
		} else {
			parts = append(parts, subtleStyle.Render("[ ] "+f))
		}
	}
	return strings.Join(parts, "  ")
}

func (m langConfiguratorModel) renderLinters() string {
	if len(m.lang.Linters) == 0 {
		return subtleStyle.Render("(none)")
	}
	var parts []string
	for i, l := range m.lang.Linters {
		sel := i < len(m.lintSel) && m.lintSel[i]
		if sel {
			parts = append(parts, selectedStyle.Render("[x] "+l))
		} else {
			parts = append(parts, subtleStyle.Render("[ ] "+l))
		}
	}
	return strings.Join(parts, "  ")
}
