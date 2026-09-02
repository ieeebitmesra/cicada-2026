// Package tui contains the root Bubble Tea model, styles and components.
package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// IEEE Cyber-Range brand palette with high-contrast accessible values.
var (
	ColorPrimary       = lipgloss.Color("#00629B") // IEEE blue
	ColorPrimaryLight  = lipgloss.Color("#00F0FF") // Electric cyber cyan
	ColorPrimaryDark   = lipgloss.Color("#003859") // Deep navy
	ColorAccent        = lipgloss.Color("#A855F7") // Electric violet
	ColorSuccess       = lipgloss.Color("#00FF9D") // Neon emerald green
	ColorSuccessBright = lipgloss.Color("#10B981") // Crisp emerald
	ColorError         = lipgloss.Color("#FF3860") // Crimson red
	ColorWarn          = lipgloss.Color("#FFB800") // Cyber amber gold
	ColorMuted         = lipgloss.Color("#8B949E") // Medium slate
	ColorText          = lipgloss.Color("#F0F6FC") // High-contrast white
	ColorTextDim       = lipgloss.Color("#C9D1D9") // Light readable slate
	ColorBgDark        = lipgloss.Color("#0D1117") // Deep obsidian
	ColorCardBg        = lipgloss.Color("#161B22") // Midnight card
	ColorBorder        = lipgloss.Color("#30363D") // Slate border
	ColorBorderGlow    = lipgloss.Color("#00F0FF") // Glowing cyan border
	ColorGold          = lipgloss.Color("#FFD700") // 1st Place Gold
	ColorSilver        = lipgloss.Color("#E2E8F0") // 2nd Place Silver
	ColorBronze        = lipgloss.Color("#FF7A00") // 3rd Place Bronze
)

// Typography & Component Styles.
var (
	// TitleStyle renders primary view titles.
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText).
			Background(ColorPrimary).
			Padding(0, 2)

	// SubtitleStyle renders descriptive headers.
	SubtitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimaryLight)

	// BoxStyle is a rounded box around content with refined padding.
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2)

	// ActiveBoxStyle represents focused or active card elements.
	ActiveBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimaryLight).
			Padding(1, 2)

	// ErrorStyle renders error text.
	ErrorStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorError)

	// SuccessStyle renders success text.
	SuccessStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorSuccessBright)

	// WarnStyle renders warning text.
	WarnStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorWarn)

	// MutedStyle renders dimmed helper text.
	MutedStyle = lipgloss.NewStyle().Foreground(ColorMuted)

	// HelpKeyStyle renders keybind names.
	HelpKeyStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorPrimaryLight)

	// KeycapStyle renders keyboard button pills.
	KeycapStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimaryLight).
			Background(lipgloss.Color("#1E293B")).
			Padding(0, 1)

	// BannerStyle styles ASCII headers.
	BannerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimaryLight)
)

// FormatPoints renders a signed point value cleanly.
func FormatPoints(v float64) string {
	if math.Abs(v-float64(int64(v))) < 0.001 {
		return fmt.Sprintf("%d", int64(v))
	}
	return fmt.Sprintf("%.1f", v)
}

// RenderBadge creates a pill-style badge with foreground and background colors.
func RenderBadge(label string, fg, bg lipgloss.Color) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(fg).
		Background(bg).
		Padding(0, 1).
		Render(label)
}

// RenderProgressBar renders an ASCII/ANSI gradient progress bar with neon highlights.
func RenderProgressBar(current, total int, width int) string {
	if width < 5 {
		width = 10
	}
	ratio := 0.0
	if total > 0 {
		ratio = float64(current) / float64(total)
		if ratio > 1.0 {
			ratio = 1.0
		}
		if ratio < 0.0 {
			ratio = 0.0
		}
	}
	filled := int(math.Round(ratio * float64(width)))
	empty := width - filled
	if empty < 0 {
		empty = 0
	}

	fillStr := strings.Repeat("█", filled)
	emptyStr := strings.Repeat("░", empty)

	fillStyled := lipgloss.NewStyle().Foreground(ColorSuccess).Render(fillStr)
	emptyStyled := lipgloss.NewStyle().Foreground(ColorBorder).Render(emptyStr)
	percentStr := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimaryLight).
		Render(fmt.Sprintf(" %3.0f%%", ratio*100))

	return "[" + fillStyled + emptyStyled + "]" + percentStr
}

// RenderStatCard produces a modern KPI stat card with top accent ribbon.
func RenderStatCard(title, value, subtext string, width int, accent lipgloss.Color) string {
	if width < 18 {
		width = 18
	}

	tStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorMuted).
		Width(width - 4)

	vStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(accent).
		Width(width - 4)

	sStyle := lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Width(width - 4)

	content := lipgloss.JoinVertical(lipgloss.Left,
		tStyle.Render(strings.ToUpper(title)),
		vStyle.Render(value),
		sStyle.Render(subtext),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Background(ColorCardBg).
		Padding(0, 1).
		Width(width).
		Render(content)
}

// RenderCyberCard wraps content in a styled midnight box with a title ribbon.
func RenderCyberCard(title string, headerBg lipgloss.Color, content string, width int, active bool) string {
	borderCol := ColorBorder
	if active {
		borderCol = ColorPrimaryLight
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorBgDark).
		Background(headerBg).
		Padding(0, 1).
		Render(" " + title + " ")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderCol).
		Background(ColorCardBg).
		Padding(1, 2).
		Width(width).
		Render(header + "\n\n" + content)
}

// StripANSI removes ANSI escape codes from a string for accurate measuring.
func StripANSI(s string) string {
	var b strings.Builder
	inAnsi := false
	for _, ch := range s {
		if ch == '\033' {
			inAnsi = true
			continue
		}
		if inAnsi {
			if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
				inAnsi = false
			}
			continue
		}
		b.WriteRune(ch)
	}
	return b.String()
}

// VisualWidth measures the true terminal display width, stripping ANSI and emoji variation selectors.
func VisualWidth(s string) int {
	stripped := StripANSI(s)
	// Strip variation selectors to work around width discrepancies across terminals
	stripped = strings.ReplaceAll(stripped, "\uFE0F", "")
	stripped = strings.ReplaceAll(stripped, "\uFE0E", "")
	return lipgloss.Width(stripped)
}

// Truncate ensures text never auto-wraps inside bordered panels (Golden Rule #2).
func Truncate(s string, maxLen int) string {
	if maxLen < 1 {
		return ""
	}
	w := VisualWidth(s)
	if w <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return strings.Repeat(".", maxLen)
	}

	runes := []rune(s)
	for i := len(runes); i > 0; i-- {
		candidate := string(runes[:i]) + "…"
		if VisualWidth(candidate) <= maxLen {
			return candidate
		}
	}
	return "…"
}

// PadToVisualWidth pads a string to target visual width.
func PadToVisualWidth(s string, targetWidth int) string {
	w := VisualWidth(s)
	if w >= targetWidth {
		return s
	}
	return s + strings.Repeat(" ", targetWidth-w)
}


