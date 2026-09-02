package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Footer renders the contextual keybind help bar with styled keycaps.
func Footer(bindings []string, width int) string {
	if width < 20 {
		width = 20
	}

	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#30363D")).
		Render(strings.Repeat("─", width))

	var parts []string
	for _, b := range bindings {
		// Expects formats like "[Enter] continue" or "[↑/↓] navigate"
		b = strings.TrimSpace(b)
		if strings.HasPrefix(b, "[") && strings.Contains(b, "]") {
			idx := strings.Index(b, "]")
			key := b[1:idx]
			desc := strings.TrimSpace(b[idx+1:])

			keycap := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#00F0FF")).
				Background(lipgloss.Color("#161B22")).
				Padding(0, 1).
				Render(key)

			descStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#C9D1D9"))

			parts = append(parts, keycap+" "+descStyle.Render(desc))
		} else {
			parts = append(parts, lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#00F0FF")).
				Render(b))
		}
	}

	left := strings.Join(parts, "  ")
	right := ""
	if width > 85 {
		right = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B949E")).
			Render("CICADA 2026 // v2.4")
	}

	gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	barContent := " " + left + strings.Repeat(" ", gap) + right + " "
	barStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#0D1117")).
		Width(width)

	return lipgloss.JoinVertical(lipgloss.Left, divider, barStyle.Render(barContent))
}

