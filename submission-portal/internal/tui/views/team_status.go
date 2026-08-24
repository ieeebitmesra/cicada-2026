package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
	vw, vh := w-6, h-7
	if vw < 20 {
		vw = 20
	}
	if vh < 3 {
		vh = 3
	}
	m.vp.Width = vw
	m.vp.Height = vh
}

// Refresh rebuilds the report.
func (m *TeamStatusModel) Refresh() tea.Cmd {
	m.vp.SetContent(m.buildReport())
	return nil
}

func (m *TeamStatusModel) buildReport() string {
	var b strings.Builder
	bd, err := m.svc.Breakdown(m.team.ID)
	if err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5252")).Render("Failed to load status: " + err.Error())
	}

	b.WriteString("Team: " + m.team.Name + "\n")
	fp := "(no PGP key registered)"
	if m.team.PGPPubkey != "" {
		fp = "registered"
	}
	b.WriteString("PGP key: " + fp + "\n\n")

	b.WriteString("Round progress:\n")
	rounds, _ := m.svc.DB.ListRounds(false)
	solvedSet := map[int]bool{}
	for _, id := range bd.SolvedRounds {
		solvedSet[id] = true
	}
	skippedSet := map[int]bool{}
	for _, sk := range bd.Skips {
		skippedSet[sk.RoundID] = true
	}
	for _, r := range rounds {
		mark := "[ ]"
		color := "#6B7280"
		note := fmt.Sprintf("%d pts", r.Points)
		switch {
		case solvedSet[r.ID]:
			mark, color, note = "[✓]", "#00C853", fmt.Sprintf("+%d pts", r.Points)
		case skippedSet[r.ID]:
			mark, color, note = "[S]", "#FF5252", "skipped"
		case !r.IsActive:
			mark, color, note = "[-]", "#6B7280", "inactive"
		}
		line := fmt.Sprintf("  %s Round %d — %s (%s)", mark, r.ID, r.Name, note)
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(line) + "\n")
	}

	b.WriteString("\nPoints ledger:\n")
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#00C853"))
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5252"))
	b.WriteString(green.Render(fmt.Sprintf("  earned from rounds ....... +%s", formatPoints(bd.Earned))) + "\n")
	b.WriteString(red.Render(fmt.Sprintf("  hint costs .............. −%s", formatPoints(bd.HintCosts))) + "\n")
	b.WriteString(red.Render(fmt.Sprintf("  skip penalties .......... −%s", formatPoints(bd.SkipCosts))) + "\n")

	totalStyle := lipgloss.NewStyle().Bold(true)
	if bd.Total < 0 {
		totalStyle = totalStyle.Foreground(lipgloss.Color("#FF5252"))
	} else {
		totalStyle = totalStyle.Foreground(lipgloss.Color("#00C853"))
	}
	b.WriteString(totalStyle.Render(fmt.Sprintf("\n  TOTAL ................... %s pts", formatPoints(bd.Total))) + "\n")

	if len(bd.Hints) > 0 {
		b.WriteString("\nHints used:\n")
		for _, h := range bd.Hints {
			b.WriteString(fmt.Sprintf("  round %d • %-7s #%d • −%s pts\n",
				h.RoundID, h.HintType, h.HintIndex+1, formatPoints(h.CostPoints)))
		}
	}
	return b.String()
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
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("My Status") + "\n\n")
	b.WriteString(m.vp.View())
	return lipgloss.NewStyle().Width(m.width).Padding(1, 2).Render(b.String())
}
