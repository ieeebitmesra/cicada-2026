// Package views contains the individual screens of the CTF portal TUI.
package views

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ieee-ctf/internal/tui/msg"
)

// WelcomeBanner is the stylized ASCII art for the landing view.
const WelcomeBanner = `  ██████╗██╗ ██████╗ █████╗ ██████╗  █████╗     ██████╗  ██████╗ ██████╗ ██████╗ 
 ██╔════╝██║██╔════╝██╔══██╗██╔══██╗██╔══██╗    ╚════██╗██╔═████╗╚════██╗██╔════╝ 
 ██║     ██║██║     ███████║██║  ██║███████║     █████╔╝██║██╔██║ █████╔╝███████╗ 
 ██║     ██║██║     ██╔══██║██║  ██║██╔══██║    ██╔═══╝ ████╔╝██║██╔═══╝ ██╔═══██╗
 ╚██████╗██║╚██████╗██║  ██║██████╔╝██║  ██║    ███████╗╚██████╔╝███████╗╚██████╔╝
  ╚═════╝╚═╝ ╚═════╝╚═╝  ╚═╝╚═════╝ ╚═╝  ╚═╝    ╚══════╝ ╚═════╝ ╚══════╝ ╚═════╝ `

// WelcomeModel is the landing screen.
type WelcomeModel struct {
	width, height int
}

// NewWelcome creates the welcome screen.
func NewWelcome() WelcomeModel { return WelcomeModel{} }

// SetSize implements sizing.
func (m *WelcomeModel) SetSize(w, h int) { m.width, m.height = w, h }

// Update handles keys.
func (m WelcomeModel) Update(m_ tea.Msg) (WelcomeModel, tea.Cmd) {
	if key, ok := m_.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter", " ":
			return m, func() tea.Msg { return msg.Navigate{Target: msg.TDashboard} }
		}
	}
	return m, nil
}

// View renders cyber banner + briefing cards + launch CTA.
func (m WelcomeModel) View() string {
	w := m.width
	if w < 40 {
		w = 40
	}

	bannerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00F0FF"))

	var banner string
	if w >= 84 {
		banner = bannerStyle.Render(WelcomeBanner)
	} else {
		banner = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F0F6FC")).
			Background(lipgloss.Color("#00629B")).
			Padding(0, 2).
			Render(" ⬢ CICADA 2026 // IEEE CYBER-RANGE ")
	}

	subHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00F0FF")).
		Render(Truncate("⚡ IEEE CYBER RANGE // SECURE ATTACK & DEFENSE ARENA ⚡", w-4))

	// Card 1: Mission Briefing
	c1Title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#00F0FF")).
		Padding(0, 1).
		Render(" PROTOCOL & BRIEFING ")

	var c1Body strings.Builder
	c1Body.WriteString("\n• Challenges operate in isolated sandboxed targets.\n")
	c1Body.WriteString("• Solve rounds in any order at your team's discretion.\n")
	c1Body.WriteString("• Valid flags conform strictly to: ")
	c1Body.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFB800")).Render("IEEE{...}") + "\n")
	c1Body.WriteString("• Submissions are strictly rate-limited against brute-force.\n")

	// Card 2: Scoring Economics
	c2Title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#00FF9D")).
		Padding(0, 1).
		Render(" SCORING & PENALTIES ")

	var c2Body strings.Builder
	c2Body.WriteString("\n• Solved Round : ")
	c2Body.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF9D")).Render("+100% Points bounty\n"))
	c2Body.WriteString("• Plain Hint   : ")
	c2Body.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFB800")).Render("−20% of round point value\n"))
	c2Body.WriteString("• Encoded Hint : ")
	c2Body.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00F0FF")).Render("−10% of round point value (decode it)\n"))
	c2Body.WriteString("• Skip Round   : ")
	c2Body.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF3860")).Render("−50% penalty (permanent forfeiture)\n"))
	c2Body.WriteString("• Integrity    : Hint requests require local PGP clearsign verification.\n")

	// Determine layout based on terminal width (Rule #4)
	var cardsView string
	availW := w - 8
	cardWidth := (availW - 2) / 2
	if cardWidth >= 38 {
		card1 := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Background(lipgloss.Color("#161B22")).
			Padding(0, 1).
			Width(cardWidth).
			Render(c1Title + c1Body.String())

		card2 := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Background(lipgloss.Color("#161B22")).
			Padding(0, 1).
			Width(cardWidth).
			Render(c2Title + c2Body.String())

		cardsView = lipgloss.JoinHorizontal(lipgloss.Top, card1, "  ", card2)
	} else {
		fullCardWidth := availW
		card1 := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Background(lipgloss.Color("#161B22")).
			Padding(0, 1).
			Width(fullCardWidth).
			Render(c1Title + c1Body.String())

		card2 := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Background(lipgloss.Color("#161B22")).
			Padding(0, 1).
			Width(fullCardWidth).
			Render(c2Title + c2Body.String())

		cardsView = lipgloss.JoinVertical(lipgloss.Left, card1, card2)
	}

	ctaPill := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#00F0FF")).
		Padding(0, 2).
		Render("▸▸▸ PRESS [ENTER] OR [SPACE] TO INITIALIZE COMMAND DECK ◂◂◂")

	content := lipgloss.JoinVertical(lipgloss.Left,
		banner,
		subHeader,
		"",
		cardsView,
		"",
		"  "+ctaPill,
	)

	return lipgloss.NewStyle().
		Width(m.width).
		Padding(1, 2).
		Render(content)
}

