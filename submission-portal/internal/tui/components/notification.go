package components

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ieee-ctf/internal/tui/msg"
)

// Notification is a transient flash message (success/error).
type Notification struct {
	visible bool
	text    string
	success bool
}

// NewNotification creates an empty notification.
func NewNotification() Notification { return Notification{} }

// Set shows a message and returns a command that clears it after a delay.
func (n *Notification) Set(text string, success bool) tea.Cmd {
	n.visible = true
	n.text = text
	n.success = success
	return tea.Tick(4*time.Second, func(time.Time) tea.Msg {
		return msg.ClearNotification{}
	})
}

// Handle processes the expiry message. Returns true if it consumed the msg.
func (n *Notification) Handle(m tea.Msg) bool {
	if _, ok := m.(msg.ClearNotification); ok {
		n.visible = false
		return true
	}
	return false
}

// Visible reports whether a message is showing.
func (n *Notification) Visible() bool { return n.visible }

// View renders the flash message as a tactical HUD toast.
func (n *Notification) View(width int) string {
	if !n.visible {
		return ""
	}
	if width < 30 {
		width = 30
	}

	var badge, borderCol, bgCol string
	if n.success {
		badge = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#0B0F19")).
			Background(lipgloss.Color("#00FF9D")).
			Padding(0, 1).
			Render("✔ SUCCESS")
		borderCol = "#00FF9D"
		bgCol = "#064E3B"
	} else {
		badge = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#0B0F19")).
			Background(lipgloss.Color("#FF3860")).
			Padding(0, 1).
			Render("✖ ALERT")
		borderCol = "#FF3860"
		bgCol = "#450A1A"
	}

	content := badge + "  " + lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F0F6FC")).
		Render(n.text)

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderCol)).
		Background(lipgloss.Color(bgCol)).
		Padding(0, 2).
		Render(content)

	// Center or pad nicely
	cardW := lipgloss.Width(card)
	if cardW < width {
		pad := (width - cardW) / 2
		if pad < 1 {
			pad = 1
		}
		return strings.Repeat(" ", pad) + card
	}
	return card
}

