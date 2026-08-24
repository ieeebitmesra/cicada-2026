package views

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ieee-ctf/internal/models"
	"ieee-ctf/internal/store"
	"ieee-ctf/internal/tui/msg"
)

const scoreboardRefreshInterval = 15 * time.Second

// ScoreboardTickMsg drives periodic leaderboard refreshes.
type ScoreboardTickMsg struct{}

// ScoreboardModel renders the live leaderboard.
type ScoreboardModel struct {
	width, height int
	table         table.Model
	db            *store.DB
	team          *models.Team
}

// NewScoreboard builds the leaderboard view.
func NewScoreboard(db *store.DB, team *models.Team) ScoreboardModel {
	cols := []table.Column{
		{Title: "Rank", Width: 6},
		{Title: "Team", Width: 28},
		{Title: "Score", Width: 12},
		{Title: "Solved", Width: 8},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(12),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#00629B"))
	s.Selected = s.Selected.
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#3FA7DC"))
	t.SetStyles(s)

	return ScoreboardModel{table: t, db: db, team: team}
}

// SetSize implements sizing.
func (m *ScoreboardModel) SetSize(w, h int) {
	m.width, m.height = w, h
	height := h - 8
	if height < 3 {
		height = 3
	}
	m.table.SetHeight(height)
	if w > 60 {
		m.table.SetWidth(w - 4)
	}
}

// Refresh reloads rows from the DB view and schedules the next auto tick.
func (m *ScoreboardModel) Refresh() tea.Cmd {
	entries, err := m.db.Scoreboard()
	if err == nil {
		rows := make([]table.Row, 0, len(entries))
		for i, e := range entries {
			name := e.TeamName
			if m.team != nil && e.TeamID == m.team.ID {
				name = "► " + name
			}
			rows = append(rows, table.Row{
				fmt.Sprintf("%d", i+1),
				name,
				formatPoints(e.TotalScore),
				fmt.Sprintf("%d", e.RoundsSolved),
			})
		}
		m.table.SetRows(rows)
	}
	return scoreboardTickCmd()
}

func scoreboardTickCmd() tea.Cmd {
	return tea.Tick(scoreboardRefreshInterval, func(time.Time) tea.Msg {
		return msg.ScoreboardTick{}
	})
}

// Update handles keys + ticks.
func (m ScoreboardModel) Update(msg_ tea.Msg) (ScoreboardModel, tea.Cmd) {
	switch msg_.(type) {
	case msg.ScoreboardTick:
		return m, m.Refresh()
	}

	if key, ok := msg_.(tea.KeyMsg); ok {
		switch key.String() {
		case "r":
			return m, m.Refresh()
		case "esc":
			return m, func() tea.Msg { return msg.Navigate{Target: msg.TDashboard} }
		case "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg_)
	return m, cmd
}

// View renders the table.
func (m ScoreboardModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Render("Scoreboard")
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).
		Render("auto-refreshes every 15s • [r] refresh now  [Esc] back")
	return lipgloss.NewStyle().Padding(1, 2).Render(title + "\n\n" + m.table.View() + "\n" + hint)
}
