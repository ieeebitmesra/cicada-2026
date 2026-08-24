package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ieee-ctf/internal/auth"
	"ieee-ctf/internal/models"
	"ieee-ctf/internal/scoring"
	"ieee-ctf/internal/tui/msg"
)

type hintPhase int

const (
	hintSelectRound hintPhase = iota
	hintSelectType
	hintSign
	hintDone
)

type hintTypeItem struct {
	typ  string // "plain" | "encoded"
	cost string
}

func (i hintTypeItem) Title() string       { return i.typ + " hint — cost " + i.cost }
func (i hintTypeItem) Description() string { return "" }
func (i hintTypeItem) FilterValue() string { return i.typ }

// RequestHintModel walks the PGP-signed hint request flow.
type RequestHintModel struct {
	width, height int
	phase         hintPhase

	list  list.Model
	typ   list.Model
	paste textarea.Model

	svc      *scoring.Service
	team     *models.Team
	selected models.Round
	typeSel  string
	cost     float64

	challenge string
	result    string
}

// NewRequestHint builds the hint flow view.
func NewRequestHint(svc *scoring.Service, team *models.Team) RequestHintModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Request Hint — select round"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	t := list.New([]list.Item{
		hintTypeItem{"plain", "20% of round"},
		hintTypeItem{"encoded", "10% of round (you decode it)"},
	}, list.NewDefaultDelegate(), 0, 0)
	t.Title = "Select hint type"
	t.SetShowStatusBar(false)
	t.SetFilteringEnabled(false)
	t.SetShowHelp(false)

	paste := textarea.New()
	paste.Placeholder = "-----BEGIN PGP SIGNED MESSAGE-----\n...\n-----END PGP SIGNATURE-----"
	paste.CharLimit = 8192
	paste.ShowLineNumbers = false

	return RequestHintModel{list: l, typ: t, paste: paste, svc: svc, team: team}
}

// SetSize implements sizing.
func (m *RequestHintModel) SetSize(w, h int) {
	m.width, m.height = w, h
	m.list.SetSize(w-4, h-8)
	m.typ.SetSize(w-4, h-8)
	if w > 10 {
		m.paste.SetWidth(w - 10)
	}
	m.paste.SetHeight(h / 3)
}

// Refresh resets to round selection.
func (m *RequestHintModel) Refresh() tea.Cmd {
	rounds, err := m.svc.DB.ListRounds(true)
	if err != nil {
		return nil
	}
	var items []list.Item
	for _, r := range rounds {
		solved, _ := m.svc.DB.HasSolvedRound(m.team.ID, r.ID)
		if solved {
			continue
		}
		items = append(items, roundItem{
			id:    r.ID,
			title: fmt.Sprintf("Round %d — %s (%d pts)", r.ID, r.Name, r.Points),
			desc:  fmt.Sprintf("hints available: plain ×%d • encoded ×%d",
				m.svc.Rounds.CountHints(r.ID, "plain"), m.svc.Rounds.CountHints(r.ID, "encoded")),
			open: true,
		})
	}
	cmd := m.list.SetItems(items)
	m.phase = hintSelectRound
	return cmd
}

// Update handles the multi-phase flow.
func (m RequestHintModel) Update(msg_ tea.Msg) (RequestHintModel, tea.Cmd) {
	if key, ok := msg_.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			switch m.phase {
			case hintSign:
				m.phase = hintSelectType
				m.paste.Blur()
				return m, nil
			case hintSelectType:
				m.phase = hintSelectRound
				return m, nil
			default:
				return m, func() tea.Msg { return msg.Navigate{Target: msg.TDashboard} }
			}
		case "ctrl+s":
			if m.phase == hintSign {
				return m, m.redeem()
			}
		}
	}

	switch m.phase {
	case hintSelectRound:
		if key, ok := msg_.(tea.KeyMsg); ok && key.String() == "enter" {
			if it, ok := m.list.SelectedItem().(roundItem); ok {
				for _, r := range m.loadRounds() {
					if r.ID == it.id {
						m.selected = r
						break
					}
				}
				m.phase = hintSelectType
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg_)
		return m, cmd

	case hintSelectType:
		if key, ok := msg_.(tea.KeyMsg); ok && key.String() == "enter" {
			if it, ok := m.typ.SelectedItem().(hintTypeItem); ok {
				m.typeSel = it.typ
				return m, m.issueChallenge()
			}
		}
		var cmd tea.Cmd
		m.typ, cmd = m.typ.Update(msg_)
		return m, cmd

	case hintSign:
		var cmd tea.Cmd
		m.paste, cmd = m.paste.Update(msg_)
		return m, cmd

	default: // hintDone
		if key, ok := msg_.(tea.KeyMsg); ok && key.String() == "enter" {
			return m, func() tea.Msg { return msg.Navigate{Target: msg.TDashboard} }
		}
		return m, nil
	}
}

func (m *RequestHintModel) loadRounds() []models.Round {
	rounds, err := m.svc.DB.ListRounds(true)
	if err != nil {
		return nil
	}
	return rounds
}

func (m *RequestHintModel) issueChallenge() tea.Cmd {
	challenge, _, cost, err := m.svc.HintChallenge(m.team, m.selected.ID, m.typeSel)
	if err != nil {
		text := "Cannot dispense hint: " + err.Error()
		return func() tea.Msg { return msg.StatusFlash{Text: text, Success: false} }
	}
	m.challenge = challenge
	m.cost = cost
	m.phase = hintSign
	m.paste.Reset()
	m.paste.Focus()
	return textarea.Blink
}

func (m *RequestHintModel) redeem() tea.Cmd {
	body, cost, err := m.svc.RedeemHint(m.team, m.selected.ID, m.typeSel, m.paste.Value())
	if err != nil {
		text := "Verification failed. Hint not dispensed."
		if err != scoring.ErrNoHintsLeft {
			text += " (" + err.Error() + ")"
		}
		flashErr := func() tea.Msg { return msg.StatusFlash{Text: text, Success: false} }
		return flashErr
	}
	m.result = body
	m.cost = cost
	m.phase = hintDone
	flashOK := func() tea.Msg {
		return msg.StatusFlash{Text: fmt.Sprintf("Hint dispensed (−%s pts).", formatPoints(m.cost)), Success: true}
	}
	return flashOK
}

// View renders the current phase.
func (m RequestHintModel) View() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Request Hint") + "\n\n")

	switch m.phase {
	case hintSelectRound:
		b.WriteString(m.list.View())

	case hintSelectType:
		b.WriteString(fmt.Sprintf("Round %d — %s\n", m.selected.ID, m.selected.Name))
		plainLeft := m.svc.Rounds.CountHints(m.selected.ID, "plain")
		encLeft := m.svc.Rounds.CountHints(m.selected.ID, "encoded")
		usedP, _ := m.svc.DB.CountHintsUsed(m.team.ID, m.selected.ID, "plain")
		usedE, _ := m.svc.DB.CountHintsUsed(m.team.ID, m.selected.ID, "encoded")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).
			Render(fmt.Sprintf("remaining: plain %d/%d • encoded %d/%d\n\n",
				max0(plainLeft-usedP), plainLeft, max0(encLeft-usedE), encLeft)))
		b.WriteString(m.typ.View() + "\n")
		b.WriteString("[↑/↓] choose  [Enter] next  [Esc] back")

	case hintSign:
		fp := auth.PublicKeyFingerprint(m.team.PGPPubkey)
		b.WriteString("Step 1 — clearsign this challenge locally:\n\n")
		b.WriteString(lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#FFB300")).
			Padding(0, 1).
			Render(m.challenge) + "\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).
			Render(fmt.Sprintf("  $ gpg --clearsign --local-user ...%s\n", fp)) + "\n")
		b.WriteString("Step 2 — paste the signed message below, then press Ctrl+S:\n\n")
		b.WriteString(m.paste.View())

	default:
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00C853")).Bold(true).
			Render(fmt.Sprintf("Hint (%s, −%s pts):", m.typeSel, formatPoints(m.cost))) + "\n\n")
		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#00C853")).
			Padding(1, 2).
			Render(m.result)
		b.WriteString(box + "\n\n[Enter] back to dashboard")
	}
	return lipgloss.NewStyle().Width(m.width).Padding(1, 2).Render(b.String())
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
