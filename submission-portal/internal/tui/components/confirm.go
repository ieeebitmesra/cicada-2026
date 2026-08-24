package components

import (
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

// View renders the dialog box.
func (c *Confirm) View() string {
	if !c.active {
		return ""
	}
	base := lipgloss.NewStyle().Padding(0, 2)
	sel := lipgloss.NewStyle().Bold(true).
		Background(lipgloss.Color("#00629B")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 2)

	no := base.Render("No")
	yes := base.Render("Yes")
	if c.selected == 0 {
		no = sel.Render("No")
	} else {
		yes = sel.Render("Yes")
	}

	body := c.Question + "\n\n"
	for _, d := range c.Details {
		body += d + "\n"
	}
	body += "\n      " + no + "    " + yes

	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("#FFB300")).
		Padding(1, 2).
		Render(body)
}
