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
