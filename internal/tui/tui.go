package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	cursorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	titleStyle      = lipgloss.NewStyle().Bold(true).Underline(true)
	subtleStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	warningStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	highlightStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
)

// Item is a selectable list entry.
type Item struct {
	Label   string
	Value   string
	Extra   string // shown right-aligned
	IsNew   bool   // highlight as [new]
	Selected bool
}

// MultiSelectModel is a reusable bubbletea model for multi-select lists.
type MultiSelectModel struct {
	Title   string
	Items   []Item
	cursor  int
	filter  string
	filtering bool
	done    bool
	aborted bool
}

func NewMultiSelect(title string, items []Item) MultiSelectModel {
	return MultiSelectModel{Title: title, Items: items}
}

func (m MultiSelectModel) Init() tea.Cmd { return nil }

func (m MultiSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.filtering {
			switch msg.String() {
			case "enter", "esc":
				m.filtering = false
			case "backspace":
				if len(m.filter) > 0 {
					m.filter = m.filter[:len(m.filter)-1]
				}
			default:
				if len(msg.Runes) > 0 {
					m.filter += string(msg.Runes)
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			m.aborted = true
			return m, tea.Quit
		case "/":
			m.filtering = true
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.visibleItems())-1 {
				m.cursor++
			}
		case " ", "tab":
			visible := m.visibleIndices()
			if m.cursor < len(visible) {
				idx := visible[m.cursor]
				m.Items[idx].Selected = !m.Items[idx].Selected
			}
		case "enter":
			m.done = true
			return m, tea.Quit
		case "esc":
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m MultiSelectModel) visibleIndices() []int {
	var out []int
	for i, item := range m.Items {
		if m.filter == "" || strings.Contains(strings.ToLower(item.Label), strings.ToLower(m.filter)) {
			out = append(out, i)
		}
	}
	return out
}

func (m MultiSelectModel) visibleItems() []Item {
	indices := m.visibleIndices()
	out := make([]Item, len(indices))
	for i, idx := range indices {
		out[i] = m.Items[idx]
	}
	return out
}

func (m MultiSelectModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(m.Title) + "\n\n")

	if m.filtering {
		b.WriteString(fmt.Sprintf("  filter: %s_\n\n", m.filter))
	} else if m.filter != "" {
		b.WriteString(subtleStyle.Render(fmt.Sprintf("  filter: %s  (esc to clear)\n\n", m.filter)))
	}

	visible := m.visibleIndices()
	for i, idx := range visible {
		item := m.Items[idx]
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("❯ ")
		}

		checkbox := "◯"
		if item.Selected {
			checkbox = selectedStyle.Render("◉")
		}

		label := item.Label
		if item.IsNew {
			label += highlightStyle.Render(" [new]")
		}

		extra := ""
		if item.Extra != "" {
			extra = subtleStyle.Render("  " + item.Extra)
		}

		b.WriteString(fmt.Sprintf("%s%s %s%s\n", cursor, checkbox, label, extra))
	}

	if len(visible) == 0 {
		b.WriteString(subtleStyle.Render("  (no matches)\n"))
	}

	b.WriteString(subtleStyle.Render("\n  / filter   space toggle   enter confirm   ctrl+c abort\n"))
	return b.String()
}

// ConfirmPrompt shows a yes/no prompt. Returns true if confirmed.
func ConfirmPrompt(prompt string) (bool, error) {
	m := confirmModel{prompt: prompt}
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return false, err
	}
	cm := final.(confirmModel)
	return cm.confirmed, nil
}

type confirmModel struct {
	prompt    string
	confirmed bool
	done      bool
}

func (m confirmModel) Init() tea.Cmd { return nil }
func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch strings.ToLower(km.String()) {
		case "y", "enter":
			m.confirmed = true
			m.done = true
			return m, tea.Quit
		case "n", "ctrl+c", "esc":
			m.confirmed = false
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}
func (m confirmModel) View() string {
	return fmt.Sprintf("%s [Y/n] ", m.prompt)
}

// TypeConfirm prompts the user to type a specific word to confirm.
type TypeConfirmModel struct {
	Prompt   string
	Expected string
	input    string
	done     bool
	Confirmed bool
}

func (m TypeConfirmModel) Init() tea.Cmd { return nil }
func (m TypeConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "enter":
			m.Confirmed = m.input == m.Expected
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.done = true
			return m, tea.Quit
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if len(km.Runes) > 0 {
				m.input += string(km.Runes)
			}
		}
	}
	return m, nil
}
func (m TypeConfirmModel) View() string {
	return fmt.Sprintf("%s: %s", m.prompt(), m.input)
}
func (m TypeConfirmModel) prompt() string {
	return warningStyle.Render(m.Prompt)
}
