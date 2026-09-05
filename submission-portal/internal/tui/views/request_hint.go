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

type hintRoundItem struct {
	id            int
	title         string
	desc          string
	description   string
	points        int
	unlockedCount int
	plainLeft     int
	plainTotal    int
	encLeft       int
	encTotal      int
}

func (i hintRoundItem) Title() string       { return i.title }
func (i hintRoundItem) Description() string { return i.desc }
func (i hintRoundItem) FilterValue() string { return i.title }

type hintRoundDelegate struct {
	width int
}

func (d hintRoundDelegate) Height() int                             { return 2 }
func (d hintRoundDelegate) Spacing() int                            { return 1 }
func (d hintRoundDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d hintRoundDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(hintRoundItem)
	if !ok {
		return
	}

	selected := index == m.Index()
	maxWidth := d.width - 4
	if maxWidth < 20 {
		maxWidth = 20
	}

	var badge string
	if it.unlockedCount > 0 {
		badge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
			Background(lipgloss.Color("#00FF9D")).Padding(0, 1).Render(fmt.Sprintf("🔓 %d INTEL", it.unlockedCount))
	} else {
		badge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
			Background(lipgloss.Color("#00F0FF")).Padding(0, 1).Render("🔒 0 INTEL")
	}

	pointsBadge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFB800")).
		Background(lipgloss.Color("#161B22")).Padding(0, 1).Render(fmt.Sprintf("+%d PTS", it.points))

	var prefix, titleStyled, descStyled string
	availTitleW := maxWidth - 22
	if availTitleW < 10 {
		availTitleW = 10
	}

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

	row1 := prefix + badge + " " + titleStyled + "  " + pointsBadge
	row2 := descStyled

	fmt.Fprintf(w, "%s\n%s", row1, row2)
}

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

	badgeColor := "#00F0FF"
	if it.typ == "plain" {
		badgeColor = "#FFB800"
	}
	badge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color(badgeColor)).Padding(0, 1).Render(strings.ToUpper(it.typ))

	costBadge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F0F6FC")).
		Background(lipgloss.Color("#161B22")).Padding(0, 1).Render(it.cost)

	var prefix, descStyled string
	if selected {
		prefix = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00F0FF")).Render("▸ ")
		descStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("#00F0FF")).Render("    " + it.desc)
	} else {
		prefix = "  "
		descStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).Render("    " + it.desc)
	}

	fmt.Fprintf(w, "%s%s  %s\n%s", prefix, badge, costBadge, descStyled)
}

// RequestHintModel walks the hint request and persistent intel viewing flow.
type RequestHintModel struct {
	width, height int
	phase         hintPhase

	list    list.Model
	typ     list.Model
	passInp textinput.Model

	svc           *scoring.Service
	team          *models.Team
	selected      models.Round
	unlockedHints []scoring.UnlockedHint
	typeSel       string
	cost          float64

	result string
}

// NewRequestHint builds the hint flow view.
func NewRequestHint(svc *scoring.Service, team *models.Team) RequestHintModel {
	l := list.New([]list.Item{}, hintRoundDelegate{width: 54}, 0, 0)
	l.Title = "SELECT TARGET CHALLENGE FOR INTEL"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#00F0FF")).
		Padding(0, 1)

	t := list.New([]list.Item{}, hintTypeDelegate{width: 54}, 0, 0)
	t.Title = "SELECT INTEL CLEARANCE TYPE TO REQUEST"
	t.SetShowStatusBar(false)
	t.SetFilteringEnabled(false)
	t.SetShowHelp(false)
	t.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#FFB800")).
		Padding(0, 1)

	passInp := textinput.New()
	passInp.Placeholder = "Enter your team password to authorize"
	passInp.EchoMode = textinput.EchoPassword
	passInp.CharLimit = 128
	passInp.Width = 44
	passInp.Prompt = "❯ "
	passInp.PromptStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00F0FF"))
	passInp.TextStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F0F6FC"))
	passInp.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E"))

	return RequestHintModel{list: l, typ: t, passInp: passInp, svc: svc, team: team}
}

// SetSize implements sizing.
func (m *RequestHintModel) SetSize(w, h int) {
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

	m.list.SetDelegate(hintRoundDelegate{width: listW})
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
		if !m.svc.Rounds.HintsAllowed(r.ID) {
			continue
		}

		plainTotal := m.svc.Rounds.CountHints(r.ID, "plain")
		encTotal := m.svc.Rounds.CountHints(r.ID, "encoded")
		usedP, _ := m.svc.DB.CountHintsUsed(m.team.ID, r.ID, "plain")
		usedE, _ := m.svc.DB.CountHintsUsed(m.team.ID, r.ID, "encoded")
		unlockedCount := usedP + usedE

		var desc string
		if r.Description != "" {
			if unlockedCount > 0 {
				desc = fmt.Sprintf("[%d Intel] %s", unlockedCount, r.Description)
			} else {
				desc = r.Description
			}
		} else if unlockedCount > 0 {
			desc = fmt.Sprintf("Unlocked: %d intel • Plain %d/%d • Encoded %d/%d left",
				unlockedCount, max0(plainTotal-usedP), plainTotal, max0(encTotal-usedE), encTotal)
		} else {
			desc = fmt.Sprintf("Available: Plain %d/%d • Encoded %d/%d",
				max0(plainTotal-usedP), plainTotal, max0(encTotal-usedE), encTotal)
		}

		items = append(items, hintRoundItem{
			id:            r.ID,
			title:         fmt.Sprintf("Round %d — %s", r.ID, r.Name),
			desc:          desc,
			description:   r.Description,
			points:        r.Points,
			unlockedCount: unlockedCount,
			plainLeft:     max0(plainTotal - usedP),
			plainTotal:    plainTotal,
			encLeft:       max0(encTotal - usedE),
			encTotal:      encTotal,
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
			case hintDone:
				m.phase = hintSelectType
				return m, nil
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
			if it, ok := m.list.SelectedItem().(hintRoundItem); ok {
				for _, r := range m.loadRounds() {
					if r.ID == it.id {
						m.selected = r
						break
					}
				}
				// Load unlocked hints for this round
				m.unlockedHints, _ = m.svc.RoundUnlockedHints(m.team.ID, m.selected.ID)

				// Populate available hint types
				plainLeft := m.svc.Rounds.CountHints(m.selected.ID, "plain")
				encLeft := m.svc.Rounds.CountHints(m.selected.ID, "encoded")
				usedP, _ := m.svc.DB.CountHintsUsed(m.team.ID, m.selected.ID, "plain")
				usedE, _ := m.svc.DB.CountHintsUsed(m.team.ID, m.selected.ID, "encoded")

				var typeItems []list.Item
				if usedP < plainLeft {
					costP := scoring.HintCost(m.selected.Points, "plain")
					typeItems = append(typeItems, hintTypeItem{
						typ:  "plain",
						cost: fmt.Sprintf("Cost: −%s PTS (20%%)", formatPoints(costP)),
						desc: "Direct plain text clue • Immediate tactical advantage",
					})
				}
				if usedE < encLeft {
					costE := scoring.HintCost(m.selected.Points, "encoded")
					typeItems = append(typeItems, hintTypeItem{
						typ:  "encoded",
						cost: fmt.Sprintf("Cost: −%s PTS (10%%)", formatPoints(costE)),
						desc: "Encrypted / algorithmic puzzle clue • Lower penalty",
					})
				}

				m.typ.SetItems(typeItems)
				m.phase = hintSelectType
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg_)
		return m, cmd

	case hintSelectType:
		if key, ok := msg_.(tea.KeyMsg); ok && key.String() == "enter" {
			if len(m.typ.Items()) > 0 {
				if it, ok := m.typ.SelectedItem().(hintTypeItem); ok {
					m.typeSel = it.typ
					m.cost = scoring.HintCost(m.selected.Points, m.typeSel)
					m.phase = hintConfirm
					m.passInp.SetValue("")
					m.passInp.Focus()
					return m, textinput.Blink
				}
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
			m.unlockedHints, _ = m.svc.RoundUnlockedHints(m.team.ID, m.selected.ID)
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
		if key, ok := msg_.(tea.KeyMsg); ok && (key.String() == "enter" || key.String() == " ") {
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
		{hintSelectType, "2. UNLOCKED INTEL & REQUEST"},
		{hintConfirm, "3. PASSWORD CONFIRMATION"},
		{hintDone, "4. NEW INTEL DISPENSED"},
	}

	var parts []string
	for _, s := range steps {
		if s.id == m.phase {
			parts = append(parts, lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#0B0F19")).
				Background(lipgloss.Color("#00F0FF")).
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

// View renders the current phase.
func (m RequestHintModel) View() string {
	w := m.width
	if w < 40 {
		w = 40
	}

	wizard := m.renderWizardBar()

	var body string
	if !m.svc.Rounds.HintsAllowed(0) {
		body = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF3860")).
			Background(lipgloss.Color("#161B22")).
			Padding(2, 3).
			Width(w - 6).
			Render(lipgloss.JoinVertical(lipgloss.Left,
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
					Background(lipgloss.Color("#FF3860")).Padding(0, 1).Render(" ⛔ HINTS RESTRICTED "),
				"\n"+lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F0F6FC")).
					Render("Tactical intel and hints have been disabled by competition organizers."),
				lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).
					Render("All challenges must be solved independently without point-deduction clues."),
				"\n"+lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00F0FF")).
					Render("Press [Esc] to return to Dashboard."),
			))
		return lipgloss.NewStyle().
			Width(m.width).
			Padding(1, 2).
			Render(lipgloss.JoinVertical(lipgloss.Left, wizard, "\n", body))
	}

	switch m.phase {
	case hintSelectRound:
		if len(m.list.Items()) == 0 {
			body = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#30363D")).
				Background(lipgloss.Color("#161B22")).
				Padding(1, 2).
				Render(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF9D")).
					Render("✔ No remaining active challenges with hints available."))
		} else {
			if w >= 90 {
				hudW := w - m.list.Width() - 8
				if hudW < 30 {
					hudW = 30
				}
				hud := m.renderRoundHUD(hudW)
				body = lipgloss.JoinHorizontal(lipgloss.Top, m.list.View(), "  ", hud)
			} else {
				body = m.list.View()
			}
		}

	case hintSelectType:
		var b strings.Builder
		availW := w - 8

		// 1. Unlocked Intel Archive Section
		unlockedCard := m.renderUnlockedHintsBox(availW)
		b.WriteString(unlockedCard + "\n\n")

		// 2. Request Next Hint Section
		if len(m.typ.Items()) > 0 {
			b.WriteString(m.typ.View() + "\n\n")
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).
				Render("[↑/↓] Choose Hint Type  •  [Enter] Request Intel  •  [Esc] Back to Rounds"))
		} else {
			allDone := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#00FF9D")).
				Render("✔ All available tactical intel has been unlocked for this challenge.")
			escHelp := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8B949E")).
				Render("Press [Esc] to return to Round Selection.")
			b.WriteString(allDone + "\n" + escHelp)
		}
		body = b.String()

	case hintConfirm:
		header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
			Background(lipgloss.Color("#00F0FF")).Padding(0, 1).Render(" 🔒 AUTHORIZE INTEL CLEARANCE ")

		targetInfo := fmt.Sprintf("\nTarget   : Round %d — %s\n", m.selected.ID, m.selected.Name)
		if m.selected.Description != "" {
			targetInfo += fmt.Sprintf("Details  : %s\n", m.selected.Description)
		}
		targetInfo += fmt.Sprintf("Hint Type: %s\nCost     : −%s PTS\n", strings.ToUpper(m.typeSel), formatPoints(m.cost))

		targetStyled := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F0F6FC")).
			Render(targetInfo)

		prompt := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00F0FF")).
			Render("Enter your team password to authorize hint points deduction:")

		inputCard := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00F0FF")).
			Background(lipgloss.Color("#0D1117")).
			Padding(1, 2).
			Render(m.passInp.View())

		btnHelp := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FF9D")).
			Render("[Enter] Verify Password & Unlock Intel  ") +
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

		archiveNote := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF9D")).
			Render("💡 This intel is permanently archived. You can re-view it anytime in Request Hint or My Status.")

		cta := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00F0FF")).
			Render("Press [Enter] to return to Command Deck • [Esc] Back to Round Selection")

		b.WriteString(lipgloss.JoinVertical(lipgloss.Left, unlockBanner, "\n", intelBox, "\n", archiveNote, "\n", cta))
		body = b.String()
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, wizard, "\n", body))
}

func (m RequestHintModel) renderUnlockedHintsBox(width int) string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#00FF9D")).
		Padding(0, 1).
		Render(fmt.Sprintf(" UNLOCKED INTEL ARCHIVE: Round %d — %s ", m.selected.ID, m.selected.Name))

	var b strings.Builder
	b.WriteString(title + "\n")
	if m.selected.Description != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8")).Render("  ↳ "+m.selected.Description) + "\n\n")
	} else {
		b.WriteString("\n")
	}

	if len(m.unlockedHints) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).
			Render("No intel unlocked yet for this challenge. Select a clearance type below to request a clue."))
	} else {
		cardW := width - 6
		if cardW < 24 {
			cardW = 24
		}

		for i, h := range m.unlockedHints {
			badgeColor := "#00F0FF"
			if h.HintType == "plain" {
				badgeColor = "#FFB800"
			}
			badge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
				Background(lipgloss.Color(badgeColor)).Padding(0, 1).
				Render(fmt.Sprintf("%s INTEL #%d", strings.ToUpper(h.HintType), h.HintIndex+1))

			costTag := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF3860")).
				Render(fmt.Sprintf("−%s PTS", formatPoints(h.CostPoints)))

			clueText := lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#30363D")).
				Background(lipgloss.Color("#0D1117")).
				Foreground(lipgloss.Color("#00FF9D")).
				Padding(0, 1).
				Width(cardW).
				Render(h.Text)

			b.WriteString(fmt.Sprintf("%s  %s\n%s\n", badge, costTag, clueText))
			if i < len(m.unlockedHints)-1 {
				b.WriteString("\n")
			}
		}
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#30363D")).
		Background(lipgloss.Color("#161B22")).
		Padding(1, 2).
		Width(width).
		Render(b.String())
}

func (m RequestHintModel) renderRoundHUD(width int) string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#00F0FF")).
		Padding(0, 1).
		Render(" TACTICAL INTEL DOSSIER ")

	maxTextW := width - 4
	if maxTextW < 10 {
		maxTextW = 10
	}

	var detail strings.Builder
	selected := m.list.SelectedItem()
	if it, ok := selected.(hintRoundItem); ok {
		detail.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F0F6FC")).
			Render("\nSelected Target: "+Truncate(it.title, maxTextW-16)) + "\n")
		if it.description != "" {
			detail.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8")).
				Render(Truncate(it.description, maxTextW)) + "\n")
		}
		detail.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB800")).
			Render(fmt.Sprintf("\nChallenge Value: +%d PTS\n\n", it.points)))

		// Fetch unlocked hints for highlighted round
		unlocked, _ := m.svc.RoundUnlockedHints(m.team.ID, it.id)
		if len(unlocked) > 0 {
			detail.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF9D")).
				Render(fmt.Sprintf("🔓 PREVIOUSLY UNLOCKED INTEL (%d):", len(unlocked))) + "\n")
			for _, uh := range unlocked {
				badge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
					Background(lipgloss.Color("#00FF9D")).Padding(0, 1).
					Render(fmt.Sprintf("%s #%d", strings.ToUpper(uh.HintType), uh.HintIndex+1))
				cluePreview := lipgloss.NewStyle().Foreground(lipgloss.Color("#F0F6FC")).
					Render(Truncate(uh.Text, maxTextW-14))
				detail.WriteString(fmt.Sprintf(" • %s %s\n", badge, cluePreview))
			}
			detail.WriteString("\n")
		} else {
			detail.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).
				Render("No intel unlocked yet for this target.\n\n"))
		}

		detail.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#58A6FF")).
			Render("INTEL INVENTORY:") + "\n")
		detail.WriteString(Truncate(fmt.Sprintf("• Plain Hints   : %d of %d remaining (−20%% cost)", it.plainLeft, it.plainTotal), maxTextW) + "\n")
		detail.WriteString(Truncate(fmt.Sprintf("• Encoded Hints : %d of %d remaining (−10%% cost)", it.encLeft, it.encTotal), maxTextW) + "\n\n")
	}

	detail.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).
		Render("Press [Enter] to browse intel or request next clue"))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#30363D")).
		Background(lipgloss.Color("#161B22")).
		Padding(0, 1).
		Width(width).
		Render(title + detail.String())
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

