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
	offset       int // top of visible window
	height       int // terminal height (set via WindowSizeMsg)
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
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.height = ws.Height
		return m, nil
	}
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
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
		}
	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
			if m.cursor >= m.offset+m.visibleHeight() {
				m.offset = m.cursor - m.visibleHeight() + 1
			}
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
	selected := len(m.selectedConfigs())
	header := fmt.Sprintf("Languages — %d/%d selected", selected, len(m.items))
	b.WriteString(titleStyle.Render(header) + "\n\n")

	vh := m.visibleHeight()
	end := m.offset + vh
	if end > len(m.items) {
		end = len(m.items)
	}

	for i := m.offset; i < end; i++ {
		item := m.items[i]
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
		b.WriteString(fmt.Sprintf("%s%s %-14s%s%s\n", cursor, checkbox, item.lang.Name, tierLabel, configHint))
	}

	// scroll indicator
	if len(m.items) > vh {
		scrollInfo := fmt.Sprintf("%d-%d of %d", m.offset+1, end, len(m.items))
		b.WriteString(subtleStyle.Render(fmt.Sprintf("\n  %s\n", scrollInfo)))
	}
	b.WriteString(subtleStyle.Render("  space: toggle   enter: configure   q: done   ↑↓: navigate   ctrl+c: abort\n"))
	return b.String()
}

func (m langListModel) visibleHeight() int {
	const overhead = 6 // title + blank + footer + padding
	h := m.height - overhead
	if h < 8 {
		h = 15 // sensible default before first WindowSizeMsg
	}
	return h
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
	p := tea.NewProgram(m, tea.WithAltScreen())
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

// activeRuntime returns the runtime relevant to the current tier selection.
// When the user has chosen "lsp-only" and the LSP-only runtime has its own
// AvailableVersions (e.g. rust-analyzer, kotlin-language-server), we show
// those versions instead of the full runtime's language versions.
func (m langConfiguratorModel) activeRuntime() *registry.Runtime {
	if m.tierIdx == 1 && m.lang.LSPOnlyRuntime != nil && len(m.lang.LSPOnlyRuntime.AvailableVersions) > 0 {
		return m.lang.LSPOnlyRuntime
	}
	return m.lang.Runtime
}

func newLangConfiguratorModel(lang registry.Language, cfg config.LanguageConfig) langConfiguratorModel {
	m := langConfiguratorModel{lang: lang, initial: cfg}
	if cfg.Tier == "lsp-only" {
		m.tierIdx = 1
	}
	if rt := m.activeRuntime(); rt != nil {
		for i, v := range rt.AvailableVersions {
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
		// Linters default off — LSP covers most diagnostics interactively.
		// Only pre-select if the user has previously chosen them.
		m.lintSel[i] = lintSet[l]
	}
	m.pkgInput = strings.Join(cfg.ExtraPackages, " ")
	return m
}

func (m langConfiguratorModel) result() config.LanguageConfig {
	cfg := config.LanguageConfig{Name: m.lang.Name, Tier: "full"}
	if m.tierIdx == 1 {
		cfg.Tier = "lsp-only"
	}
	if rt := m.activeRuntime(); rt != nil && m.versionIdx < len(rt.AvailableVersions) {
		cfg.RuntimeVersion = rt.AvailableVersions[m.versionIdx]
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
			m.versionIdx = 0
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
			m.versionIdx = 0
		}
	case cfgFieldVersion:
		if rt := m.activeRuntime(); rt != nil && m.versionIdx < len(rt.AvailableVersions)-1 {
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
		{"Linters ", cfgFieldLinters, m.renderLinters() + subtleStyle.Render("  (optional)")},
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

// renderSelector renders a horizontal list of options, highlighting the current one with ❯.
func renderSelector(options []string, current int) string {
	parts := make([]string, len(options))
	for i, opt := range options {
		if i == current {
			parts[i] = selectedStyle.Render("❯ " + opt)
		} else {
			parts[i] = subtleStyle.Render(opt)
		}
	}
	return strings.Join(parts, "  ")
}

// versionLabels converts exact version strings to display labels.
// The first entry becomes "latest"; subsequent entries are abbreviated
// to major.minor with a trailing ~ (e.g. "1.22.5" → "v1.22~", "22" → "v22~").
// Named channels such as "stable", "beta", "nightly" are kept as-is.
func versionLabels(versions []string) []string {
	labels := make([]string, len(versions))
	for i, v := range versions {
		if isNamedChannel(v) {
			labels[i] = v
			continue
		}
		if i == 0 {
			labels[i] = "latest"
		} else {
			labels[i] = "v" + majorMinorOf(v) + "~"
		}
	}
	return labels
}

// isNamedChannel reports whether v is a word-only release channel
// (e.g. "stable", "beta", "nightly") rather than a numeric version.
func isNamedChannel(v string) bool {
	for _, r := range v {
		if r >= '0' && r <= '9' {
			return false
		}
	}
	return true
}

// majorMinorOf returns the major.minor prefix of a version string,
// collapsing a trailing .0 minor back to just the major.
// Examples: "1.22.5" → "1.22",  "8.0" → "8",  "22" → "22",  "0.21.3" → "0.21"
func majorMinorOf(v string) string {
	parts := strings.SplitN(v, ".", 3)
	if len(parts) == 1 {
		return parts[0]
	}
	if parts[1] == "0" {
		return parts[0]
	}
	return parts[0] + "." + parts[1]
}

// renderCheckboxes renders a horizontal list of togglable items.
func renderCheckboxes(labels []string, selected []bool) string {
	if len(labels) == 0 {
		return subtleStyle.Render("(none)")
	}
	parts := make([]string, len(labels))
	for i, label := range labels {
		if i < len(selected) && selected[i] {
			parts[i] = selectedStyle.Render("[x] " + label)
		} else {
			parts[i] = subtleStyle.Render("[ ] " + label)
		}
	}
	return strings.Join(parts, "  ")
}

func (m langConfiguratorModel) renderTier() string {
	return renderSelector([]string{"Full runtime", "LSP only"}, m.tierIdx)
}

func (m langConfiguratorModel) renderVersion() string {
	rt := m.activeRuntime()
	if rt == nil || len(rt.AvailableVersions) == 0 {
		return subtleStyle.Render("(n/a)")
	}
	return renderSelector(versionLabels(rt.AvailableVersions), m.versionIdx)
}

func (m langConfiguratorModel) renderLSP() string {
	options := append([]string{m.lang.LSP}, func() []string {
		alts := make([]string, len(m.lang.AlternateLSPs))
		for i, a := range m.lang.AlternateLSPs {
			alts[i] = a.Name
		}
		return alts
	}()...)
	return renderSelector(options, m.lspIdx)
}

func (m langConfiguratorModel) renderFormatters() string {
	return renderCheckboxes(m.lang.Formatters, m.fmtSel)
}

func (m langConfiguratorModel) renderLinters() string {
	return renderCheckboxes(m.lang.Linters, m.lintSel)
}
