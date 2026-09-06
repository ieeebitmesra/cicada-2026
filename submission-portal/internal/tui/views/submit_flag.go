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
	id          int
	title       string
	desc        string
	description string
	points      int
	solved      bool
	skip        bool
	limitSolves bool
	maxSolves   int
	solveCount  int
	quotaFull   bool
	open        bool
}

func (i roundItem) Title() string       { return i.title }
func (i roundItem) Description() string { return i.desc }
func (i roundItem) FilterValue() string { return i.title }

type flagRoundDelegate struct {
	width int
}

func (d flagRoundDelegate) Height() int                             { return 2 }
func (d flagRoundDelegate) Spacing() int                            { return 1 }
func (d flagRoundDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d flagRoundDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(roundItem)
	if !ok {
		return
	}

	selected := index == m.Index()
	maxWidth := d.width - 4
	if maxWidth < 20 {
		maxWidth = 20
	}

	var badge string
	switch {
	case it.solved:
		badge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
			Background(lipgloss.Color("#10B981")).Padding(0, 1).Render("✓ SOLVED")
	case it.skip:
		badge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
			Background(lipgloss.Color("#EF4444")).Padding(0, 1).Render("✗ SKIPPED")
	case it.quotaFull:
		badge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
			Background(lipgloss.Color("#EF4444")).Padding(0, 1).Render("🔒 QUOTA FULL")
	case it.limitSolves && it.maxSolves > 0:
		left := it.maxSolves - it.solveCount
		if left < 0 {
			left = 0
		}
		badge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
			Background(lipgloss.Color("#F59E0B")).Padding(0, 1).Render(fmt.Sprintf("⚡ %d/%d LEFT", left, it.maxSolves))
	default:
		badge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
			Background(lipgloss.Color("#00B4D8")).Padding(0, 1).Render("⚡ OPEN")
	}

	pointsBadge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FBBF24")).
		Background(lipgloss.Color("#1E293B")).Padding(0, 1).Render(fmt.Sprintf("+%d PTS", it.points))

	var prefix, titleStyled, descStyled string
	availTitleW := maxWidth - 20
	if availTitleW < 10 {
		availTitleW = 10
	}

	if selected {
		prefix = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00B4D8")).Render("▸ ")
		titleStyled = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8FAFC")).
			Background(lipgloss.Color("#1E293B")).Padding(0, 1).Render(Truncate(it.title, availTitleW))
		descStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("#00B4D8")).Render("    " + Truncate(it.desc, maxWidth-6))
	} else {
		prefix = "  "
		titleStyled = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E2E8F0")).Render(Truncate(it.title, availTitleW))
		descStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B")).Render("    " + Truncate(it.desc, maxWidth-6))
	}

	row1 := prefix + badge + " " + titleStyled + "  " + pointsBadge
	row2 := descStyled

	fmt.Fprintf(w, "%s\n%s", row1, row2)
}

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
	l := list.New([]list.Item{}, flagRoundDelegate{width: 54}, 0, 0)
	l.Title = "SELECT TARGET CHALLENGE"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(true)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#00B4D8")).
		Padding(0, 1)
	l.Styles.PaginationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00B4D8")).Padding(0, 1)

	input := textinput.New()
	input.Placeholder = "PANTHEON{...}"
	input.CharLimit = 256
	input.Prompt = "❯ ENTER FLAG: "
	input.PromptStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00B4D8"))
	input.TextStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8FAFC"))
	input.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#475569"))

	return SubmitFlagModel{list: l, input: input, svc: svc, team: team}
}

// SetSize implements sizing.
func (m *SubmitFlagModel) SetSize(w, h int) {
	m.width, m.height = w, h

	// Apply Rule #4 (Weights, Not Pixels):
	availableW := w - 6
	listW := availableW
	if w >= 90 {
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

	m.list.SetDelegate(flagRoundDelegate{width: listW})
	m.list.SetSize(listW, listH)

	inputWidth := 50
	if w > 16 {
		inputWidth = w - 16
		if inputWidth > 64 {
			inputWidth = 64
		}
	}
	m.input.Width = inputWidth
}

// Refresh reloads round availability.
func (m *SubmitFlagModel) Refresh() tea.Cmd {
	rounds, err := m.svc.DB.ListRounds(true)
	if err != nil {
		return nil
	}
	solvesMap, _ := m.svc.DB.SolvesCountMap()
	var items []list.Item
	for _, r := range rounds {
		solved, _ := m.svc.DB.HasSolvedRound(m.team.ID, r.ID)
		skipped, _ := m.svc.DB.HasSkipped(m.team.ID, r.ID)

		solveCount := solvesMap[r.ID]
		maxSolves := r.MaxSolves
		if maxSolves == 0 && m.svc.Rounds.IsSolveLimited(r.ID) {
			maxSolves = m.svc.Rounds.MaxAllowedSolves(r.ID)
		}
		isLimited := r.LimitSolves || m.svc.Rounds.IsSolveLimited(r.ID)
		quotaFull := isLimited && maxSolves > 0 && solveCount >= maxSolves

		desc := "Available for flag submission"
		if r.Description != "" {
			desc = r.Description
		}
		if solved {
			if r.Description != "" {
				desc = "✓ Solved • " + r.Description
			} else {
				desc = "Solved by your team — bounty secured"
			}
		} else if skipped {
			if r.Description != "" {
				desc = "✗ Skipped • " + r.Description
			} else {
				desc = "Bypassed via strategic skip — locked"
			}
		} else if quotaFull {
			if r.Description != "" {
				desc = fmt.Sprintf("🔒 Quota full (%d/%d solves claimed) • %s", solveCount, maxSolves, r.Description)
			} else {
				desc = fmt.Sprintf("Solve quota reached (%d/%d) — no points available", solveCount, maxSolves)
			}
		} else if isLimited && maxSolves > 0 {
			left := maxSolves - solveCount
			if r.Description != "" {
				desc = fmt.Sprintf("⚡ Limited (%d/%d solves left) • %s", left, maxSolves, r.Description)
			} else {
				desc = fmt.Sprintf("⚡ Limited solve window: %d of %d spots remaining", left, maxSolves)
			}
		}

		items = append(items, roundItem{
			id:          r.ID,
			title:       fmt.Sprintf("Round %d — %s", r.ID, r.Name),
			desc:        desc,
			description: r.Description,
			points:      r.Points,
			solved:      solved,
			skip:        skipped,
			limitSolves: isLimited,
			maxSolves:   maxSolves,
			solveCount:  solveCount,
			quotaFull:   quotaFull,
			open:        !solved && !skipped && !quotaFull,
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
						flashText := "That round is closed (already solved or skipped)."
						if it.quotaFull {
							flashText = "That round is closed: solve quota has been filled."
						}
						flash := func() tea.Msg {
							return msg.StatusFlash{Text: flashText, Success: false}
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
			text = "Rate limited: too many attempts. Please wait."
		case errors.Is(err, store.ErrAlreadySolved):
			text = "Round is already solved!"
		case errors.Is(err, scoring.ErrAlreadySkipped):
			text = "Round was skipped."
		case errors.Is(err, scoring.ErrRoundInactive):
			text = "Round is inactive."
		case errors.Is(err, scoring.ErrMaxSolvesReached):
			text = "Solve quota reached: only the first 3 teams to solve receive points."
		case errors.Is(err, scoring.ErrFlagFormat):
			text = "Format invalid. Flags must match PANTHEON{...}"
		case errors.Is(err, scoring.ErrFlagTooLong):
			text = "Flag token exceeds maximum length."
		case strings.Contains(err.Error(), "cooldown"):
			text = err.Error()
		}
		flashErr := func() tea.Msg { return msg.StatusFlash{Text: text, Success: false} }
		return flashErr
	}
	if !res.Correct {
		flashWrong := func() tea.Msg { return msg.StatusFlash{Text: "Verification Failed: Incorrect Flag.", Success: false} }
		return flashWrong
	}
	flashOK := func() tea.Msg {
		return msg.StatusFlash{Text: fmt.Sprintf("FLAG ACCEPTED! +%s points awarded!", formatPoints(res.PointsAwarded)), Success: true}
	}
	nav := func() tea.Msg { return msg.Navigate{Target: msg.TDashboard} }
	return tea.Sequence(flashOK, nav)
}

// View renders the phase-appropriate UI.
func (m SubmitFlagModel) View() string {
	w := m.width
	if w < 40 {
		w = 40
	}

	var content string
	if m.phase == submitSelectRound {
		if w >= 95 {
			hudW := w - m.list.Width() - 8
			if hudW < 30 {
				hudW = 30
			}
			hud := m.renderRoundBriefingHUD(hudW)
			content = lipgloss.JoinHorizontal(lipgloss.Top, m.list.View(), "  ", hud)
		} else {
			content = m.list.View()
		}
	} else {
		content = m.renderFlagInputTerminal()
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Padding(1, 2).
		Render(content)
}

func (m SubmitFlagModel) renderRoundBriefingHUD(width int) string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#00F0FF")).
		Padding(0, 1).
		Render(" CHALLENGE DOSSIER ")

	maxTextW := width - 4
	if maxTextW < 10 {
		maxTextW = 10
	}

	var detail strings.Builder
	selected := m.list.SelectedItem()
	if it, ok := selected.(roundItem); ok {
		detail.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F0F6FC")).
			Render("\nTarget: "+Truncate(it.title, maxTextW-10)) + "\n")
		if it.description != "" {
			detail.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8")).
				Render(Truncate(it.description, maxTextW)) + "\n")
		}
		detail.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFD700")).
			Render(fmt.Sprintf("\nBounty: +%d Points\n", it.points)))
		detail.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00F0FF")).
			Render("Status: "+Truncate(it.desc, maxTextW)) + "\n\n")
	}

	detail.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#58A6FF")).
		Render("SUBMISSION ADVISORY:") + "\n")
	detail.WriteString(Truncate("• Verify token string carefully", maxTextW) + "\n")
	detail.WriteString(Truncate("• Submissions undergo cryptographic check", maxTextW) + "\n")
	detail.WriteString(Truncate("• Excessive invalid attempts trigger cooldown", maxTextW) + "\n\n")

	detail.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).
		Render("Press [Enter] to open flag entry terminal"))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#30363D")).
		Background(lipgloss.Color("#161B22")).
		Padding(0, 1).
		Width(width).
		Render(title + detail.String())
}

func (m SubmitFlagModel) renderFlagInputTerminal() string {
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#00F0FF")).
		Padding(0, 1).
		Render(" FLAG INGESTION TERMINAL ")

	targetInfo := fmt.Sprintf("\nTarget   : Round %d — %s\n", m.selected.ID, m.selected.Name)
	if m.selected.Description != "" {
		targetInfo += fmt.Sprintf("Details  : %s\n", m.selected.Description)
	}
	if m.selected.LimitSolves && m.selected.MaxSolves > 0 {
		targetInfo += fmt.Sprintf("Quota    : ⚡ Limited to first %d solves only\n", m.selected.MaxSolves)
	}
	targetInfo += fmt.Sprintf("Bounty   : +%d PTS\nStatus   : ACTIVE CHALLENGE\n", m.selected.Points)

	targetStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F0F6FC")).
		Render(targetInfo)

	inputCard := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00F0FF")).
		Background(lipgloss.Color("#0D1117")).
		Padding(1, 2).
		Render(m.input.View())

	advisory := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8B949E")).
		Render("Token format: PANTHEON{...} • Case-sensitive • Rate-limited")

	btnHelp := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FF9D")).
		Render("[Enter] Validate & Submit Flag  ") +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).Render("•  [Esc] Cancel & Return")

	var intelSection string
	unlocked, _ := m.svc.RoundUnlockedHints(m.team.ID, m.selected.ID)
	if len(unlocked) > 0 {
		var ib strings.Builder
		ib.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFB800")).
			Render("💡 UNLOCKED INTEL CLUES:") + "\n")
		for _, uh := range unlocked {
			badge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F19")).
				Background(lipgloss.Color("#00FF9D")).Padding(0, 1).
				Render(fmt.Sprintf("%s #%d", strings.ToUpper(uh.HintType), uh.HintIndex+1))
			clue := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF9D")).Render(uh.Text)
			ib.WriteString(fmt.Sprintf("• %s %s\n", badge, clue))
		}
		intelCardW := m.width - 16
		if intelCardW < 24 {
			intelCardW = 24
		}
		intelSection = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Background(lipgloss.Color("#0D1117")).
			Padding(0, 1).
			Width(intelCardW).
			Render(strings.TrimSpace(ib.String())) + "\n"
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#30363D")).
		Background(lipgloss.Color("#161B22")).
		Padding(1, 3).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			header,
			targetStyled,
			intelSection,
			inputCard,
			"",
			advisory,
			"",
			btnHelp,
		))

	return box
}

