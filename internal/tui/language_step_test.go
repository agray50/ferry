package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/registry"
)

func TestLangListModel_toggleSelection(t *testing.T) {
	langs := []registry.Language{
		{Name: "go", ApproxSizeMB: 130},
		{Name: "python", ApproxSizeMB: 150},
	}
	m := newLangListModel(langs, nil)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	mm := m2.(langListModel)
	if !mm.items[0].selected {
		t.Error("pressing space should select item 0")
	}
	m3, _ := mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	mm2 := m3.(langListModel)
	if mm2.items[0].selected {
		t.Error("pressing space again should deselect item 0")
	}
}

func TestLangListModel_drillInOnEnter(t *testing.T) {
	langs := []registry.Language{
		{Name: "python", ApproxSizeMB: 150, ApproxLSPOnlyMB: 25},
	}
	m := newLangListModel(langs, nil)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m3, _ := m2.(langListModel).Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm := m3.(langListModel)
	if mm.mode != langModeConfig {
		t.Errorf("pressing enter on selected item should switch to config mode, got mode=%d", mm.mode)
	}
}

func TestLangListModel_nodrillInOnUnselected(t *testing.T) {
	langs := []registry.Language{
		{Name: "go", ApproxSizeMB: 130},
	}
	m := newLangListModel(langs, nil)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm := m2.(langListModel)
	if mm.mode == langModeConfig {
		t.Error("enter on unselected item should not open configurator")
	}
}

func TestLangListModel_configResultPropagates(t *testing.T) {
	langs := []registry.Language{
		{Name: "python", ApproxSizeMB: 150, ApproxLSPOnlyMB: 25,
			LSP: "pyright", Formatters: []string{"black"}, Linters: []string{"flake8"}},
	}
	m := newLangListModel(langs, nil)
	m.items[0].selected = true
	m.mode = langModeConfig
	m.configurator = newLangConfiguratorModel(langs[0], config.LanguageConfig{Name: "python", Tier: "full"})
	m.configurator.field = cfgFieldTier
	m.configurator.tierIdx = 1 // lsp-only
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm := m2.(langListModel)
	if mm.mode != langModeList {
		t.Error("esc from configurator should return to list mode")
	}
	if mm.items[0].cfg.Tier != "lsp-only" {
		t.Errorf("tier should be saved as lsp-only, got %s", mm.items[0].cfg.Tier)
	}
}

func TestLangListModel_collectConfigs(t *testing.T) {
	langs := []registry.Language{
		{Name: "go", ApproxSizeMB: 130},
		{Name: "python", ApproxSizeMB: 150},
	}
	m := newLangListModel(langs, nil)
	m.items[0].selected = true
	m.items[0].cfg = config.LanguageConfig{Name: "go", Tier: "full"}
	m.items[1].selected = false
	cfgs := m.selectedConfigs()
	if len(cfgs) != 1 {
		t.Fatalf("want 1 selected config, got %d", len(cfgs))
	}
	if cfgs[0].Name != "go" {
		t.Errorf("want go, got %s", cfgs[0].Name)
	}
}

func TestLangConfiguratorModel_result(t *testing.T) {
	lang := registry.Language{
		Name:         "python",
		ApproxSizeMB: 150,
		Formatters:   []string{"black", "isort"},
		Linters:      []string{"flake8"},
		Runtime: &registry.Runtime{
			AvailableVersions: []string{"3.11", "3.12"},
		},
	}
	m := newLangConfiguratorModel(lang, config.LanguageConfig{Name: "python", Tier: "full"})
	m.tierIdx = 1 // lsp-only
	m.versionIdx = 1 // 3.12
	m.fmtSel = []bool{true, false} // only black
	m.pkgInput = "requests httpx"

	result := m.result()
	if result.Tier != "lsp-only" {
		t.Errorf("want tier lsp-only, got %s", result.Tier)
	}
	if result.RuntimeVersion != "3.12" {
		t.Errorf("want version 3.12, got %s", result.RuntimeVersion)
	}
	if len(result.Formatters) != 1 || result.Formatters[0] != "black" {
		t.Errorf("want [black], got %v", result.Formatters)
	}
	if len(result.ExtraPackages) != 2 {
		t.Errorf("want 2 extra packages, got %d", len(result.ExtraPackages))
	}
}

func TestVersionLabels(t *testing.T) {
	cases := []struct {
		versions []string
		want     []string
	}{
		// Numeric semver: first → "latest", rest → major.minor~
		{[]string{"3.12", "3.11", "3.10"}, []string{"latest", "v3.11~", "v3.10~"}},
		{[]string{"1.22.5", "1.21.11", "1.20.14"}, []string{"latest", "v1.21~", "v1.20~"}},
		{[]string{"22", "20", "18"}, []string{"latest", "v20~", "v18~"}},
		{[]string{"21", "17", "11"}, []string{"latest", "v17~", "v11~"}},
		// Minor == 0: collapse to major only
		{[]string{"8.0", "7.0", "6.0"}, []string{"latest", "v7~", "v6~"}},
		// Named channels kept as-is (no "latest" substitution)
		{[]string{"stable", "beta", "nightly"}, []string{"stable", "beta", "nightly"}},
		// Single entry
		{[]string{"0.14.0"}, []string{"latest"}},
	}
	for _, tc := range cases {
		got := versionLabels(tc.versions)
		if len(got) != len(tc.want) {
			t.Errorf("versionLabels(%v) len=%d, want %d", tc.versions, len(got), len(tc.want))
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("versionLabels(%v)[%d] = %q, want %q", tc.versions, i, got[i], tc.want[i])
			}
		}
	}
}

func TestNewLangListModel_prePopulatesExisting(t *testing.T) {
	langs := []registry.Language{
		{Name: "go", ApproxSizeMB: 130},
		{Name: "python", ApproxSizeMB: 150},
	}
	existing := []config.LanguageConfig{
		{Name: "go", Tier: "lsp-only"},
	}
	m := newLangListModel(langs, existing)

	if !m.items[0].selected {
		t.Error("go should be pre-selected from existing")
	}
	if m.items[0].cfg.Tier != "lsp-only" {
		t.Errorf("go tier should be lsp-only, got %s", m.items[0].cfg.Tier)
	}
	if m.items[1].selected {
		t.Error("python should not be selected (not in existing)")
	}
}
