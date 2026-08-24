// Package tui contains the root Bubble Tea model, styles and components.
package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// IEEE brand palette.
var (
	ColorPrimary   = lipgloss.Color("#00629B") // IEEE blue
	ColorPrimaryLt = lipgloss.Color("#3FA7DC")
	ColorSuccess   = lipgloss.Color("#00C853")
	ColorError     = lipgloss.Color("#FF5252")
	ColorWarn      = lipgloss.Color("#FFB300")
	ColorMuted     = lipgloss.Color("#6B7280")
	ColorText      = lipgloss.Color("#E5E7EB")
)

var (
	// TitleStyle renders view titles.
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorPrimary).
			Padding(0, 1)

	// BoxStyle is a rounded box around content.
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	// ErrorStyle renders error text.
	ErrorStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorError)

	// SuccessStyle renders success text.
	SuccessStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess)

	// WarnStyle renders warning text.
	WarnStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorWarn)

	// MutedStyle renders dimmed helper text.
	MutedStyle = lipgloss.NewStyle().Foreground(ColorMuted)

	// HelpKeyStyle renders keybind names.
	HelpKeyStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorPrimaryLt)

	// BannerStyle styles the ASCII banner.
	BannerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimaryLt)
)

// FormatPoints renders a signed point value.
func FormatPoints(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%d", int64(v))
	}
	return fmt.Sprintf("%.1f", v)
}
