package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/anthropics/ferry/internal/config"
	"github.com/anthropics/ferry/internal/discovery"
)

// ProfileRow is a processed row for the profile manager table.
type ProfileRow struct {
	Name            string
	LanguageSummary string
	Targets         []TargetRow
}

// TargetRow is a single target entry within a profile row.
type TargetRow struct {
	Host   string
	User   string
	Status string // "up-to-date" | "stale" | "not deployed"
}

// buildProfileRows joins LockFile profiles, targets.json, and local hash into display rows.
func buildProfileRows(lf *config.LockFile, targets config.TargetsFile, localHash *string) []ProfileRow {
	var rows []ProfileRow
	names := make([]string, 0, len(lf.Profiles))
	for name := range lf.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		prof := lf.Profiles[name]
		row := ProfileRow{
			Name:            name,
			LanguageSummary: buildLangSummary(prof.Languages),
		}
		for host, t := range targets {
			if t.Profile != name {
				continue
			}
			status := "stale"
			if localHash != nil && t.BundleHash == *localHash {
				status = "up-to-date"
			}
			row.Targets = append(row.Targets, TargetRow{Host: host, Status: status})
		}
		sort.Slice(row.Targets, func(i, j int) bool {
			return row.Targets[i].Host < row.Targets[j].Host
		})
		rows = append(rows, row)
	}
	return rows
}

func buildLangSummary(langs []config.LanguageConfig) string {
	if len(langs) == 0 {
		return "(no languages)"
	}
	parts := make([]string, len(langs))
	for i, l := range langs {
		name := l.Name
		if l.Tier == "lsp-only" {
			name += " (LSP)"
		}
		parts[i] = name
	}
	return strings.Join(parts, " · ")
}

// ProfileManagerAction represents what the user chose.
type ProfileManagerAction int

const (
	PMActionQuit ProfileManagerAction = iota
	PMActionNew
	PMActionEdit
	PMActionDelete
	PMActionBuild
)

// ProfileManagerResult is returned after RunProfileManager.
type ProfileManagerResult struct {
	Action      ProfileManagerAction
	ProfileName string
	Aborted     bool
}

type profileManagerModel struct {
	rows     []ProfileRow
	cursor   int
	readOnly bool
	result   *ProfileManagerResult
}

func newProfileManagerModel(
	lf *config.LockFile,
	targets config.TargetsFile,
	localHash *string,
	_ []discovery.SSHHost,
	readOnly bool,
) profileManagerModel {
	rows := buildProfileRows(lf, targets, localHash)
	return profileManagerModel{rows: rows, readOnly: readOnly}
}

func (m profileManagerModel) Init() tea.Cmd { return nil }

func (m profileManagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch km.String() {
	case "ctrl+c", "q":
		m.result = &ProfileManagerResult{Action: PMActionQuit}
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
	case "e":
		if !m.readOnly && len(m.rows) > 0 {
			m.result = &ProfileManagerResult{
				Action:      PMActionEdit,
				ProfileName: m.rows[m.cursor].Name,
			}
			return m, tea.Quit
		}
	case "n":
		if !m.readOnly {
			m.result = &ProfileManagerResult{Action: PMActionNew}
			return m, tea.Quit
		}
	case "d":
		if !m.readOnly && len(m.rows) > 0 {
			m.result = &ProfileManagerResult{
				Action:      PMActionDelete,
				ProfileName: m.rows[m.cursor].Name,
			}
			return m, tea.Quit
		}
	case "b":
		if !m.readOnly && len(m.rows) > 0 {
			m.result = &ProfileManagerResult{
				Action:      PMActionBuild,
				ProfileName: m.rows[m.cursor].Name,
			}
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m profileManagerModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("⛴  ferry") + "\n\n")
	b.WriteString(fmt.Sprintf("  %-16s %-40s %s\n", "PROFILE", "LANGUAGES", "TARGETS"))
	b.WriteString("  " + strings.Repeat("─", 80) + "\n")

	for i, row := range m.rows {
		cursor := "  "
		if !m.readOnly && i == m.cursor {
			cursor = cursorStyle.Render("❯ ")
		}

		langCol := row.LanguageSummary
		if len(langCol) > 38 {
			langCol = langCol[:35] + "..."
		}

		if len(row.Targets) == 0 {
			b.WriteString(fmt.Sprintf("%s%-16s %-40s %s\n",
				cursor, row.Name, langCol,
				subtleStyle.Render("(no targets)")))
		} else {
			for j, tr := range row.Targets {
				var tgtStr string
				switch tr.Status {
				case "up-to-date":
					tgtStr = greenStyle.Render(tr.Host) + "  " + greenStyle.Render("up-to-date")
				case "stale":
					tgtStr = yellowStyle.Render(tr.Host) + "  " + yellowStyle.Render("stale")
				default:
					tgtStr = greyStyle.Render(tr.Host) + "  " + greyStyle.Render(tr.Status)
				}
				if j == 0 {
					b.WriteString(fmt.Sprintf("%s%-16s %-40s %s\n",
						cursor, row.Name, langCol, tgtStr))
				} else {
					b.WriteString(fmt.Sprintf("  %-16s %-40s %s\n",
						"", "", tgtStr))
				}
			}
		}
	}

	if len(m.rows) == 0 {
		b.WriteString(subtleStyle.Render("  (no profiles — press n to create one)\n"))
	}

	b.WriteString("\n")
	if m.readOnly {
		b.WriteString(subtleStyle.Render("  ↑↓: navigate   q: quit\n"))
	} else {
		b.WriteString(subtleStyle.Render("  e: edit   n: new   d: delete   b: build   ↑↓: navigate   q: quit\n"))
	}
	return b.String()
}

// RunProfileManager runs the interactive profile manager.
func RunProfileManager(lf *config.LockFile, readOnly bool) (*ProfileManagerResult, error) {
	targets, err := config.ReadTargets()
	if err != nil {
		return nil, err
	}
	lhash, _ := config.LockFileHash()
	var localHash *string
	if lhash != "" {
		localHash = &lhash
	}
	sshHosts, _ := discovery.ParseSSHConfig()

	m := newProfileManagerModel(lf, targets, localHash, sshHosts, readOnly)
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return nil, err
	}
	fm := final.(profileManagerModel)
	if fm.result == nil {
		return &ProfileManagerResult{Action: PMActionQuit}, nil
	}
	return fm.result, nil
}

// RenderProfileTable returns a static (non-interactive) string of the profile × targets view.
func RenderProfileTable(lf *config.LockFile) (string, error) {
	targets, err := config.ReadTargets()
	if err != nil {
		return "", err
	}
	lhash, _ := config.LockFileHash()
	var localHash *string
	if lhash != "" {
		localHash = &lhash
	}

	rows := buildProfileRows(lf, targets, localHash)

	var b strings.Builder
	b.WriteString("⛴  ferry ls\n\n")
	b.WriteString(fmt.Sprintf("  %-16s %-40s %s\n", "PROFILE", "LANGUAGES", "TARGETS"))
	b.WriteString("  " + strings.Repeat("─", 80) + "\n")

	for _, row := range rows {
		langCol := row.LanguageSummary
		if len(langCol) > 38 {
			langCol = langCol[:35] + "..."
		}
		if len(row.Targets) == 0 {
			b.WriteString(fmt.Sprintf("  %-16s %-40s %s\n",
				row.Name, langCol, "(no targets)"))
		} else {
			for j, tr := range row.Targets {
				tgtStr := fmt.Sprintf("%-20s  %s", tr.Host, tr.Status)
				if j == 0 {
					b.WriteString(fmt.Sprintf("  %-16s %-40s %s\n",
						row.Name, langCol, tgtStr))
				} else {
					b.WriteString(fmt.Sprintf("  %-16s %-40s %s\n",
						"", "", tgtStr))
				}
			}
		}
	}

	if len(rows) == 0 {
		b.WriteString("  (no profiles — run: ferry init)\n")
	}
	return b.String(), nil
}

// Local lipgloss styles for status coloring.
var (
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	greyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)
