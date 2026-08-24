// Package views contains the individual screens of the CTF portal TUI.
package views

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ieee-ctf/internal/tui/components"
	"ieee-ctf/internal/tui/msg"
)

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

// View renders banner + instructions.
func (m WelcomeModel) View() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#3FA7DC")).
		Render(components.Banner))
	b.WriteString("\n")
	b.WriteString("Welcome to the IEEE CTF Submission Portal.\n\n")
	b.WriteString("Rules of engagement:\n")
	b.WriteString("  • Solve rounds in any order — flags look like ")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB300")).Render("IEEE{...}") + "\n")
	b.WriteString("  • Clearing a round awards its points\n")
	b.WriteString("  • Plain hint: −20% of the round • Encoded hint: −10%\n")
	b.WriteString("  • Skipping a round: −50% of your CURRENT total\n")
	b.WriteString("  • Hint requests must be PGP clearsigned — no accidental clicks\n\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("[Enter] Continue"))
	return lipgloss.NewStyle().
		Width(m.width).
		Padding(1, 2).
		Render(b.String())
}
