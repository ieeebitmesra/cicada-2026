package views

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
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
	skipPhaseSign
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

// SkipRoundModel executes a round skip with cryptographic PGP signature verification.
type SkipRoundModel struct {
	width, height int
	phase         skipPhase

	list      list.Model
	paste     textarea.Model
	confirm   components.Confirm
	selected  models.Round
	challenge string
	cost      float64
	total     float64

	svc  *scoring.Service
	team *models.Team
}

// NewSkipRound builds the skip view with PGP signing support.
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

	paste := textarea.New()
	paste.Placeholder = "-----BEGIN PGP SIGNED MESSAGE-----\nHash: SHA256\n...\n-----BEGIN PGP SIGNATURE-----\n...\n-----END PGP SIGNATURE-----"
	paste.CharLimit = 8192
	paste.ShowLineNumbers = false

	return SkipRoundModel{
		phase:   skipPhaseSelect,
		list:    l,
		paste:   paste,
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

	if w > 10 {
		m.paste.SetWidth(w - 12)
	}
	pH := h / 3
	if pH < 5 {
		pH = 5
	}
	m.paste.SetHeight(pH)
}

// Refresh reloads skippable rounds.
func (m *SkipRoundModel) Refresh() tea.Cmd {
	m.phase = skipPhaseSelect
	m.paste.Reset()

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

// Update handles user input, phase transitions, and PGP submission.
func (m SkipRoundModel) Update(msg_ tea.Msg) (SkipRoundModel, tea.Cmd) {
	if key, ok := msg_.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			switch m.phase {
			case skipPhaseSign:
				m.phase = skipPhaseSelect
				m.paste.Reset()
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
					if m.team.PGPPubkey == "" {
						flashErr := func() tea.Msg {
							return msg.StatusFlash{
								Text:    "PGP key unlinked. Register a public key to authorize skips.",
								Success: false,
							}
						}
						return m, flashErr
					}

					challenge, cost, err := m.svc.SkipChallenge(m.team, r.ID)
					if err != nil {
						flashErr := func() tea.Msg {
							return msg.StatusFlash{Text: "Cannot initiate bypass: " + err.Error(), Success: false}
						}
						return m, flashErr
					}

					b, _ := m.svc.Breakdown(m.team.ID)
					m.challenge = challenge
					m.cost = cost
					m.total = b.Total
					m.phase = skipPhaseSign
					m.paste.Reset()
					m.paste.Focus()
					return m, textarea.Blink
				}
			}
		}
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg_)
		return m, cmd

	case skipPhaseSign:
		if key, ok := msg_.(tea.KeyMsg); ok && key.String() == "ctrl+s" {
			signed := strings.TrimSpace(m.paste.Value())
			if signed == "" {
				flashErr := func() tea.Msg {
					return msg.StatusFlash{Text: "Paste your PGP clearsigned message before verifying.", Success: false}
				}
				return m, flashErr
			}

			cost, err := m.svc.ExecuteSkip(m.team, m.selected.ID, signed)
			if err != nil {
				text := "PGP signature verification failed."
				switch err {
				case auth.ErrNoSignature:
					text = "No valid PGP signature block found in text."
				case auth.ErrBadSignature:
					text = "PGP signature is INVALID for registered team key."
				case scoring.ErrBadChallenge:
					text = "Signed challenge content does not match the issued nonce."
				case scoring.ErrChallengeStale:
					text = "Challenge timestamp expired (>15 min). Request a fresh nonce."
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
		m.paste, cmd = m.paste.Update(msg_)
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
		{skipPhaseSign, "2. PGP CLEARSIGN VERIFICATION"},
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

	case skipPhaseSign:
		var b strings.Builder
		keyID := auth.PublicKeyID(m.team.PGPPubkey)
		gpgCmd := "gpg --clearsign"
		if keyID != "" {
			gpgCmd = fmt.Sprintf("gpg --clearsign -u %s", keyID)
		}

		availW := w - 8

		nonceHeader := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
			Background(lipgloss.Color("#FFB800")).Padding(0, 1).Render("STEP 1: FORFEITURE NONCE CHALLENGE")

		inputHeader := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
			Background(lipgloss.Color("#00F0FF")).Padding(0, 1).Render("STEP 2: PASTE PGP CLEARSIGNATURE")

		if w >= 90 {
			paneW := (availW - 2) / 2
			if paneW < 38 {
				paneW = 38
			}
			m.paste.SetWidth(paneW - 4)
			m.paste.SetHeight(10)

			tokenBox := lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#30363D")).
				Background(lipgloss.Color("#0D1117")).
				Foreground(lipgloss.Color("#FF3860")).
				Bold(true).
				Padding(0, 1).
				Width(paneW - 6).
				Render(m.challenge)

			cmdBox := lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#30363D")).
				Background(lipgloss.Color("#0D1117")).
				Foreground(lipgloss.Color("#00F0FF")).
				Bold(true).
				Padding(0, 1).
				Width(paneW - 6).
				Render(fmt.Sprintf("$ %s", gpgCmd))

			eofGuide := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#C9D1D9")).
				Render("• Paste token into stdin, then send EOF:\n  Linux/macOS: [Ctrl+D]\n  Windows:     [Ctrl+Z] then [Enter]")

			fileTip := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8B949E")).
				Render(fmt.Sprintf("• Or save to skip.txt and run:\n  $ %s skip.txt", gpgCmd))

			nonceContent := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FFB800")).
				Background(lipgloss.Color("#161B22")).
				Padding(1, 2).
				Width(paneW).
				Render(lipgloss.JoinVertical(lipgloss.Left,
					nonceHeader,
					"",
					lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF3860")).
						Render(fmt.Sprintf("Target: Round %d — %s (−%s PTS)", m.selected.ID, m.selected.Name, formatPoints(m.cost))),
					"",
					lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#58A6FF")).Render("1. Forfeiture Challenge Nonce:"),
					tokenBox,
					"",
					lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#58A6FF")).Render("2. Run in terminal:"),
					cmdBox,
					"",
					eofGuide,
					"",
					fileTip,
				))

			inputCard := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#00F0FF")).
				Background(lipgloss.Color("#161B22")).
				Padding(1, 2).
				Width(paneW).
				Render(lipgloss.JoinVertical(lipgloss.Left,
					inputHeader,
					"",
					m.paste.View(),
					"",
					lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF3860")).Render("⚠️ Irreversible −50% Forfeiture Deduction"),
					"",
					lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF9D")).Render("[Ctrl+S] Verify Signature & Forfeit Round"),
					lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).Render("[Esc] Cancel & Return"),
				))

			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, nonceContent, "  ", inputCard))
		} else {
			m.paste.SetWidth(availW - 4)
			m.paste.SetHeight(5)

			tokenBox := lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#30363D")).
				Background(lipgloss.Color("#0D1117")).
				Foreground(lipgloss.Color("#FF3860")).
				Bold(true).
				Padding(0, 1).
				Render(m.challenge)

			cmdBox := lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#30363D")).
				Background(lipgloss.Color("#0D1117")).
				Foreground(lipgloss.Color("#00F0FF")).
				Bold(true).
				Padding(0, 1).
				Render(fmt.Sprintf("$ %s", gpgCmd))

			nonceContent := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FFB800")).
				Background(lipgloss.Color("#161B22")).
				Padding(1, 2).
				Width(availW).
				Render(lipgloss.JoinVertical(lipgloss.Left,
					nonceHeader,
					"",
					tokenBox,
					cmdBox,
					lipgloss.NewStyle().Foreground(lipgloss.Color("#C9D1D9")).Render("Paste token, then press Ctrl+D (Unix) or Ctrl+Z+Enter (Windows)"),
				))

			inputCard := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#00F0FF")).
				Background(lipgloss.Color("#161B22")).
				Padding(1, 2).
				Width(availW).
				Render(lipgloss.JoinVertical(lipgloss.Left,
					inputHeader,
					"",
					m.paste.View(),
					"",
					lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF9D")).Render("[Ctrl+S] Verify & Forfeit  •  [Esc] Cancel"),
				))

			b.WriteString(lipgloss.JoinVertical(lipgloss.Left, nonceContent, "\n", inputCard))
		}
		body = b.String()

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
			Render(fmt.Sprintf("Target Challenge: Round %d — %s\nPenalty Incurred: −%s Points (50%% of challenge value)\nStatus: PERMANENTLY FORFEITED & LOCKED\nProof: PGP signature verified and stored for non-repudiation.",
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
	detail.WriteString(Truncate("• Requires PGP Clearsign verification of team identity", maxTextW) + "\n")
	detail.WriteString(Truncate("• Use only when stuck on critical roadblocks", maxTextW) + "\n\n")

	detail.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).
		Render("Press [Enter] to initiate PGP signature verification"))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#30363D")).
		Background(lipgloss.Color("#161B22")).
		Padding(0, 1).
		Width(width).
		Render(title + detail.String())
}


