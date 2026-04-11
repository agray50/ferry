package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anthropics/ferry/internal/discovery"
)

type sshMode int

const (
	sshModeSelect  sshMode = iota
	sshModeAddForm         // add-new-host form
)

type formField int

const (
	formFieldAlias    formField = iota
	formFieldHostname
	formFieldUser
	formFieldPort
	formFieldIdentity
)

type addHostForm struct {
	field    formField
	alias    string
	hostname string
	user     string
	port     string
	identity string
}

type sshHostsModel struct {
	hosts    []discovery.SSHHost
	selected []bool
	cursor   int
	mode     sshMode
	form     addHostForm
	newHosts []discovery.SSHHost // hosts appended during this session
	done     bool
	aborted  bool
}

func newSSHHostsModel(hosts []discovery.SSHHost, preSelected []string) sshHostsModel {
	sel := make([]bool, len(hosts))
	preSet := make(map[string]bool, len(preSelected))
	for _, a := range preSelected {
		preSet[a] = true
	}
	for i, h := range hosts {
		sel[i] = preSet[h.Alias]
	}
	return sshHostsModel{hosts: hosts, selected: sel}
}

func (m sshHostsModel) Init() tea.Cmd { return nil }

func (m sshHostsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.mode == sshModeAddForm {
		return m.updateForm(msg)
	}
	return m.updateSelect(msg)
}

func (m sshHostsModel) updateSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		if m.cursor < len(m.hosts)-1 {
			m.cursor++
		}
	case " ", "tab":
		if m.cursor < len(m.selected) {
			m.selected[m.cursor] = !m.selected[m.cursor]
		}
	case "a":
		m.mode = sshModeAddForm
		m.form = addHostForm{}
	case "enter", "q":
		m.done = true
		return m, tea.Quit
	case "esc":
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m sshHostsModel) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch km.String() {
	case "ctrl+c":
		m.aborted = true
		return m, tea.Quit
	case "esc":
		m.mode = sshModeSelect
		return m, nil
	case "tab":
		if int(m.form.field) < int(formFieldIdentity) {
			m.form.field++
		}
	case "shift+tab":
		if m.form.field > 0 {
			m.form.field--
		}
	case "enter":
		if m.form.field < formFieldIdentity {
			m.form.field++
		} else {
			// save and append
			h := m.formToHost()
			if h.Alias != "" && h.Hostname != "" {
				m.newHosts = append(m.newHosts, h)
				m.hosts = append(m.hosts, h)
				m.selected = append(m.selected, true)
				// write to ~/.ssh/config (best effort — errors silently ignored in TUI)
				_ = discovery.AppendSSHHost(h)
			}
			m.mode = sshModeSelect
			m.cursor = len(m.hosts) - 1
		}
	case "backspace":
		m = m.deleteChar()
	default:
		if len(km.Runes) > 0 {
			m = m.appendChar(string(km.Runes))
		}
	}
	return m, nil
}

func (m sshHostsModel) appendChar(s string) sshHostsModel {
	switch m.form.field {
	case formFieldAlias:
		m.form.alias += s
	case formFieldHostname:
		m.form.hostname += s
	case formFieldUser:
		m.form.user += s
	case formFieldPort:
		m.form.port += s
	case formFieldIdentity:
		m.form.identity += s
	}
	return m
}

func (m sshHostsModel) deleteChar() sshHostsModel {
	switch m.form.field {
	case formFieldAlias:
		if len(m.form.alias) > 0 {
			m.form.alias = m.form.alias[:len(m.form.alias)-1]
		}
	case formFieldHostname:
		if len(m.form.hostname) > 0 {
			m.form.hostname = m.form.hostname[:len(m.form.hostname)-1]
		}
	case formFieldUser:
		if len(m.form.user) > 0 {
			m.form.user = m.form.user[:len(m.form.user)-1]
		}
	case formFieldPort:
		if len(m.form.port) > 0 {
			m.form.port = m.form.port[:len(m.form.port)-1]
		}
	case formFieldIdentity:
		if len(m.form.identity) > 0 {
			m.form.identity = m.form.identity[:len(m.form.identity)-1]
		}
	}
	return m
}

func (m sshHostsModel) formToHost() discovery.SSHHost {
	port := 22
	if m.form.port != "" {
		fmt.Sscanf(m.form.port, "%d", &port)
	}
	ident := m.form.identity
	if ident == "" {
		ident = "~/.ssh/id_ed25519"
	}
	return discovery.SSHHost{
		Alias:        m.form.alias,
		Hostname:     m.form.hostname,
		User:         m.form.user,
		Port:         port,
		IdentityFile: ident,
		FerryManaged: true,
	}
}

func (m sshHostsModel) View() string {
	if m.mode == sshModeAddForm {
		return m.viewForm()
	}
	return m.viewSelect()
}

func (m sshHostsModel) viewSelect() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("SSH hosts — select targets for this profile") + "\n\n")

	if len(m.hosts) == 0 {
		b.WriteString(subtleStyle.Render("  (no hosts found in ~/.ssh/config)\n"))
	}

	for i, h := range m.hosts {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("❯ ")
		}
		checkbox := "◯"
		if i < len(m.selected) && m.selected[i] {
			checkbox = selectedStyle.Render("◉")
		}
		info := fmt.Sprintf("%s@%s", h.User, h.Hostname)
		if h.Port != 0 && h.Port != 22 {
			info += fmt.Sprintf(":%d", h.Port)
		}
		b.WriteString(fmt.Sprintf("%s%s %-20s %s\n",
			cursor, checkbox, h.Alias, subtleStyle.Render(info)))
	}

	b.WriteString(subtleStyle.Render("\n  space: toggle   a: add new   enter: done   ctrl+c: abort\n"))
	return b.String()
}

func (m sshHostsModel) viewForm() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Add SSH host") + "\n\n")

	fields := []struct {
		label string
		field formField
		value string
	}{
		{"Alias      ", formFieldAlias, m.form.alias},
		{"Hostname   ", formFieldHostname, m.form.hostname},
		{"User       ", formFieldUser, m.form.user},
		{"Port       ", formFieldPort, m.form.port},
		{"Identity   ", formFieldIdentity, m.form.identity},
	}

	for _, f := range fields {
		prefix := "  "
		val := f.value
		if f.field == m.form.field {
			prefix = cursorStyle.Render("❯ ")
			val += "_"
		}
		placeholder := ""
		switch f.field {
		case formFieldPort:
			if f.value == "" && f.field != m.form.field {
				placeholder = subtleStyle.Render("22")
			}
		case formFieldIdentity:
			if f.value == "" && f.field != m.form.field {
				placeholder = subtleStyle.Render("~/.ssh/id_ed25519")
			}
		}
		display := val
		if display == "" {
			display = placeholder
		}
		b.WriteString(fmt.Sprintf("%s%s  %s\n", prefix, subtleStyle.Render(f.label), display))
	}

	b.WriteString(subtleStyle.Render("\n  tab/enter: next field   esc: cancel   enter on last field: save\n"))
	return b.String()
}

func (m sshHostsModel) selectedAliases() []string {
	var out []string
	for i, h := range m.hosts {
		if i < len(m.selected) && m.selected[i] {
			out = append(out, h.Alias)
		}
	}
	return out
}

// SSHHostsStepResult is returned by RunSSHHostsStep.
type SSHHostsStepResult struct {
	SelectedAliases []string
	NewHosts        []discovery.SSHHost
	Aborted         bool
}

// RunSSHHostsStep runs the SSH hosts selection step.
// preSelected is a list of aliases already associated with the profile.
func RunSSHHostsStep(preSelected []string) (*SSHHostsStepResult, error) {
	hosts, err := discovery.ParseSSHConfig()
	if err != nil {
		hosts = nil // best effort
	}
	m := newSSHHostsModel(hosts, preSelected)
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return nil, err
	}
	fm := final.(sshHostsModel)
	if fm.aborted {
		return &SSHHostsStepResult{Aborted: true}, nil
	}
	return &SSHHostsStepResult{
		SelectedAliases: fm.selectedAliases(),
		NewHosts:        fm.newHosts,
	}, nil
}
