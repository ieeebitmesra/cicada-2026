package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Footer renders the contextual keybind help bar.
func Footer(bindings []string, width int) string {
	var parts []string
	for _, b := range bindings {
		parts = append(parts, lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#3FA7DC")).
			Render(b))
	}
	line := strings.Join(parts, "  •  ")
	if w := lipgloss.Width(line); w < width {
		line += strings.Repeat(" ", width-w)
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Padding(0, 1).
		Render(line)
}
