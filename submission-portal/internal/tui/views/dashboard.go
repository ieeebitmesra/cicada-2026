package views

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ieee-ctf/internal/models"
	"ieee-ctf/internal/scoring"
	"ieee-ctf/internal/tui/msg"
)

type menuItem struct {
	key    string
	icon   string
	title  string
	desc   string
	target msg.Target
	quit   bool
}

func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.desc }
func (i menuItem) FilterValue() string { return i.title }

// menuDelegate customizes the rendering of each Command Deck menu option.
type menuDelegate struct {
	width int
}

func (d menuDelegate) Height() int                             { return 2 }
func (d menuDelegate) Spacing() int                            { return 1 }
func (d menuDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d menuDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(menuItem)
	if !ok {
		return
	}

	selected := index == m.Index()
	maxWidth := d.width - 4
	if maxWidth < 20 {
		maxWidth = 20
	}

	var prefix, numKey, iconStyled, titleStyled, descStyled string

	numKey = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00B4D8")).
		Background(lipgloss.Color("#1E293B")).
		Padding(0, 1).
		Render("[" + item.key + "]")

	if selected {
		prefix = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00B4D8")).Render("▸ ")
		iconStyled = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8FAFC")).
			Background(lipgloss.Color("#00629B")).Padding(0, 1).Render(item.icon)
		titleStyled = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8FAFC")).
			Background(lipgloss.Color("#1E293B")).Padding(0, 1).Render(Truncate(item.title, maxWidth-12))
		descStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("#00B4D8")).Render("      " + Truncate(item.desc, maxWidth-8))
	} else {
		prefix = "  "
		iconStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B")).
			Background(lipgloss.Color("#111827")).Padding(0, 1).Render(item.icon)
		titleStyled = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E2E8F0")).Render(" " + Truncate(item.title, maxWidth-12))
		descStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B")).Render("      " + Truncate(item.desc, maxWidth-8))
	}

	row1 := prefix + numKey + " " + iconStyled + " " + titleStyled
	row2 := descStyled

	fmt.Fprintf(w, "%s\n%s", row1, row2)
}

// DashboardModel is the main Command Deck.
type DashboardModel struct {
	width, height int
	list          list.Model
	svc           *scoring.Service
	team          *models.Team

	score     float64
	solved    int
	totalRds  int
	hintsUsed int
	skipsUsed int
}

// NewDashboard builds the dashboard.
func NewDashboard(svc *scoring.Service, team *models.Team) DashboardModel {
	l := list.New([]list.Item{}, menuDelegate{width: 50}, 0, 0)
	l.Title = "COMMAND DECK ACTIONS"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#00B4D8")).
		Padding(0, 1)

	return DashboardModel{list: l, svc: svc, team: team}
}

// SetSize implements sizing.
func (m *DashboardModel) SetSize(w, h int) {
	m.width, m.height = w, h

	// Apply Rule #4 (Weights, Not Pixels):
	availableW := w - 6
	listW := availableW
	if w >= 90 {
		// 5:4 proportional split
		listW = (availableW * 5) / 9
	}
	if listW < 30 {
		listW = 30
	}

	// Apply Rule #1 (Borders accounting):
	listH := h - 8
	if listH < 6 {
		listH = 6
	}

	m.list.SetDelegate(menuDelegate{width: listW})
	m.list.SetSize(listW, listH)
}

// Refresh recomputes score + rebuilds the menu.
func (m *DashboardModel) Refresh() tea.Cmd {
	b, err := m.svc.Breakdown(m.team.ID)
	if err == nil && b != nil {
		m.score = b.Total
		m.solved = len(b.SolvedRounds)
		m.hintsUsed = len(b.Hints)
		m.skipsUsed = len(b.Skips)
	}
	if rounds, err := m.svc.DB.ListRounds(false); err == nil {
		m.totalRds = len(rounds)
	}

	items := []list.Item{
		menuItem{"1", "⚡", "Submit Flag", "Validate challenge token & claim bounty", msg.TSubmitFlag, false},
		menuItem{"2", "🔑", "Request Hint", "PGP challenge-response tactical intel", msg.TRequestHint, false},
		menuItem{"3", "⏩", "Skip Round", "Strategic bypass (−50% of round value)", msg.TSkipRound, false},
		menuItem{"4", "🏆", "Scoreboard", "Real-time global team leaderboard", msg.TScoreboard, false},
		menuItem{"5", "📊", "Team Dossier", "Progress breakdown & points ledger", msg.TTeamStatus, false},
		menuItem{"6", "🚪", "Disconnect", "Terminate secure SSH session", msg.Target(-1), true},
	}
	cmd := m.list.SetItems(items)
	return cmd
}

// Update handles keys.
func (m DashboardModel) Update(msg_ tea.Msg) (DashboardModel, tea.Cmd) {
	if key, ok := msg_.(tea.KeyMsg); ok {
		switch key.String() {
		case "1":
			return m, func() tea.Msg { return msg.Navigate{Target: msg.TSubmitFlag} }
		case "2":
			return m, func() tea.Msg { return msg.Navigate{Target: msg.TRequestHint} }
		case "3":
			return m, func() tea.Msg { return msg.Navigate{Target: msg.TSkipRound} }
		case "4":
			return m, func() tea.Msg { return msg.Navigate{Target: msg.TScoreboard} }
		case "5":
			return m, func() tea.Msg { return msg.Navigate{Target: msg.TTeamStatus} }
		case "6", "q":
			return m, tea.Quit
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

// View renders KPI stat cards + menu + tactical HUD panel.
func (m DashboardModel) View() string {
	w := m.width
	if w < 40 {
		w = 40
	}

	// 1. Proportional calculation for Top KPI Metric Cards (Golden Rule #4)
	availableW := w - 8
	numCards := 4
	cardWidth := (availableW - (numCards-1)*2) / numCards
	if cardWidth < 18 {
		cardWidth = 18
	}

	scoreColor := lipgloss.Color("#00FF9D")
	if m.score < 0 {
		scoreColor = lipgloss.Color("#FF3860")
	}

	pgpStatus := "CONFIGURED"
	pgpColor := lipgloss.Color("#00FF9D")
	if m.team.PGPPubkey == "" {
		pgpStatus = "UNCONFIGURED"
		pgpColor = lipgloss.Color("#FF3860")
	}

	scoreCard := renderMiniCard("TOTAL SCORE", fmt.Sprintf("%.1f PTS", m.score), "Live net bounty", cardWidth, scoreColor)
	progressCard := renderMiniCard("CHALLENGES", fmt.Sprintf("%d / %d SOLVED", m.solved, m.totalRds), renderBar(m.solved, m.totalRds, 8), cardWidth, lipgloss.Color("#00F0FF"))
	pgpCard := renderMiniCard("PGP IDENTITY", pgpStatus, "Clearsign active", cardWidth, pgpColor)
	intelCard := renderMiniCard("INTEL & SKIPS", fmt.Sprintf("%d hints • %d skips", m.hintsUsed, m.skipsUsed), "Penalty deductions", cardWidth, lipgloss.Color("#FFB800"))

	topCards := lipgloss.JoinHorizontal(lipgloss.Top, scoreCard, "  ", progressCard, "  ", pgpCard, "  ", intelCard)

	// 2. Main content area: Side-by-side (5:4 ratio) or stacked
	var mainArea string
	if w >= 90 {
		hudWidth := availableW - m.list.Width() - 2
		if hudWidth < 28 {
			hudWidth = 28
		}
		hud := m.renderTacticalHUD(hudWidth)
		mainArea = lipgloss.JoinHorizontal(lipgloss.Top, m.list.View(), "  ", hud)
	} else {
		mainArea = m.list.View()
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, topCards, "\n", mainArea))
}

func (m DashboardModel) renderTacticalHUD(width int) string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#00F0FF")).
		Padding(0, 1).
		Render(" TACTICAL HUD // ACTIVE DIRECTIVE ")

	maxTextW := width - 4
	if maxTextW < 10 {
		maxTextW = 10
	}

	var detail strings.Builder
	selected := m.list.SelectedItem()
	if it, ok := selected.(menuItem); ok {
		detail.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F0F6FC")).
			Render("\nSelected: "+it.icon+" "+Truncate(it.title, maxTextW-12)) + "\n")
		detail.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00F0FF")).
			Render(Truncate(it.desc, maxTextW)) + "\n\n")
	}

	detail.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#58A6FF")).
		Render("RULES & ADVISORY:") + "\n")
	detail.WriteString(Truncate("• Flag syntax strictly matches PANTHEON{...}", maxTextW) + "\n")
	detail.WriteString(Truncate("• Flag verification is rate-limited per IP/account", maxTextW) + "\n")
	detail.WriteString(Truncate("• Hints require local PGP clearsigning of nonce", maxTextW) + "\n")
	detail.WriteString(Truncate("• Skip penalties are permanent for the round", maxTextW) + "\n\n")

	detail.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).
		Render("Press [1-6] or [Enter] to activate directive"))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#30363D")).
		Background(lipgloss.Color("#161B22")).
		Padding(0, 1).
		Width(width).
		Render(title + detail.String())
}

func renderMiniCard(title, value, subtext string, width int, accent lipgloss.Color) string {
	innerW := width - 4
	if innerW < 10 {
		innerW = 10
	}

	tStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8B949E"))
	vStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	sStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#C9D1D9"))

	content := lipgloss.JoinVertical(lipgloss.Left,
		tStyle.Render(Truncate(title, innerW)),
		vStyle.Render(Truncate(value, innerW)),
		sStyle.Render(Truncate(subtext, innerW)),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#30363D")).
		Background(lipgloss.Color("#161B22")).
		Padding(0, 1).
		Width(width).
		Render(content)
}

func renderBar(cur, total, w int) string {
	if total <= 0 {
		return "[░░░░░░░░] 0%"
	}
	ratio := float64(cur) / float64(total)
	if ratio > 1.0 {
		ratio = 1.0
	}
	fill := int(ratio * float64(w))
	empty := w - fill
	if empty < 0 {
		empty = 0
	}
	fillPart := lipgloss.NewStyle().Foreground(lipgloss.Color("#00F0FF")).Render(strings.Repeat("█", fill))
	emptyPart := lipgloss.NewStyle().Foreground(lipgloss.Color("#30363D")).Render(strings.Repeat("░", empty))
	pct := fmt.Sprintf(" %.0f%%", ratio*100)
	return "[" + fillPart + emptyPart + "]" + pct
}


