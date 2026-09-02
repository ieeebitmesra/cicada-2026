package views

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
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
	hintConfirm
	hintDone
)

type hintTypeItem struct {
	typ  string // "plain" | "encoded"
	cost string
	desc string
}

func (i hintTypeItem) Title() string       { return i.typ + " hint — " + i.cost }
func (i hintTypeItem) Description() string { return i.desc }
func (i hintTypeItem) FilterValue() string { return i.typ }

type hintTypeDelegate struct {
	width int
}

func (d hintTypeDelegate) Height() int                             { return 2 }
func (d hintTypeDelegate) Spacing() int                            { return 1 }
func (d hintTypeDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d hintTypeDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(hintTypeItem)
	if !ok {
		return
	}

	selected := index == m.Index()

	badgeColor := "#00B4D8"
	if it.typ == "plain" {
		badgeColor = "#F59E0B"
	}
	badge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color(badgeColor)).Padding(0, 1).Render(strings.ToUpper(it.typ))

	costBadge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8FAFC")).
		Background(lipgloss.Color("#1E293B")).Padding(0, 1).Render(it.cost)

	var prefix, descStyled string
	if selected {
		prefix = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00B4D8")).Render("▸ ")
		descStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("#00B4D8")).Render("    " + it.desc)
	} else {
		prefix = "  "
		descStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B")).Render("    " + it.desc)
	}

	fmt.Fprintf(w, "%s%s  %s\n%s", prefix, badge, costBadge, descStyled)
}

// RequestHintModel walks the password-confirmed hint request flow.
type RequestHintModel struct {
	width, height int
	phase         hintPhase

	list    list.Model
	typ     list.Model
	passInp textinput.Model

	svc      *scoring.Service
	team     *models.Team
	selected models.Round
	typeSel  string
	cost     float64

	result string
}

// NewRequestHint builds the hint flow view.
func NewRequestHint(svc *scoring.Service, team *models.Team) RequestHintModel {
	l := list.New([]list.Item{}, flagRoundDelegate{}, 0, 0)
	l.Title = "SELECT TARGET CHALLENGE FOR INTEL"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#00B4D8")).
		Padding(0, 1)

	t := list.New([]list.Item{
		hintTypeItem{"plain", "Cost: 20% of round value", "Direct plain text clue • Immediate tactical advantage"},
		hintTypeItem{"encoded", "Cost: 10% of round value", "Encrypted / algorithmic puzzle clue • Lower penalty"},
	}, hintTypeDelegate{width: 54}, 0, 0)
	t.Title = "SELECT INTEL CLEARANCE TYPE"
	t.SetShowStatusBar(false)
	t.SetFilteringEnabled(false)
	t.SetShowHelp(false)
	t.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#F59E0B")).
		Padding(0, 1)

	passInp := textinput.New()
	passInp.Placeholder = "Enter your team password to confirm"
	passInp.EchoMode = textinput.EchoPassword
	passInp.CharLimit = 128
	passInp.Width = 44
	passInp.Prompt = "❯ "
	passInp.PromptStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00B4D8"))
	passInp.TextStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8FAFC"))
	passInp.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#475569"))

	return RequestHintModel{list: l, typ: t, passInp: passInp, svc: svc, team: team}
}

// SetSize implements sizing.
func (m *RequestHintModel) SetSize(w, h int) {
	m.width, m.height = w, h
	listW := w - 4
	listH := h - 8
	if listH < 6 {
		listH = 6
	}
	m.list.SetDelegate(flagRoundDelegate{width: listW})
	m.list.SetSize(listW, listH)
	m.typ.SetDelegate(hintTypeDelegate{width: listW})
	m.typ.SetSize(listW, listH)

	inputWidth := 50
	if w > 16 {
		inputWidth = w - 16
		if inputWidth > 64 {
			inputWidth = 64
		}
	}
	m.passInp.Width = inputWidth
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
		skipped, _ := m.svc.DB.HasSkipped(m.team.ID, r.ID)
		if skipped {
			continue
		}

		plainLeft := m.svc.Rounds.CountHints(r.ID, "plain")
		encLeft := m.svc.Rounds.CountHints(r.ID, "encoded")
		usedP, _ := m.svc.DB.CountHintsUsed(m.team.ID, r.ID, "plain")
		usedE, _ := m.svc.DB.CountHintsUsed(m.team.ID, r.ID, "encoded")

		desc := fmt.Sprintf("Hints remaining: plain %d/%d • encoded %d/%d",
			max0(plainLeft-usedP), plainLeft, max0(encLeft-usedE), encLeft)

		items = append(items, roundItem{
			id:     r.ID,
			title:  fmt.Sprintf("Round %d — %s", r.ID, r.Name),
			desc:   desc,
			points: r.Points,
			solved: false,
			skip:   false,
			open:   true,
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
			case hintConfirm:
				m.phase = hintSelectType
				m.passInp.Blur()
				m.passInp.SetValue("")
				return m, nil
			case hintSelectType:
				m.phase = hintSelectRound
				return m, nil
			default:
				return m, func() tea.Msg { return msg.Navigate{Target: msg.TDashboard} }
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
				// Calculate cost preview
				round := m.svc.Rounds.Def(m.selected.ID)
				if round != nil {
					m.cost = scoring.HintCost(round.Points, m.typeSel)
				}
				m.phase = hintConfirm
				m.passInp.SetValue("")
				m.passInp.Focus()
				return m, textinput.Blink
			}
		}
		var cmd tea.Cmd
		m.typ, cmd = m.typ.Update(msg_)
		return m, cmd

	case hintConfirm:
		if key, ok := msg_.(tea.KeyMsg); ok && (key.String() == "enter" || key.String() == "ctrl+s") {
			password := m.passInp.Value()
			if password == "" {
				flashErr := func() tea.Msg {
					return msg.StatusFlash{Text: "Enter your team password to confirm hint request.", Success: false}
				}
				return m, flashErr
			}

			// Verify password
			if !auth.VerifyTeamLogin(m.svc.DB, m.team.SSHUser, password) {
				flashErr := func() tea.Msg {
					return msg.StatusFlash{Text: "Incorrect password. Hint request denied.", Success: false}
				}
				m.passInp.SetValue("")
				return m, flashErr
			}

			// Password verified — dispense hint directly
			body, cost, err := m.svc.DirectRedeemHint(m.team, m.selected.ID, m.typeSel)
			if err != nil {
				text := "Verification failed. Hint not dispensed."
				switch {
				case errors.Is(err, scoring.ErrNoHintsLeft):
					text = "No hints of that type remain for this round."
				case errors.Is(err, scoring.ErrRoundInactive):
					text = "Round is not active."
				}
				flashErr := func() tea.Msg { return msg.StatusFlash{Text: text, Success: false} }
				return m, flashErr
			}
			m.result = body
			m.cost = cost
			m.phase = hintDone
			flashOK := func() tea.Msg {
				return msg.StatusFlash{Text: fmt.Sprintf("INTEL DISPENSED! (−%s pts).", formatPoints(m.cost)), Success: true}
			}
			return m, flashOK
		}

		var cmd tea.Cmd
		m.passInp, cmd = m.passInp.Update(msg_)
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

func (m RequestHintModel) renderWizardBar() string {
	steps := []struct {
		id    hintPhase
		label string
	}{
		{hintSelectRound, "1. SELECT ROUND"},
		{hintSelectType, "2. HINT TYPE"},
		{hintConfirm, "3. PASSWORD CONFIRMATION"},
		{hintDone, "4. UNLOCKED INTEL"},
	}

	var parts []string
	for _, s := range steps {
		if s.id == m.phase {
			// Current active
			parts = append(parts, lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#0B0F19")).
				Background(lipgloss.Color("#00F0FF")).
				Padding(0, 1).
				Render("► "+s.label))
		} else if s.id < m.phase {
			// Completed
			parts = append(parts, lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#00FF9D")).
				Background(lipgloss.Color("#161B22")).
				Padding(0, 1).
				Render("✓ "+s.label))
		} else {
			// Future
			parts = append(parts, lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8B949E")).
				Background(lipgloss.Color("#161B22")).
				Padding(0, 1).
				Render("  "+s.label))
		}
	}

	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#30363D")).Render(" ❯ ")
	return strings.Join(parts, sep)
}

// View renders the current phase.
func (m RequestHintModel) View() string {
	w := m.width
	if w < 40 {
		w = 40
	}

	wizard := m.renderWizardBar()

	var body string
	switch m.phase {
	case hintSelectRound:
		body = m.list.View()

	case hintSelectType:
		var b strings.Builder
		plainLeft := m.svc.Rounds.CountHints(m.selected.ID, "plain")
		encLeft := m.svc.Rounds.CountHints(m.selected.ID, "encoded")
		usedP, _ := m.svc.DB.CountHintsUsed(m.team.ID, m.selected.ID, "plain")
		usedE, _ := m.svc.DB.CountHintsUsed(m.team.ID, m.selected.ID, "encoded")

		targetCard := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Background(lipgloss.Color("#161B22")).
			Padding(0, 1).
			Render(fmt.Sprintf("Target: Round %d — %s (%d pts)  •  Inventory: Plain %d/%d  Encoded %d/%d",
				m.selected.ID, m.selected.Name, m.selected.Points,
				max0(plainLeft-usedP), plainLeft, max0(encLeft-usedE), encLeft))

		b.WriteString(targetCard + "\n\n")
		b.WriteString(m.typ.View() + "\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).
			Render("[↑/↓] Choose Type  •  [Enter] Confirm Selection  •  [Esc] Back to Rounds"))
		body = b.String()

	case hintConfirm:
		header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
			Background(lipgloss.Color("#00B4D8")).Padding(0, 1).Render(" 🔒 CONFIRM INTEL REQUEST ")

		targetInfo := fmt.Sprintf("\nTarget   : Round %d — %s\nHint Type: %s\nCost     : −%s PTS\n",
			m.selected.ID, m.selected.Name, strings.ToUpper(m.typeSel), formatPoints(m.cost))

		targetStyled := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F0F6FC")).
			Render(targetInfo)

		prompt := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00B4D8")).
			Render("Enter your team password to authorize hint dispensation:")

		inputCard := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00B4D8")).
			Background(lipgloss.Color("#0D1117")).
			Padding(1, 2).
			Render(m.passInp.View())

		btnHelp := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FF9D")).
			Render("[Enter] Verify & Dispense Intel  ") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).Render("•  [Esc] Back to Type Selection")

		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Background(lipgloss.Color("#161B22")).
			Padding(1, 3).
			Width(w - 6).
			Render(lipgloss.JoinVertical(lipgloss.Left,
				header,
				targetStyled,
				prompt,
				"",
				inputCard,
				"",
				btnHelp,
			))

		body = box

	default: // hintDone
		var b strings.Builder
		unlockBanner := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#0B0F19")).
			Background(lipgloss.Color("#00FF9D")).
			Padding(0, 2).
			Render(fmt.Sprintf("🔓 TACTICAL INTEL UNLOCKED (%s, −%s PTS)", strings.ToUpper(m.typeSel), formatPoints(m.cost)))

		intelBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00FF9D")).
			Background(lipgloss.Color("#0D1117")).
			Foreground(lipgloss.Color("#F0F6FC")).
			Padding(1, 2).
			Width(w - 8).
			Render(m.result)

		cta := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00F0FF")).
			Render("Press [Enter] to return to Command Deck")

		b.WriteString(lipgloss.JoinVertical(lipgloss.Left, unlockBanner, "\n", intelBox, "\n", cta))
		body = b.String()
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, wizard, "\n", body))
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
