package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
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
	regFocusPubkey
)

// RegisterModel handles first-login registration (new password + PGP key).
type RegisterModel struct {
	width, height int

	passInput    textinput.Model
	confirmInput textinput.Model
	pubkey       textarea.Model
	focus        int

	err  string
	db   *store.DB
	team *models.Team
}

// NewRegister builds the registration form.
func NewRegister(db *store.DB, team *models.Team) RegisterModel {
	pass := textinput.New()
	pass.Placeholder = "New password (min 8 chars)"
	pass.EchoMode = textinput.EchoPassword
	pass.CharLimit = 128
	pass.Width = 40

	confirm := textinput.New()
	confirm.Placeholder = "Confirm password"
	confirm.EchoMode = textinput.EchoPassword
	confirm.CharLimit = 128
	confirm.Width = 40

	pk := textarea.New()
	pk.Placeholder = "Paste your ARMORED PGP PUBLIC KEY here\n(-----BEGIN PGP PUBLIC KEY BLOCK----- ...)"
	pk.CharLimit = 16384
	pk.SetWidth(70)
	pk.SetHeight(8)
	pk.ShowLineNumbers = false

	m := RegisterModel{passInput: pass, confirmInput: confirm, pubkey: pk, db: db, team: team}
	m.refocus()
	return m
}

func (m *RegisterModel) refocus() {
	m.passInput.Blur()
	m.confirmInput.Blur()
	m.pubkey.Blur()
	switch m.focus {
	case regFocusPass:
		m.passInput.Focus()
	case regFocusConfirm:
		m.confirmInput.Focus()
	case regFocusPubkey:
		m.pubkey.Focus()
	}
}

// SetSize implements sizing.
func (m *RegisterModel) SetSize(w, h int) {
	m.width, m.height = w, h
	if w > 8 {
		m.pubkey.SetWidth(w - 10)
	}
}

// Update handles input + submission.
func (m RegisterModel) Update(msg_ tea.Msg) (RegisterModel, tea.Cmd) {
	if key, ok := msg_.(tea.KeyMsg); ok {
		switch key.String() {
		case "tab", "down":
			m.focus = (m.focus + 1) % 3
			m.refocus()
			return m, nil
		case "shift+tab", "up":
			m.focus = (m.focus + 2) % 3
			m.refocus()
			return m, nil
		case "ctrl+s":
			return m, m.submit()
		}
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch m.focus {
	case regFocusPass:
		m.passInput, cmd = m.passInput.Update(msg_)
		cmds = append(cmds, cmd)
	case regFocusConfirm:
		m.confirmInput, cmd = m.confirmInput.Update(msg_)
		cmds = append(cmds, cmd)
	default:
		m.pubkey, cmd = m.pubkey.Update(msg_)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *RegisterModel) submit() tea.Cmd {
	pass := m.passInput.Value()
	confirm := m.confirmInput.Value()

	if err := validateRegistration(pass, confirm, m.pubkey.Value()); err != nil {
		m.err = err.Error()
		return nil
	}
	m.err = ""

	newKey := strings.TrimSpace(m.pubkey.Value())
	if err := auth.CompleteRegistration(m.db, m.team, pass, newKey); err != nil {
		m.err = err.Error()
		return nil
	}
	// Mutate shared team state so the rest of the TUI sees registration.
	m.team.PGPPubkey = newKey
	m.team.Registered = true

	flash := func() tea.Msg { return msg.StatusFlash{Text: "Registration complete. Welcome!", Success: true} }
	nav := func() tea.Msg { return msg.Navigate{Target: msg.TDashboard} }
	return tea.Sequence(flash, nav)
}

func validateRegistration(pass, confirm, pubkey string) error {
	if len(pass) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	if pass != confirm {
		return fmt.Errorf("passwords do not match")
	}
	if !strings.Contains(pubkey, "-----BEGIN PGP PUBLIC KEY BLOCK-----") {
		return fmt.Errorf("paste an armored PGP PUBLIC key")
	}
	return nil
}

// View renders the form.
func (m RegisterModel) View() string {
	var b strings.Builder
	b.WriteString("First-time setup — secure your team account.\n\n")

	focusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#3FA7DC"))
	b.WriteString(focusStyle.Render(func() string {
		if m.focus == regFocusPass {
			return "▸ Password:"
		}
		return "  Password:"
	}()) + "\n  " + m.passInput.View() + "\n\n")

	b.WriteString(focusStyle.Render(func() string {
		if m.focus == regFocusConfirm {
			return "▸ Confirm :"
		}
		return "  Confirm :"
	}()) + "\n  " + m.confirmInput.View() + "\n\n")

	b.WriteString(focusStyle.Render(func() string {
		if m.focus == regFocusPubkey {
			return "▸ PGP public key:"
		}
		return "  PGP public key:"
	}()) + "\n" + m.pubkey.View() + "\n")

	if m.err != "" {
		b.WriteString("\n" + lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5252")).Render("✗ "+m.err) + "\n")
	}

	b.WriteString("\n[Tab] next field  [Ctrl+S] register")
	return lipgloss.NewStyle().Width(m.width).Padding(1, 2).Render(b.String())
}
