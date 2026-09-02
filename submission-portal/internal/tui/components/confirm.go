package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ieee-ctf/internal/tui/msg"
)

// Confirm is a reusable Yes/No dialog.
type Confirm struct {
	ID       int
	Question string
	Details  []string
	active   bool
	selected int // 0 = No, 1 = Yes
	width    int
}

// NewConfirm creates an inactive confirm dialog.
func NewConfirm(id int, question string, details []string) Confirm {
	return Confirm{ID: id, Question: question, Details: details, selected: 0}
}

// Activate arms the dialog with a default answer.
func (c *Confirm) Activate(defaultYes bool) {
	c.active = true
	if defaultYes {
		c.selected = 1
	} else {
		c.selected = 0
	}
}

// Deactivate hides the dialog.
func (c *Confirm) Deactivate() { c.active = false }

// Active reports whether the dialog is showing.
func (c *Confirm) Active() bool { return c.active }

// SetSize sets the render width.
func (c *Confirm) SetSize(w, _ int) { c.width = w }

// Update handles keys while active. Returns (consumed, cmd).
func (c *Confirm) Update(m tea.Msg) (bool, tea.Cmd) {
	key, ok := m.(tea.KeyMsg)
	if !ok || !c.active {
		return false, nil
	}
	switch key.String() {
	case "left", "h", "shift+tab":
		c.selected = 0
		return true, nil
	case "right", "l", "tab":
		c.selected = 1
		return true, nil
	case "y", "Y":
		c.active = false
		return true, c.emit(true, false)
	case "n", "N", "esc":
		c.active = false
		return true, c.emit(false, true)
	case "enter":
		yes := c.selected == 1
		c.active = false
		return true, c.emit(yes, false)
	}
	return false, nil
}

func (c *Confirm) emit(yes, cancelled bool) tea.Cmd {
	return func() tea.Msg {
		return msg.ConfirmResult{ID: c.ID, Yes: yes, Cancelled: cancelled}
	}
}

// View renders the modal dialog box.
func (c *Confirm) View() string {
	if !c.active {
		return ""
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#FFB800")).
		Padding(0, 1).
		Render("⚠ CONFIRMATION REQUIRED")

	question := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F0F6FC")).
		Render(c.Question)

	var detailLines string
	if len(c.Details) > 0 {
		var dParts []string
		for _, d := range c.Details {
			dParts = append(dParts, "  • "+d)
		}
		detailLines = "\n" + strings.Join(dParts, "\n") + "\n"
	}

	btnBase := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#8B949E")).
		Background(lipgloss.Color("#161B22")).
		Padding(0, 2)

	btnSelectedNo := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F0F6FC")).
		Background(lipgloss.Color("#FF3860")).
		Padding(0, 2).
		Render("[ ✖ No, Cancel ]")

	btnSelectedYes := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#00FF9D")).
		Padding(0, 2).
		Render("[ ✔ Yes, Proceed ]")

	noBtn := btnBase.Render("  ✖ No, Cancel  ")
	yesBtn := btnBase.Render("  ✔ Yes, Proceed  ")

	if c.selected == 0 {
		noBtn = btnSelectedNo
	} else {
		yesBtn = btnSelectedYes
	}

	btnRow := lipgloss.JoinHorizontal(lipgloss.Center, noBtn, "    ", yesBtn)

	navHelp := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8B949E")).
		Render("[←/→] Toggle Selection  •  [Enter] Confirm  •  [Esc] Cancel")

	body := header + "\n\n" + question + "\n" + detailLines + "\n" + btnRow + "\n\n" + navHelp

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FFB800")).
		Background(lipgloss.Color("#161B22")).
		Padding(1, 3).
		Render(body)
}

