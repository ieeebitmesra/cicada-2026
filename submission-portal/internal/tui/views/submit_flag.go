package views

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ieee-ctf/internal/models"
	"ieee-ctf/internal/scoring"
	"ieee-ctf/internal/store"
	"ieee-ctf/internal/tui/msg"
)

type submitPhase int

const (
	submitSelectRound submitPhase = iota
	submitInputFlag
)

type roundItem struct {
	id     int
	title  string
	desc   string
	open   bool
}

func (i roundItem) Title() string       { return i.title }
func (i roundItem) Description() string { return i.desc }
func (i roundItem) FilterValue() string { return i.title }

// SubmitFlagModel lets the player pick a round and submit a flag.
type SubmitFlagModel struct {
	width, height int
	phase         submitPhase

	list  list.Model
	input textinput.Model

	svc      *scoring.Service
	team     *models.Team
	selected models.Round
}

// NewSubmitFlag builds the flag submission view.
func NewSubmitFlag(svc *scoring.Service, team *models.Team) SubmitFlagModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select Round"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	input := textinput.New()
	input.Placeholder = "IEEE{...}"
	input.CharLimit = 256
	input.Prompt = "flag> "

	return SubmitFlagModel{list: l, input: input, svc: svc, team: team}
}

// SetSize implements sizing.
func (m *SubmitFlagModel) SetSize(w, h int) {
	m.width, m.height = w, h
	m.list.SetSize(w-4, h-8)
	inputWidth := 60
	if w > 12 {
		inputWidth = w - 12
	}
	m.input.Width = inputWidth
}

// Refresh reloads round availability.
func (m *SubmitFlagModel) Refresh() tea.Cmd {
	rounds, err := m.svc.DB.ListRounds(true)
	if err != nil {
		return nil
	}
	var items []list.Item
	for _, r := range rounds {
		solved, _ := m.svc.DB.HasSolvedRound(m.team.ID, r.ID)
		skipped, _ := m.svc.DB.HasSkipped(m.team.ID, r.ID)
		status := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("open")
		if solved {
			status = lipgloss.NewStyle().Foreground(lipgloss.Color("#00C853")).Render("solved ✓")
		} else if skipped {
			status = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5252")).Render("skipped ✗")
		}
		items = append(items, roundItem{
			id:    r.ID,
			title: fmt.Sprintf("Round %d — %s (%d pts)", r.ID, r.Name, r.Points),
			desc:  "status: " + status,
			open:  !solved && !skipped,
		})
	}
	cmd := m.list.SetItems(items)
	m.phase = submitSelectRound
	m.input.Blur()
	return cmd
}

// Update handles keys per phase.
func (m SubmitFlagModel) Update(msg_ tea.Msg) (SubmitFlagModel, tea.Cmd) {
	if key, ok := msg_.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			if m.phase == submitInputFlag {
				return m, m.backToSelect()
			}
			return m, func() tea.Msg { return msg.Navigate{Target: msg.TDashboard} }
		case "ctrl+c":
			return m, tea.Quit
		}
	}

	switch m.phase {
	case submitSelectRound:
		if key, ok := msg_.(tea.KeyMsg); ok && key.String() == "enter" {
			if it, ok := m.list.SelectedItem().(roundItem); ok {
				for _, r := range m.roundsFromList() {
					if r.ID != it.id {
						continue
					}
					if !it.open {
						flash := func() tea.Msg {
							return msg.StatusFlash{Text: "That round is closed for you (solved or skipped).", Success: false}
						}
						return m, flash
					}
					m.selected = r
					m.phase = submitInputFlag
					m.input.SetValue("")
					m.input.Focus()
					return m, textinput.Blink
				}
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg_)
		return m, cmd

	default: // submitInputFlag
		if key, ok := msg_.(tea.KeyMsg); ok && (key.String() == "enter" || key.String() == "ctrl+s") {
			return m, m.submit()
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg_)
		return m, cmd
	}
}

func (m *SubmitFlagModel) backToSelect() tea.Cmd {
	m.phase = submitSelectRound
	m.input.Blur()
	return nil
}

func (m *SubmitFlagModel) roundsFromList() []models.Round {
	rounds, err := m.svc.DB.ListRounds(true)
	if err != nil {
		return nil
	}
	return rounds
}

func (m *SubmitFlagModel) submit() tea.Cmd {
	raw := m.input.Value()
	res, err := m.svc.SubmitFlag(m.team, m.selected.ID, raw)
	if err != nil {
		text := "Submission failed."
		switch {
		case errors.Is(err, scoring.ErrRateLimited):
			text = "Too many submissions — slow down."
		case errors.Is(err, store.ErrAlreadySolved):
			text = "Already solved this round!"
		case errors.Is(err, scoring.ErrAlreadySkipped):
			text = "Round was already skipped."
		case errors.Is(err, scoring.ErrRoundInactive):
			text = "Round is not active."
		case errors.Is(err, scoring.ErrFlagFormat):
			text = "Invalid format. Flags look like IEEE{...}"
		case errors.Is(err, scoring.ErrFlagTooLong):
			text = "Flag is too long."
		case strings.Contains(err.Error(), "cooldown"):
			text = err.Error()
		}
		flashErr := func() tea.Msg { return msg.StatusFlash{Text: text, Success: false} }
		return flashErr
	}
	if !res.Correct {
		flashWrong := func() tea.Msg { return msg.StatusFlash{Text: "Incorrect flag.", Success: false} }
		return flashWrong
	}
	flashOK := func() tea.Msg { return msg.StatusFlash{Text: fmt.Sprintf("Correct! +%s points!", formatPoints(res.PointsAwarded)), Success: true} }
	nav := func() tea.Msg { return msg.Navigate{Target: msg.TDashboard} }
	return tea.Sequence(flashOK, nav)
}

// View renders the phase-appropriate UI.
func (m SubmitFlagModel) View() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Submit Flag") + "\n\n")

	if m.phase == submitSelectRound {
		b.WriteString(m.list.View())
	} else {
		b.WriteString(fmt.Sprintf("Target: Round %d — %s (%d pts)\n\n", m.selected.ID, m.selected.Name, m.selected.Points))
		b.WriteString(m.input.View() + "\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(
			"Flags are case-sensitive and look like IEEE{...}. Attempts are rate-limited.") + "\n")
		b.WriteString("[Enter] submit  [Esc] back")
	}
	return lipgloss.NewStyle().Width(m.width).Padding(1, 2).Render(b.String())
}
