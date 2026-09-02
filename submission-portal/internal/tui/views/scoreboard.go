package views

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ieee-ctf/internal/middleware"
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

	top3        []models.ScoreboardEntry
	myRank      int
	totalTeams  int
	lastUpdated time.Time
}

// NewScoreboard builds the leaderboard view.
func NewScoreboard(db *store.DB, team *models.Team) ScoreboardModel {
	cols := []table.Column{
		{Title: "RANK", Width: 8},
		{Title: "TEAM IDENTITY", Width: 32},
		{Title: "TOTAL BOUNTY", Width: 18},
		{Title: "SOLVED", Width: 12},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		Bold(true).
		Foreground(lipgloss.Color("#F8FAFC")).
		Background(lipgloss.Color("#00629B")).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("#00B4D8"))

	s.Selected = s.Selected.
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#00B4D8"))

	t.SetStyles(s)

	return ScoreboardModel{table: t, db: db, team: team, lastUpdated: time.Now()}
}

// SetSize implements sizing.
func (m *ScoreboardModel) SetSize(w, h int) {
	m.width, m.height = w, h

	// Apply Rule #1 (Borders accounting):
	tH := h - 14
	if tH < 5 {
		tH = 5
	}
	m.table.SetHeight(tH)

	tW := w - 8
	if tW < 60 {
		tW = 60
	}
	m.table.SetWidth(tW)

	// Apply Rule #4 (Weights, Not Pixels):
	rankW := 8
	scoreW := 18
	solvedW := 14
	nameW := tW - rankW - scoreW - solvedW - 6
	if nameW < 20 {
		nameW = 20
	}

	cols := []table.Column{
		{Title: "RANK", Width: rankW},
		{Title: "TEAM IDENTITY", Width: nameW},
		{Title: "TOTAL BOUNTY", Width: scoreW},
		{Title: "SOLVED", Width: solvedW},
	}
	m.table.SetColumns(cols)
}

// Refresh reloads rows from the DB view and schedules the next auto tick.
func (m *ScoreboardModel) Refresh() tea.Cmd {
	entries, err := m.db.Scoreboard()
	if err == nil {
		m.totalTeams = len(entries)
		m.myRank = 0
		m.top3 = nil
		if len(entries) > 0 {
			max3 := 3
			if len(entries) < 3 {
				max3 = len(entries)
			}
			m.top3 = entries[:max3]
		}

		rows := make([]table.Row, 0, len(entries))
		for i, e := range entries {
			rankStr := fmt.Sprintf("%d", i+1)
			switch i {
			case 0:
				rankStr = "🥇 1"
			case 1:
				rankStr = "🥈 2"
			case 2:
				rankStr = "🥉 3"
			}

			name := middleware.SanitizeInput(middleware.StripANSI(e.TeamName))
			if m.team != nil && e.TeamID == m.team.ID {
				name = "► [YOU] " + name
				m.myRank = i + 1
			}

			scoreStr := formatPoints(e.TotalScore) + " pts"
			solvedStr := fmt.Sprintf("%d rounds", e.RoundsSolved)

			rows = append(rows, table.Row{
				rankStr,
				Truncate(name, 36),
				scoreStr,
				solvedStr,
			})
		}
		m.table.SetRows(rows)
		m.lastUpdated = time.Now()
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

// View renders podium + table + live telemetry.
func (m ScoreboardModel) View() string {
	w := m.width
	if w < 40 {
		w = 40
	}

	// 1. Top Bar / Podium Highlights
	var podiumCards []string
	if len(m.top3) > 0 {
		medals := []struct {
			badge string
			bg    string
			fg    string
		}{
			{"🥇 1ST PLACE", "#FFD700", "#0B0F19"},
			{"🥈 2ND PLACE", "#E2E8F0", "#0B0F19"},
			{"🥉 3RD PLACE", "#FF7A00", "#0B0F19"},
		}

		availW := w - 8
		cardW := (availW - (len(m.top3)-1)*2) / len(m.top3)
		if cardW < 22 {
			cardW = 22
		}

		for idx, entry := range m.top3 {
			if idx >= len(medals) {
				break
			}
			meta := medals[idx]
			tag := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(meta.fg)).
				Background(lipgloss.Color(meta.bg)).
				Padding(0, 1).
				Render(meta.badge)

			teamName := middleware.SanitizeInput(middleware.StripANSI(entry.TeamName))
			nameStyled := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F0F6FC")).Render(Truncate(teamName, cardW-4))
			scoreStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("#00F0FF")).
				Render(fmt.Sprintf("%s pts • %d solved", formatPoints(entry.TotalScore), entry.RoundsSolved))

			pCard := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#30363D")).
				Background(lipgloss.Color("#161B22")).
				Padding(0, 1).
				Width(cardW).
				Render(lipgloss.JoinVertical(lipgloss.Left, tag, nameStyled, scoreStyled))

			podiumCards = append(podiumCards, pCard)
		}
	}

	var podiumRow string
	if len(podiumCards) > 0 {
		podiumRow = lipgloss.JoinHorizontal(lipgloss.Top, podiumCards...)
	}

	// 2. Table Box
	tableBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#30363D")).
		Background(lipgloss.Color("#161B22")).
		Padding(0, 1).
		Render(m.table.View())

	// 3. Telemetry Footer
	pulseDot := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF9D")).Render("● ")
	syncInfo := lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).
		Render(fmt.Sprintf("Live Auto-Sync Active (every 15s) • Last sync: %s", m.lastUpdated.Format("15:04:05")))

	myRankStr := "Unranked"
	if m.myRank > 0 {
		myRankStr = fmt.Sprintf("#%d of %d", m.myRank, m.totalTeams)
	}

	rankBadge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#00F0FF")).Padding(0, 1).
		Render(fmt.Sprintf("YOUR STANDING: %s", myRankStr))

	telemetryRow := lipgloss.JoinHorizontal(lipgloss.Center, pulseDot+syncInfo, "    ", rankBadge)

	keyHelp := lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).
		Render("[↑/↓] Scroll Standings  •  [r] Refresh  •  [Esc] Back to Deck")

	content := lipgloss.JoinVertical(lipgloss.Left,
		podiumRow,
		"",
		tableBox,
		"",
		telemetryRow,
		"",
		keyHelp,
	)

	return lipgloss.NewStyle().
		Width(m.width).
		Padding(1, 2).
		Render(content)
}

