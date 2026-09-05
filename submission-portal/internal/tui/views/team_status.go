package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ieee-ctf/internal/auth"
	"ieee-ctf/internal/models"
	"ieee-ctf/internal/scoring"
	"ieee-ctf/internal/tui/msg"
)

// TeamStatusModel shows per-team progress and the points ledger.
type TeamStatusModel struct {
	width, height int
	vp            viewport.Model

	svc  *scoring.Service
	team *models.Team
}

// NewTeamStatus builds the status view.
func NewTeamStatus(svc *scoring.Service, team *models.Team) TeamStatusModel {
	vp := viewport.New(80, 20)
	vp.SetContent("")
	return TeamStatusModel{vp: vp, svc: svc, team: team}
}

// SetSize implements sizing.
func (m *TeamStatusModel) SetSize(w, h int) {
	m.width, m.height = w, h
	vw, vh := w-6, h-8
	if vw < 30 {
		vw = 30
	}
	if vh < 4 {
		vh = 4
	}
	m.vp.Width = vw
	m.vp.Height = vh
	m.vp.SetContent(m.buildReport())
}

// Refresh rebuilds the report.
func (m *TeamStatusModel) Refresh() tea.Cmd {
	m.vp.SetContent(m.buildReport())
	return nil
}

func (m *TeamStatusModel) buildReport() string {
	bd, err := m.svc.Breakdown(m.team.ID)
	if err != nil {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EF4444")).
			Render("Failed to load status ledger: " + err.Error())
	}

	w := m.width
	if w < 40 {
		w = 40
	}

	// 1. Top Dossier Summary Card
	teamPill := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#00B4D8")).
		Padding(0, 1).
		Render("TEAM: " + m.team.Name)

	var pgpPill string
	if m.team.PGPPubkey != "" {
		keyID := auth.PublicKeyID(m.team.PGPPubkey)
		pgpPill = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#0B0F19")).
			Background(lipgloss.Color("#00FF9D")).
			Padding(0, 1).
			Render("PGP KEY: " + keyID)
	} else {
		pgpPill = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F8FAFC")).
			Background(lipgloss.Color("#FF3860")).
			Padding(0, 1).
			Render("PGP: UNCONFIGURED")
	}

	rounds, _ := m.svc.DB.ListRounds(false)
	totalRounds := len(rounds)
	solvedCount := len(bd.SolvedRounds)
	progressBar := renderBar(solvedCount, totalRounds, 12)

	summaryRow := lipgloss.JoinHorizontal(lipgloss.Center, teamPill, "  ", pgpPill, "   Progress: ", progressBar)

	summaryBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#30363D")).
		Background(lipgloss.Color("#161B22")).
		Padding(0, 1).
		Width(w - 10).
		Render(summaryRow)

	// 2. Challenge Matrix Section
	var matrixBuilder strings.Builder
	matrixTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#00F0FF")).
		Padding(0, 1).
		Render(" CHALLENGE ENGAGEMENT MATRIX ")

	matrixBuilder.WriteString(matrixTitle + "\n\n")

	solvedSet := map[int]bool{}
	for _, id := range bd.SolvedRounds {
		solvedSet[id] = true
	}
	skippedSet := map[int]bool{}
	for _, sk := range bd.Skips {
		skippedSet[sk.RoundID] = true
	}
	solvesCountMap, _ := m.svc.DB.SolvesCountMap()

	for _, r := range rounds {
		limitSolves := r.LimitSolves || m.svc.Rounds.IsSolveLimited(r.ID)
		maxSolves := r.MaxSolves
		if maxSolves == 0 && limitSolves {
			maxSolves = m.svc.Rounds.MaxAllowedSolves(r.ID)
		}
		solveCount := solvesCountMap[r.ID]
		quotaFull := limitSolves && maxSolves > 0 && solveCount >= maxSolves

		var badge, note string
		switch {
		case solvedSet[r.ID]:
			badge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
				Background(lipgloss.Color("#00FF9D")).Padding(0, 1).Render("✓ SOLVED")
			note = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF9D")).
				Render(fmt.Sprintf("+%d pts", r.Points))
		case skippedSet[r.ID]:
			badge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
				Background(lipgloss.Color("#FF3860")).Padding(0, 1).Render("✗ SKIPPED")
			note = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3860")).
				Render(fmt.Sprintf("−%s pts", formatPoints(scoring.SkipCost(r.Points))))
		case !r.IsActive:
			badge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8B949E")).
				Background(lipgloss.Color("#30363D")).Padding(0, 1).Render("— INACTIVE")
			note = lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).
				Render(fmt.Sprintf("%d pts", r.Points))
		case quotaFull:
			badge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
				Background(lipgloss.Color("#FF3860")).Padding(0, 1).Render("🔒 QUOTA FULL")
			note = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3860")).
				Render(fmt.Sprintf("0 pts (%d/%d solves reached)", solveCount, maxSolves))
		case limitSolves && maxSolves > 0:
			left := maxSolves - solveCount
			if left < 0 {
				left = 0
			}
			badge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
				Background(lipgloss.Color("#F59E0B")).Padding(0, 1).Render(fmt.Sprintf("⚡ %d/%d LEFT", left, maxSolves))
			note = lipgloss.NewStyle().Foreground(lipgloss.Color("#C9D1D9")).
				Render(fmt.Sprintf("%d pts", r.Points))
		default:
			badge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
				Background(lipgloss.Color("#00F0FF")).Padding(0, 1).Render("⚡ OPEN")
			note = lipgloss.NewStyle().Foreground(lipgloss.Color("#C9D1D9")).
				Render(fmt.Sprintf("%d pts", r.Points))
		}

		line := fmt.Sprintf(" %s Round %-2d — %-24s %s", badge, r.ID, r.Name, note)
		if r.Description != "" {
			line += fmt.Sprintf("\n    ↳ %s", lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).Render(r.Description))
		}
		matrixBuilder.WriteString(line + "\n")
	}

	// 3. Points Ledger Breakdown Section
	var ledgerBuilder strings.Builder
	ledgerTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#00FF9D")).
		Padding(0, 1).
		Render(" FINANCIAL BOUNTY LEDGER ")

	ledgerBuilder.WriteString(ledgerTitle + "\n\n")

	green := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF9D"))
	red := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF3860"))

	ledgerBuilder.WriteString(green.Render(fmt.Sprintf(" + Gross Bounty Earned : +%s PTS", formatPoints(bd.Earned))) + "\n")
	ledgerBuilder.WriteString(red.Render(fmt.Sprintf(" − Hint Cost Deductions : −%s PTS", formatPoints(bd.HintCosts))) + "\n")
	ledgerBuilder.WriteString(red.Render(fmt.Sprintf(" − Skip Penalties       : −%s PTS", formatPoints(bd.SkipCosts))) + "\n")
	ledgerBuilder.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#30363D")).Render(strings.Repeat("─", 38)) + "\n")

	netStyle := lipgloss.NewStyle().Bold(true)
	if bd.Total < 0 {
		netStyle = netStyle.Foreground(lipgloss.Color("#FF3860"))
	} else {
		netStyle = netStyle.Foreground(lipgloss.Color("#00FF9D"))
	}
	ledgerBuilder.WriteString(netStyle.Render(fmt.Sprintf(" NET CURRENT TOTAL      : %s PTS", formatPoints(bd.Total))) + "\n")

	// Side-by-side or stacked layout
	var middleRow string
	colW := (w - 14) / 2
	if colW >= 42 {
		box1 := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Background(lipgloss.Color("#161B22")).
			Padding(0, 1).
			Width(colW).
			Render(matrixBuilder.String())

		box2 := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Background(lipgloss.Color("#161B22")).
			Padding(0, 1).
			Width(colW).
			Render(ledgerBuilder.String())

		middleRow = lipgloss.JoinHorizontal(lipgloss.Top, box1, "  ", box2)
	} else {
		fullW := w - 10
		box1 := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Background(lipgloss.Color("#161B22")).
			Padding(0, 1).
			Width(fullW).
			Render(matrixBuilder.String())

		box2 := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Background(lipgloss.Color("#161B22")).
			Padding(0, 1).
			Width(fullW).
			Render(ledgerBuilder.String())

		middleRow = lipgloss.JoinVertical(lipgloss.Left, box1, "\n", box2)
	}

	// 4. Hints Inventory / History Section
	var hintsSection string
	if len(bd.UnlockedHints) > 0 {
		var hBuilder strings.Builder
		hTitle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#0B0F19")).
			Background(lipgloss.Color("#FFB800")).
			Padding(0, 1).
			Render(" TACTICAL INTEL (UNLOCKED HINTS ARCHIVE) ")
		hBuilder.WriteString(hTitle + "\n\n")

		cardW := w - 16
		if cardW < 24 {
			cardW = 24
		}

		for _, h := range bd.UnlockedHints {
			typPill := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
				Background(lipgloss.Color("#00F0FF")).Padding(0, 1).Render(strings.ToUpper(h.HintType) + " #" + fmt.Sprint(h.HintIndex+1))

			costTag := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF3860")).
				Render(fmt.Sprintf("−%s PTS", formatPoints(h.CostPoints)))

			roundLabel := fmt.Sprintf("Round %d", h.RoundID)
			if h.RoundName != "" {
				roundLabel = fmt.Sprintf("Round %d — %s", h.RoundID, h.RoundName)
			}

			headerLine := lipgloss.JoinHorizontal(lipgloss.Center,
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F0F6FC")).Render(roundLabel),
				"  ", typPill, "  ", costTag)

			clueText := lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#30363D")).
				Background(lipgloss.Color("#0D1117")).
				Foreground(lipgloss.Color("#00FF9D")).
				Padding(0, 1).
				Width(cardW).
				Render(h.Text)

			hBuilder.WriteString(headerLine + "\n" + clueText + "\n\n")
		}

		hintsSection = "\n" + lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Background(lipgloss.Color("#161B22")).
			Padding(1, 2).
			Width(w - 10).
			Render(hBuilder.String())
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		summaryBox,
		"\n",
		middleRow,
		hintsSection,
	)
}

// Update handles scrolling keys.
func (m TeamStatusModel) Update(msg_ tea.Msg) (TeamStatusModel, tea.Cmd) {
	if key, ok := msg_.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, func() tea.Msg { return msg.Navigate{Target: msg.TDashboard} }
		case "r":
			return m, m.Refresh()
		}
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg_)
	return m, cmd
}

// View renders the scrollable report.
func (m TeamStatusModel) View() string {
	pct := fmt.Sprintf("%.0f%%", m.vp.ScrollPercent()*100)
	scrollBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#94A3B8")).
		Render("Scroll: [↑/↓] (" + pct + ")  •  [r] Refresh  •  [Esc] Back to Deck")

	return lipgloss.NewStyle().
		Width(m.width).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, m.vp.View(), "\n", scrollBadge))
}

