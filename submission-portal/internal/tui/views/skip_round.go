package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ieee-ctf/internal/models"
	"ieee-ctf/internal/scoring"
	"ieee-ctf/internal/tui/components"
	"ieee-ctf/internal/tui/msg"
)

const confirmSkipID = 1

// SkipRoundModel confirms and executes a round skip.
type SkipRoundModel struct {
	width, height int

	list    list.Model
	confirm components.Confirm
	cost    float64
	total   float64
	pending models.Round

	svc  *scoring.Service
	team *models.Team
}

// NewSkipRound builds the skip view.
func NewSkipRound(svc *scoring.Service, team *models.Team) SkipRoundModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Skip Round — select a round"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	return SkipRoundModel{
		list:    l,
		confirm: components.NewConfirm(confirmSkipID, "Skip this round?", nil),
		svc:     svc,
		team:    team,
	}
}

// SetSize implements sizing.
func (m *SkipRoundModel) SetSize(w, h int) {
	m.width, m.height = w, h
	m.list.SetSize(w-4, h-8)
	m.confirm.SetSize(w, h)
}

// Refresh reloads skippable rounds.
func (m *SkipRoundModel) Refresh() tea.Cmd {
	rounds, err := m.svc.DB.ListRounds(true)
	if err != nil {
		return nil
	}
	var items []list.Item
	for _, r := range rounds {
		solved, _ := m.svc.DB.HasSolvedRound(m.team.ID, r.ID)
		skipped, _ := m.svc.DB.HasSkipped(m.team.ID, r.ID)
		if solved || skipped {
			continue
		}
		items = append(items, roundItem{
			id:    r.ID,
			title: fmt.Sprintf("Round %d — %s (%d pts)", r.ID, r.Name, r.Points),
			desc:  fmt.Sprintf("%d points would be forfeited", r.Points),
			open:  true,
		})
	}
	cmd := m.list.SetItems(items)
	m.confirm.Deactivate()
	return cmd
}

// Update handles keys + confirm result.
func (m SkipRoundModel) Update(msg_ tea.Msg) (SkipRoundModel, tea.Cmd) {
	if res, ok := msg_.(msg.ConfirmResult); ok && res.ID == confirmSkipID {
		if !res.Yes || res.Cancelled {
			return m, nil
		}
		return m, m.doSkip()
	}

	if key, ok := msg_.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if !m.confirm.Active() {
				return m, func() tea.Msg { return msg.Navigate{Target: msg.TDashboard} }
			}
		}
	}

	// Dialog gets priority when armed.
	if consumed, cmd := m.confirm.Update(msg_); consumed {
		return m, cmd
	}

	if key, ok := msg_.(tea.KeyMsg); ok && key.String() == "enter" && !m.confirm.Active() {
		if it, ok := m.list.SelectedItem().(roundItem); ok {
			for _, r := range m.loadRounds() {
				if r.ID != it.id {
					continue
				}
				m.pending = r
				cost, err := m.svc.PreviewSkipCost(m.team.ID)
				if err != nil {
					flashErr := func() tea.Msg { return msg.StatusFlash{Text: "Preview failed: " + err.Error(), Success: false} }
					return m, flashErr
				}
				b, _ := m.svc.Breakdown(m.team.ID)
				m.cost = cost
				m.total = b.Total
				details := []string{
					fmt.Sprintf("Round %d — %s (%d pts)", r.ID, r.Name, r.Points),
					fmt.Sprintf("Current total: %s pts", formatPoints(m.total)),
					lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5252")).
						Render(fmt.Sprintf("Penalty: −%s pts", formatPoints(cost))),
				}
				m.confirm = components.NewConfirm(confirmSkipID, "Skip this round?", details)
				m.confirm.Activate(false)
				return m, nil
			}
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg_)
	return m, cmd
}

func (m *SkipRoundModel) loadRounds() []models.Round {
	rounds, err := m.svc.DB.ListRounds(true)
	if err != nil {
		return nil
	}
	return rounds
}

func (m *SkipRoundModel) doSkip() tea.Cmd {
	cost, err := m.svc.SkipRound(m.team, m.pending.ID)
	if err != nil {
		flashErr := func() tea.Msg { return msg.StatusFlash{Text: "Skip failed: " + err.Error(), Success: false} }
		return flashErr
	}
	flashOK := func() tea.Msg {
		return msg.StatusFlash{Text: fmt.Sprintf("Round %d skipped (−%s pts).", m.pending.ID, formatPoints(cost)), Success: true}
	}
	nav := func() tea.Msg { return msg.Navigate{Target: msg.TDashboard} }
	return tea.Sequence(flashOK, nav)
}

// View renders the phase UI.
func (m SkipRoundModel) View() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Skip Round") + "\n\n")
	if m.confirm.Active() {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB300")).
			Render("⚠ Skipping costs 50% of your CURRENT total — choose carefully.") + "\n\n")
		b.WriteString(m.confirm.View())
	} else if len(m.list.Items()) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).
			Render("Nothing left to skip.") + "\n")
	} else {
		b.WriteString(m.list.View())
	}
	return lipgloss.NewStyle().Width(m.width).Padding(1, 2).Render(b.String())
}
