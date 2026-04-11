package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anthropics/ferry/internal/discovery"
)

func TestSSHHostsModel_toggleSelect(t *testing.T) {
	hosts := []discovery.SSHHost{
		{Alias: "prod-vm", Hostname: "10.0.1.5", User: "ubuntu"},
		{Alias: "dev-vm", Hostname: "10.0.1.6", User: "ubuntu"},
	}
	m := newSSHHostsModel(hosts, nil)

	// cursor at 0; press space to select
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	mm := m2.(sshHostsModel)
	if !mm.selected[0] {
		t.Error("pressing space should select item 0")
	}
}

func TestSSHHostsModel_addModeOnA(t *testing.T) {
	m := newSSHHostsModel(nil, nil)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	mm := m2.(sshHostsModel)
	if mm.mode != sshModeAddForm {
		t.Errorf("pressing 'a' should open add form, got mode=%d", mm.mode)
	}
}

func TestSSHHostsModel_formFieldAdvance(t *testing.T) {
	m := newSSHHostsModel(nil, nil)
	m.mode = sshModeAddForm
	m.form.field = formFieldAlias

	// press tab to advance field
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	mm := m2.(sshHostsModel)
	if mm.form.field != formFieldHostname {
		t.Errorf("tab should advance to hostname field, got %d", mm.form.field)
	}
}

func TestSSHHostsModel_formFieldInputAlias(t *testing.T) {
	m := newSSHHostsModel(nil, nil)
	m.mode = sshModeAddForm
	m.form.field = formFieldAlias

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m', 'y', '-', 'v', 'm'}})
	mm := m2.(sshHostsModel)
	if mm.form.alias != "my-vm" {
		t.Errorf("want alias=my-vm, got %s", mm.form.alias)
	}
}

func TestSSHHostsModel_escCancelsAddForm(t *testing.T) {
	m := newSSHHostsModel(nil, nil)
	m.mode = sshModeAddForm
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm := m2.(sshHostsModel)
	if mm.mode != sshModeSelect {
		t.Errorf("esc should return to select mode, got mode=%d", mm.mode)
	}
}

func TestSSHHostsModel_selectedAliases(t *testing.T) {
	hosts := []discovery.SSHHost{
		{Alias: "prod-vm"},
		{Alias: "dev-vm"},
		{Alias: "staging"},
	}
	m := newSSHHostsModel(hosts, []string{"prod-vm", "staging"})
	aliases := m.selectedAliases()
	if len(aliases) != 2 {
		t.Fatalf("want 2 selected, got %d", len(aliases))
	}
}

func TestSSHHostsModel_selectedAliasesContent(t *testing.T) {
	hosts := []discovery.SSHHost{
		{Alias: "prod-vm"},
		{Alias: "dev-vm"},
		{Alias: "staging"},
	}
	m := newSSHHostsModel(hosts, []string{"prod-vm", "staging"})
	aliases := m.selectedAliases()
	if len(aliases) != 2 {
		t.Fatalf("want 2 selected, got %d", len(aliases))
	}
	if aliases[0] != "prod-vm" {
		t.Errorf("want first alias prod-vm, got %s", aliases[0])
	}
	if aliases[1] != "staging" {
		t.Errorf("want second alias staging, got %s", aliases[1])
	}
}

func TestSSHHostsModel_formToHostDefaults(t *testing.T) {
	m := newSSHHostsModel(nil, nil)
	m.form.alias = "my-vm"
	m.form.hostname = "10.0.0.1"
	// port and identity left empty

	h := m.formToHost()
	if h.Port != 22 {
		t.Errorf("want default port 22, got %d", h.Port)
	}
	if h.IdentityFile != "~/.ssh/id_ed25519" {
		t.Errorf("want default identity, got %s", h.IdentityFile)
	}
	if !h.FerryManaged {
		t.Error("want FerryManaged=true")
	}
}

func TestSSHHostsModel_formToHostInvalidPort(t *testing.T) {
	m := newSSHHostsModel(nil, nil)
	m.form.alias = "my-vm"
	m.form.hostname = "10.0.0.1"
	m.form.port = "notanumber"

	h := m.formToHost()
	if h.Port != 22 {
		t.Errorf("invalid port should default to 22, got %d", h.Port)
	}
}
