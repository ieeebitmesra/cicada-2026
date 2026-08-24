package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"ieee-ctf/internal/models"
)

// Banner is the ASCII art logo shown in the header/welcome screen.
const Banner = `
  ___ _ _  __   __  ___ ___
 |_ _| | | \ \ / / | __| _ \
  | || | |_ \ V /  | _||   /
 |___|_|_(_)_/\_/  |___|_|_\
      C T F   P O R T A L
`

// Header renders the top bar: banner title + team + live score.
func Header(team *models.Team, score string, width int) string {
	title := "IEEE CTF Submission Portal"
	right := ""
	if team != nil {
		right = team.Name
		if score != "" {
			right += " • score: " + score
		}
	}

	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#00629B")).
		Padding(0, 1)

	gap := width - lipgloss.Width(title) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	bar := title + strings.Repeat(" ", gap) + right
	return style.Render(bar)
}
