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

// View renders the flash message.
func (n *Notification) View(width int) string {
	if !n.visible {
		return ""
	}
	style := lipgloss.NewStyle().Bold(true).Padding(0, 1)
	if n.success {
		style = style.Foreground(lipgloss.Color("#003300")).Background(lipgloss.Color("#00C853"))
	} else {
		style = style.Foreground(lipgloss.Color("#330000")).Background(lipgloss.Color("#FF5252"))
	}
	text := n.text
	if w := lipgloss.Width(text); w < width {
		text += strings.Repeat(" ", width-w)
	}
	return style.Render(text)
}
