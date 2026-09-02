package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"ieee-ctf/internal/models"
)

// Banner is the cyber ASCII art logo shown in the header/welcome screen.
const Banner = `
   ██████╗██╗███████╗███████╗███████╗   ██████╗████████╗███████╗
   ██╔══██╗██║██╔════╝██╔════╝██╔════╝  ██╔════╝╚══██╔══╝██╔════╝
   ██████╔╝██║█████╗  █████╗  █████╗    ██║        ██║   █████╗  
   ██╔═══╝ ██║██╔══╝  ██╔══╝  ██╔══╝    ██║        ██║   ██╔══╝  
   ██║     ██║███████╗███████╗███████╗  ╚██████╗   ██║   ██║     
   ╚═╝     ╚═╝╚══════╝╚══════╝╚══════╝   ╚═════╝   ╚═╝   ╚═╝     
               C Y B E R   R A N G E   P O R T A L`

// Header renders the top bar: brand + breadcrumbs + team + live score + divider.
func Header(team *models.Team, score string, viewName string, width int) string {
	if width < 20 {
		width = 20
	}

	// Brand badge
	brand := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F0F6FC")).
		Background(lipgloss.Color("#00629B")).
		Padding(0, 1).
		Render("⬢ IEEE CTF")

	// Breadcrumb / View Name
	var breadcrumb string
	if viewName != "" {
		sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#00F0FF")).Render(" ❯ ")
		vName := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F0F6FC")).Render(viewName)
		breadcrumb = sep + vName
	}

	left := brand + breadcrumb

	// Right info cluster
	var rightParts []string

	// Live status dot (if space allows)
	if width > 90 {
		liveDot := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF9D")).Render("● ")
		liveText := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF9D")).Render("LIVE")
		rightParts = append(rightParts, liveDot+liveText)
	}

	if team != nil {
		teamPill := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00F0FF")).
			Background(lipgloss.Color("#161B22")).
			Padding(0, 1).
			Render("👤 " + team.Name)
		rightParts = append(rightParts, teamPill)

		if score != "" {
			scorePill := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFD700")).
				Background(lipgloss.Color("#161B22")).
				Padding(0, 1).
				Render("🏆 " + score + " PTS")
			rightParts = append(rightParts, scorePill)
		}
	}

	right := strings.Join(rightParts, "  ")

	gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	topBar := " " + left + strings.Repeat(" ", gap) + right + " "

	topBarStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#0D1117")).
		Width(width)

	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#30363D")).
		Render(strings.Repeat("─", width))

	return lipgloss.JoinVertical(lipgloss.Left, topBarStyle.Render(topBar), divider)
}

