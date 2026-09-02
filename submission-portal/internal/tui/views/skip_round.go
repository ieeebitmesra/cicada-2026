package views

import (
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
	"ieee-ctf/internal/tui/components"
	"ieee-ctf/internal/tui/msg"
)

type skipPhase int

const (
	skipPhaseSelect skipPhase = iota
	skipPhaseConfirm
	skipPhaseDone
)

type skipItem struct {
	id      int
	title   string
	penalty float64
	points  int
	desc    string
}

func (i skipItem) Title() string       { return i.title }
func (i skipItem) Description() string { return i.desc }
func (i skipItem) FilterValue() string { return i.title }

type skipDelegate struct {
	width int
}

func (d skipDelegate) Height() int                             { return 2 }
func (d skipDelegate) Spacing() int                            { return 1 }
func (d skipDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d skipDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(skipItem)
	if !ok {
		return
	}

	selected := index == m.Index()
	maxWidth := d.width - 4
	if maxWidth < 20 {
		maxWidth = 20
	}

	badge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#FFB800")).Padding(0, 1).Render("⏩ BYPASS")

	penaltyBadge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF3860")).
		Background(lipgloss.Color("#161B22")).Padding(0, 1).Render(fmt.Sprintf("−%s PTS", formatPoints(it.penalty)))

	availTitleW := maxWidth - 20
	if availTitleW < 10 {
		availTitleW = 10
	}

	var prefix, titleStyled, descStyled string
	if selected {
		prefix = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00F0FF")).Render("▸ ")
		titleStyled = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F0F6FC")).
			Background(lipgloss.Color("#161B22")).Padding(0, 1).Render(Truncate(it.title, availTitleW))
		descStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("#00F0FF")).Render("    " + Truncate(it.desc, maxWidth-6))
	} else {
		prefix = "  "
		titleStyled = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#C9D1D9")).Render(Truncate(it.title, availTitleW))
		descStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).Render("    " + Truncate(it.desc, maxWidth-6))
	}

	row1 := prefix + badge + " " + titleStyled + "  " + penaltyBadge
	row2 := descStyled

	fmt.Fprintf(w, "%s\n%s", row1, row2)
}

// SkipRoundModel executes a round skip with password confirmation.
type SkipRoundModel struct {
	width, height int
	phase         skipPhase

	list     list.Model
	passInp  textinput.Model
	confirm  components.Confirm
	selected models.Round
	cost     float64
	total    float64

	svc  *scoring.Service
	team *models.Team
}

// NewSkipRound builds the skip view with password confirmation.
func NewSkipRound(svc *scoring.Service, team *models.Team) SkipRoundModel {
	l := list.New([]list.Item{}, skipDelegate{width: 54}, 0, 0)
	l.Title = "SELECT CHALLENGE TO STRATEGICALLY BYPASS"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#FFB800")).
		Padding(0, 1)

	passInp := textinput.New()
	passInp.Placeholder = "Enter your team password to confirm"
	passInp.EchoMode = textinput.EchoPassword
	passInp.CharLimit = 128
	passInp.Width = 44
	passInp.Prompt = "❯ "
	passInp.PromptStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF3860"))
	passInp.TextStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8FAFC"))

	return SkipRoundModel{
		phase:   skipPhaseSelect,
		list:    l,
		passInp: passInp,
		confirm: components.NewConfirm(1, "Skip this round?", nil),
		svc:     svc,
		team:    team,
	}
}

// SetSize implements responsive sizing based on Golden Rules.
func (m *SkipRoundModel) SetSize(w, h int) {
	m.width, m.height = w, h

	availableW := w - 6
	listW := availableW
	if w >= 90 {
		listW = (availableW * 5) / 9
	}
	if listW < 30 {
		listW = 30
	}

	listH := h - 8
	if listH < 6 {
		listH = 6
	}

	m.list.SetDelegate(skipDelegate{width: listW})
	m.list.SetSize(listW, listH)
	m.confirm.SetSize(w, h)

	inputWidth := 50
	if w > 16 {
		inputWidth = w - 16
		if inputWidth > 64 {
			inputWidth = 64
		}
	}
	m.passInp.Width = inputWidth
}

// Refresh reloads skippable rounds.
func (m *SkipRoundModel) Refresh() tea.Cmd {
	m.phase = skipPhaseSelect
	m.passInp.SetValue("")

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
		penalty := scoring.SkipCost(r.Points)
		items = append(items, skipItem{
			id:      r.ID,
			title:   fmt.Sprintf("Round %d — %s", r.ID, r.Name),
			penalty: penalty,
			points:  r.Points,
			desc:    fmt.Sprintf("Challenge Bounty: %d pts • Forfeits −%s pts permanently", r.Points, formatPoints(penalty)),
		})
	}
	cmd := m.list.SetItems(items)
	m.confirm.Deactivate()
	return cmd
}

// Update handles user input, phase transitions, and password submission.
func (m SkipRoundModel) Update(msg_ tea.Msg) (SkipRoundModel, tea.Cmd) {
	if key, ok := msg_.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			switch m.phase {
			case skipPhaseConfirm:
				m.phase = skipPhaseSelect
				m.passInp.SetValue("")
				m.passInp.Blur()
				return m, nil
			case skipPhaseDone:
				m.phase = skipPhaseSelect
				return m, func() tea.Msg { return msg.Navigate{Target: msg.TDashboard} }
			default:
				return m, func() tea.Msg { return msg.Navigate{Target: msg.TDashboard} }
			}
		}
	}

	switch m.phase {
	case skipPhaseSelect:
		if key, ok := msg_.(tea.KeyMsg); ok && key.String() == "enter" {
			if it, ok := m.list.SelectedItem().(skipItem); ok {
				for _, r := range m.loadRounds() {
					if r.ID != it.id {
						continue
					}
					m.selected = r
					m.cost = scoring.SkipCost(r.Points)
					b, _ := m.svc.Breakdown(m.team.ID)
					m.total = b.Total
					m.phase = skipPhaseConfirm
					m.passInp.SetValue("")
					m.passInp.Focus()
					return m, textinput.Blink
				}
			}
		}
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg_)
		return m, cmd

	case skipPhaseConfirm:
		if key, ok := msg_.(tea.KeyMsg); ok && (key.String() == "enter" || key.String() == "ctrl+s") {
			password := m.passInp.Value()
			if password == "" {
				flashErr := func() tea.Msg {
					return msg.StatusFlash{Text: "Enter your team password to confirm the bypass.", Success: false}
				}
				return m, flashErr
			}

			// Verify password against stored hash
			if !auth.VerifyTeamLogin(m.svc.DB, m.team.SSHUser, password) {
				flashErr := func() tea.Msg {
					return msg.StatusFlash{Text: "Incorrect password. Bypass denied.", Success: false}
				}
				m.passInp.SetValue("")
				return m, flashErr
			}

			// Password verified — execute the skip directly
			cost, err := m.svc.SkipRound(m.team, m.selected.ID)
			if err != nil {
				text := "Cannot execute bypass: " + err.Error()
				switch err {
				case scoring.ErrRoundInactive:
					text = "Challenge round is no longer active."
				case scoring.ErrAlreadySkipped:
					text = "Round was already skipped."
				case scoring.ErrSolvedNoSkip:
					text = "Round is already solved — cannot skip."
				}
				flashErr := func() tea.Msg { return msg.StatusFlash{Text: text, Success: false} }
				return m, flashErr
			}

			m.cost = cost
			m.phase = skipPhaseDone
			flashOK := func() tea.Msg {
				return msg.StatusFlash{
					Text:    fmt.Sprintf("ROUND %d BYPASS EXECUTED (−%s pts).", m.selected.ID, formatPoints(cost)),
					Success: true,
				}
			}
			return m, flashOK
		}

		var cmd tea.Cmd
		m.passInp, cmd = m.passInp.Update(msg_)
		return m, cmd

	case skipPhaseDone:
		if key, ok := msg_.(tea.KeyMsg); ok && (key.String() == "enter" || key.String() == " ") {
			return m, func() tea.Msg { return msg.Navigate{Target: msg.TDashboard} }
		}
	}

	return m, nil
}

func (m *SkipRoundModel) loadRounds() []models.Round {
	rounds, err := m.svc.DB.ListRounds(true)
	if err != nil {
		return nil
	}
	return rounds
}

func (m SkipRoundModel) renderWizardBar() string {
	steps := []struct {
		id    skipPhase
		label string
	}{
		{skipPhaseSelect, "1. SELECT TARGET"},
		{skipPhaseConfirm, "2. PASSWORD CONFIRMATION"},
		{skipPhaseDone, "3. BYPASS CONFIRMED"},
	}

	var parts []string
	for _, s := range steps {
		if s.id == m.phase {
			parts = append(parts, lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#0B0F19")).
				Background(lipgloss.Color("#FFB800")).
				Padding(0, 1).
				Render("► "+s.label))
		} else if s.id < m.phase {
			parts = append(parts, lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#00FF9D")).
				Background(lipgloss.Color("#161B22")).
				Padding(0, 1).
				Render("✓ "+s.label))
		} else {
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

// View renders the phase UI.
func (m SkipRoundModel) View() string {
	w := m.width
	if w < 40 {
		w = 40
	}

	wizard := m.renderWizardBar()

	var body string
	switch m.phase {
	case skipPhaseSelect:
		if len(m.list.Items()) == 0 {
			body = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#30363D")).
				Background(lipgloss.Color("#161B22")).
				Padding(1, 2).
				Render(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF9D")).
					Render("✔ No remaining skippable challenges. All rounds are solved or already bypassed."))
		} else {
			if w >= 95 {
				hudW := w - m.list.Width() - 8
				if hudW < 30 {
					hudW = 30
				}
				hud := m.renderSkipHUD(hudW)
				body = lipgloss.JoinHorizontal(lipgloss.Top, m.list.View(), "  ", hud)
			} else {
				body = m.list.View()
			}
		}

	case skipPhaseConfirm:
		header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
			Background(lipgloss.Color("#FF3860")).Padding(0, 1).Render(" ⚠️  CONFIRM ROUND BYPASS ")

		targetInfo := fmt.Sprintf("\nTarget   : Round %d — %s\nPenalty  : −%s PTS (50%% of challenge value)\nStatus   : IRREVERSIBLE FORFEITURE\n",
			m.selected.ID, m.selected.Name, formatPoints(m.cost))

		targetStyled := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F0F6FC")).
			Render(targetInfo)

		warning := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF3860")).
			Render("⚠️  This action is PERMANENT. Enter your team password to confirm:")

		inputCard := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF3860")).
			Background(lipgloss.Color("#0D1117")).
			Padding(1, 2).
			Render(m.passInp.View())

		btnHelp := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FF9D")).
			Render("[Enter] Verify & Forfeit Round  ") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).Render("•  [Esc] Cancel & Return")

		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Background(lipgloss.Color("#161B22")).
			Padding(1, 3).
			Width(w - 6).
			Render(lipgloss.JoinVertical(lipgloss.Left,
				header,
				targetStyled,
				warning,
				"",
				inputCard,
				"",
				btnHelp,
			))

		body = box

	case skipPhaseDone:
		var b strings.Builder
		banner := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#0B0F19")).
			Background(lipgloss.Color("#FF3860")).
			Padding(0, 2).
			Render(fmt.Sprintf("⏩ ROUND %d BYPASS EXECUTED (−%s PTS DEDUCTION)", m.selected.ID, formatPoints(m.cost)))

		summary := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF3860")).
			Background(lipgloss.Color("#0D1117")).
			Foreground(lipgloss.Color("#F0F6FC")).
			Padding(1, 2).
			Width(w - 8).
			Render(fmt.Sprintf("Target Challenge: Round %d — %s\nPenalty Incurred: −%s Points (50%% of challenge value)\nStatus: PERMANENTLY FORFEITED & LOCKED\nProof: Password-confirmed authorization.",
				m.selected.ID, m.selected.Name, formatPoints(m.cost)))

		cta := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00F0FF")).
			Render("Press [Enter] to return to Command Deck")

		b.WriteString(lipgloss.JoinVertical(lipgloss.Left, banner, "\n", summary, "\n", cta))
		body = b.String()
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, wizard, "\n", body))
}

func (m SkipRoundModel) renderSkipHUD(width int) string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#FFB800")).
		Padding(0, 1).
		Render(" STRATEGIC BYPASS PROTOCOL ")

	maxTextW := width - 4
	if maxTextW < 10 {
		maxTextW = 10
	}

	var detail strings.Builder
	selected := m.list.SelectedItem()
	if it, ok := selected.(skipItem); ok {
		detail.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F0F6FC")).
			Render("\nSelected Target: "+Truncate(it.title, maxTextW-16)) + "\n")
		detail.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF3860")).
			Render(fmt.Sprintf("Forfeiture Cost : −%s PTS\n\n", formatPoints(it.penalty))))
	}

	detail.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#58A6FF")).
		Render("BYPASS MECHANICS:") + "\n")
	detail.WriteString(Truncate("• Penalty equals 50% of the challenge's bounty", maxTextW) + "\n")
	detail.WriteString(Truncate("• Skipping permanently locks the challenge", maxTextW) + "\n")
	detail.WriteString(Truncate("• Requires password confirmation for authorization", maxTextW) + "\n")
	detail.WriteString(Truncate("• Use only when stuck on critical roadblocks", maxTextW) + "\n\n")

	detail.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).
		Render("Press [Enter] to initiate password confirmation"))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#30363D")).
		Background(lipgloss.Color("#161B22")).
		Padding(0, 1).
		Width(width).
		Render(title + detail.String())
}
