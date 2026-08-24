package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ieee-ctf/internal/models"
	"ieee-ctf/internal/scoring"
	"ieee-ctf/internal/tui/msg"
)

type menuItem struct {
	title  string
	desc   string
	target msg.Target
	quit   bool
}

func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.desc }
func (i menuItem) FilterValue() string { return i.title }

// DashboardModel is the main menu.
type DashboardModel struct {
	width, height int
	list          list.Model
	svc           *scoring.Service
	team          *models.Team

	score     float64
	solved    int
	totalRds  int
}

// NewDashboard builds the dashboard.
func NewDashboard(svc *scoring.Service, team *models.Team) DashboardModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Main Menu"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#00629B")).
		Padding(0, 1)
	return DashboardModel{list: l, svc: svc, team: team}
}

// SetSize implements sizing.
func (m *DashboardModel) SetSize(w, h int) {
	m.width, m.height = w, h
	m.list.SetSize(w-4, h-6)
}

// Refresh recomputes score + rebuilds the menu.
func (m *DashboardModel) Refresh() tea.Cmd {
	b, err := m.svc.Breakdown(m.team.ID)
	if err == nil && b != nil {
		m.score = b.Total
		m.solved = len(b.SolvedRounds)
	}
	if rounds, err := m.svc.DB.ListRounds(false); err == nil {
		m.totalRds = len(rounds)
	}

	items := []list.Item{
		menuItem{"Submit Flag", "Attempt a flag for a round", msg.TSubmitFlag, false},
		menuItem{"Request Hint", "PGP-signed hint request", msg.TRequestHint, false},
		menuItem{"Skip Round", "Skip for −50% of your current total", msg.TSkipRound, false},
		menuItem{"Scoreboard", "Live leaderboard", msg.TScoreboard, false},
		menuItem{"My Status", "Progress + points breakdown", msg.TTeamStatus, false},
		menuItem{"Quit", "Disconnect", msg.Target(-1), true},
	}
	cmd := m.list.SetItems(items)
	return cmd
}

// Update handles keys.
func (m DashboardModel) Update(msg_ tea.Msg) (DashboardModel, tea.Cmd) {
	if key, ok := msg_.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter":
			if item, ok := m.list.SelectedItem().(menuItem); ok {
				if item.quit {
					return m, tea.Quit
				}
				t := item.target
				return m, func() tea.Msg { return msg.Navigate{Target: t} }
			}
		case "r":
			return m, m.Refresh()
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg_)
	return m, cmd
}

// View renders summary + menu.
func (m DashboardModel) View() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Team %s — solved %d/%d rounds — current total: ",
		m.team.Name, m.solved, m.totalRds))
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00C853"))
	if m.score < 0 {
		style = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5252"))
	}
	b.WriteString(style.Render(fmt.Sprintf("%.1f pts", m.score)))
	b.WriteString("\n")
	return lipgloss.NewStyle().Width(m.width).Padding(1, 2).Render(b.String() + m.list.View())
}
