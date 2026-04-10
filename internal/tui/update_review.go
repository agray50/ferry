package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/anthropics/ferry/internal/store"
)

var largeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))

// UpdateReviewModel is a bubbletea model for the ferry update component selector.
type UpdateReviewModel struct {
	target   string
	diff     store.DiffResult
	items    []updateItem
	cursor   int
	filter   string
	filtering bool
	Confirmed bool
	Aborted  bool
}

type updateItem struct {
	component store.Component
	kind      string // "new" | "changed" | "removed"
	selected  bool
}

// NewUpdateReview creates an update review model.
func NewUpdateReview(target string, diff store.DiffResult) UpdateReviewModel {
	m := UpdateReviewModel{target: target, diff: diff}

	for _, c := range diff.New {
		m.items = append(m.items, updateItem{component: c, kind: "new", selected: true})
	}
	for _, c := range diff.Changed {
		m.items = append(m.items, updateItem{component: c, kind: "changed", selected: true})
	}
	return m
}

func (m UpdateReviewModel) Init() tea.Cmd { return nil }

func (m UpdateReviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "ctrl+c", "esc":
			m.Aborted = true
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
			vis := m.visibleIndices()
			if m.cursor < len(vis) {
				m.items[vis[m.cursor]].selected = !m.items[vis[m.cursor]].selected
			}
		case "enter":
			m.Confirmed = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m UpdateReviewModel) visibleIndices() []int {
	var out []int
	for i, item := range m.items {
		if m.filter == "" || strings.Contains(strings.ToLower(item.component.ID), strings.ToLower(m.filter)) {
			out = append(out, i)
		}
	}
	return out
}

func (m UpdateReviewModel) visibleItems() []updateItem {
	indices := m.visibleIndices()
	out := make([]updateItem, len(indices))
	for i, idx := range indices {
		out[i] = m.items[idx]
	}
	return out
}

func (m UpdateReviewModel) View() string {
	var b strings.Builder

	total := len(m.diff.New) + len(m.diff.Changed)
	b.WriteString(titleStyle.Render(fmt.Sprintf("⛴  ferry update  %s", m.target)) + "\n\n")
	b.WriteString(fmt.Sprintf("  %d changes detected — select what to ship:\n\n", total))

	if m.filtering {
		b.WriteString(fmt.Sprintf("  filter: %s_\n\n", m.filter))
	}

	vis := m.visibleIndices()
	for i, idx := range vis {
		item := m.items[idx]
		c := item.component

		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("❯ ")
		}

		checkbox := "◯"
		if item.selected {
			checkbox = selectedStyle.Render("◉")
		}

		sizeStr := formatSizeItem(c.SizeCompressed)
		if c.SizeCompressed > 50*1024*1024 {
			sizeStr = largeStyle.Render(sizeStr + " (!)")
		}

		version := c.Version
		if version == "" {
			version = item.kind
		}

		b.WriteString(fmt.Sprintf("%s%s %-30s %-20s %s\n",
			cursor, checkbox, c.ID, version, sizeStr))
	}

	if len(m.diff.Removed) > 0 {
		b.WriteString(subtleStyle.Render(fmt.Sprintf("\n  also: %d component(s) will be removed (no longer in profile)\n", len(m.diff.Removed))))
		for _, c := range m.diff.Removed {
			b.WriteString(subtleStyle.Render(fmt.Sprintf("    %s\n", c.ID)))
		}
	}

	// totals
	var selectedSize, skippedSize int64
	for _, item := range m.items {
		if item.selected {
			selectedSize += item.component.SizeCompressed
		} else {
			skippedSize += item.component.SizeCompressed
		}
	}
	b.WriteString(fmt.Sprintf("\n  selected: %s   skipped: %s\n",
		formatSizeItem(selectedSize), formatSizeItem(skippedSize)))

	b.WriteString(subtleStyle.Render("\n  / filter   space toggle   enter confirm   esc cancel\n"))
	return b.String()
}

// SelectedComponents returns components the user chose to ship.
func (m UpdateReviewModel) SelectedComponents() []store.Component {
	var out []store.Component
	for _, item := range m.items {
		if item.selected {
			out = append(out, item.component)
		}
	}
	return out
}

func formatSizeItem(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%dB", b)
	}
	if b < 1024*1024 {
		return fmt.Sprintf("%.0fKB", float64(b)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(b)/1024/1024)
}
