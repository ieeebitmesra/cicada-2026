package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ieee-ctf/internal/auth"
	"ieee-ctf/internal/models"
	"ieee-ctf/internal/store"
	"ieee-ctf/internal/tui/msg"
)

const (
	regFocusPass = iota
	regFocusConfirm
)

// RegisterModel handles first-login registration (new password).
type RegisterModel struct {
	width, height int

	passInput    textinput.Model
	confirmInput textinput.Model
	focus        int

	err  string
	db   *store.DB
	team *models.Team
}

// NewRegister builds the registration form.
func NewRegister(db *store.DB, team *models.Team) RegisterModel {
	pass := textinput.New()
	pass.Placeholder = "Enter new password (min 8 characters)"
	pass.EchoMode = textinput.EchoPassword
	pass.CharLimit = 128
	pass.Width = 44
	pass.Prompt = "❯ "
	pass.PromptStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00B4D8"))
	pass.TextStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8FAFC"))

	confirm := textinput.New()
	confirm.Placeholder = "Confirm your password"
	confirm.EchoMode = textinput.EchoPassword
	confirm.CharLimit = 128
	confirm.Width = 44
	confirm.Prompt = "❯ "
	confirm.PromptStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00B4D8"))
	confirm.TextStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8FAFC"))

	m := RegisterModel{passInput: pass, confirmInput: confirm, db: db, team: team}
	m.refocus()
	return m
}

func (m *RegisterModel) refocus() {
	m.passInput.Blur()
	m.confirmInput.Blur()
	switch m.focus {
	case regFocusPass:
		m.passInput.Focus()
	case regFocusConfirm:
		m.confirmInput.Focus()
	}
}

// SetSize implements sizing.
func (m *RegisterModel) SetSize(w, h int) {
	m.width, m.height = w, h
}

// Update handles input + submission.
func (m RegisterModel) Update(msg_ tea.Msg) (RegisterModel, tea.Cmd) {
	if key, ok := msg_.(tea.KeyMsg); ok {
		switch key.String() {
		case "tab", "down":
			m.focus = (m.focus + 1) % 2
			m.refocus()
			return m, nil
		case "shift+tab", "up":
			m.focus = (m.focus + 1) % 2
			m.refocus()
			return m, nil
		case "ctrl+s", "enter":
			return m, m.submit()
		}
	}

	var cmd tea.Cmd
	switch m.focus {
	case regFocusPass:
		m.passInput, cmd = m.passInput.Update(msg_)
	case regFocusConfirm:
		m.confirmInput, cmd = m.confirmInput.Update(msg_)
	}
	return m, cmd
}

func (m *RegisterModel) submit() tea.Cmd {
	pass := m.passInput.Value()
	confirm := m.confirmInput.Value()

	if err := validateRegistration(pass, confirm); err != nil {
		m.err = err.Error()
		return nil
	}
	m.err = ""

	if err := auth.CompleteRegistration(m.db, m.team, pass); err != nil {
		m.err = err.Error()
		return nil
	}
	// Mutate shared team state so the rest of the TUI sees registration.
	m.team.Registered = true

	flash := func() tea.Msg { return msg.StatusFlash{Text: "Account registered successfully. Welcome!", Success: true} }
	nav := func() tea.Msg { return msg.Navigate{Target: msg.TDashboard} }
	return tea.Sequence(flash, nav)
}

func validateRegistration(pass, confirm string) error {
	if len(pass) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	if pass != confirm {
		return fmt.Errorf("passwords do not match")
	}
	return nil
}

// View renders the form.
func (m RegisterModel) View() string {
	w := m.width
	if w < 40 {
		w = 40
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#0B0F19")).
		Background(lipgloss.Color("#00F0FF")).
		Padding(0, 1).
		Render(" INITIAL ACCOUNT PROVISIONING ")

	subtitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#C9D1D9")).
		Render("\nFirst-time login detected. Secure your team account with a new password.\n")

	// Form inputs styling
	labelActive := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00F0FF"))
	labelDim := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8B949E"))

	// Field 1: Password
	lblPass := labelDim.Render("  Password (min 8 characters):")
	if m.focus == regFocusPass {
		lblPass = labelActive.Render("▸ Password (min 8 characters):")
	}
	field1 := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(func() lipgloss.Color {
			if m.focus == regFocusPass {
				return lipgloss.Color("#00F0FF")
			}
			return lipgloss.Color("#30363D")
		}()).
		Background(lipgloss.Color("#0D1117")).
		Padding(0, 1).
		Render(m.passInput.View())

	// Field 2: Confirm
	lblConfirm := labelDim.Render("  Confirm Password:")
	if m.focus == regFocusConfirm {
		lblConfirm = labelActive.Render("▸ Confirm Password:")
	}
	field2 := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(func() lipgloss.Color {
			if m.focus == regFocusConfirm {
				return lipgloss.Color("#00F0FF")
			}
			return lipgloss.Color("#30363D")
		}()).
		Background(lipgloss.Color("#0D1117")).
		Padding(0, 1).
		Render(m.confirmInput.View())

	var errBox string
	if m.err != "" {
		errBox = "\n" + lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF3860")).
			Background(lipgloss.Color("#450A1A")).
			Bold(true).
			Foreground(lipgloss.Color("#F0F6FC")).
			Padding(0, 1).
			Render("✖ "+m.err) + "\n"
	}

	helpBar := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8B949E")).
		Render("\n[Tab] Next Field  •  [Shift+Tab] Previous Field  •  [Enter] Save Credentials")

	formBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#30363D")).
		Background(lipgloss.Color("#161B22")).
		Padding(1, 2).
		Width(w - 6).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			lblPass,
			field1,
			"",
			lblConfirm,
			field2,
			errBox,
			helpBar,
		))

	return lipgloss.NewStyle().
		Width(m.width).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, header, subtitle, formBox))
}

