package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"ieee-ctf/internal/models"
	"ieee-ctf/internal/tui/msg"
)

func TestHeader(t *testing.T) {
	// 1. Nil team
	h1 := Header(nil, "", "PORTAL ACCESS", 80)
	if !strings.Contains(h1, "PANTHEON CTF") || !strings.Contains(h1, "PORTAL ACCESS") {
		t.Errorf("Header missing expected components: %s", h1)
	}

	// 2. Team with score
	team := &models.Team{ID: 1, Name: "TeamAlpha"}
	h2 := Header(team, "1,250", "COMMAND DECK", 100)
	if !strings.Contains(h2, "TeamAlpha") || !strings.Contains(h2, "1,250 PTS") || !strings.Contains(h2, "COMMAND DECK") {
		t.Errorf("Header with team/score incomplete: %s", h2)
	}

	// 3. Narrow terminal width
	h3 := Header(team, "500", "FLAG INGESTION", 35)
	if len(h3) == 0 {
		t.Errorf("Header on narrow width empty")
	}
}

func TestFooter(t *testing.T) {
	bindings := []string{"[Enter] Select", "[Esc] Back", "[Ctrl+C] Quit"}
	f := Footer(bindings, 100)
	if !strings.Contains(f, "Enter") || !strings.Contains(f, "Select") || !strings.Contains(f, "CICADA 2026") {
		t.Errorf("Footer missing keys or system tag: %s", f)
	}

	fNarrow := Footer(bindings, 40)
	if len(fNarrow) == 0 {
		t.Errorf("Footer narrow empty")
	}
}

func TestNotification(t *testing.T) {
	n := NewNotification()
	if n.Visible() {
		t.Errorf("expected initial notification to be invisible")
	}

	cmd := n.Set("Flag accepted!", true)
	if !n.Visible() {
		t.Errorf("expected notification to be visible after Set")
	}
	if cmd == nil {
		t.Errorf("expected timer cmd from Set")
	}

	v := n.View(80)
	if !strings.Contains(v, "SUCCESS") || !strings.Contains(v, "Flag accepted!") {
		t.Errorf("notification view missing success toast: %s", v)
	}

	// Test error toast
	n.Set("Invalid PGP signature", false)
	vErr := n.View(80)
	if !strings.Contains(vErr, "ALERT") || !strings.Contains(vErr, "Invalid PGP signature") {
		t.Errorf("notification view missing alert toast: %s", vErr)
	}

	// Test clear
	handled := n.Handle(msg.ClearNotification{})
	if !handled || n.Visible() {
		t.Errorf("expected notification to clear on ClearNotification msg")
	}
}

func TestConfirm(t *testing.T) {
	c := NewConfirm(42, "Skip this round?", []string{"Penalty: -250 pts"})
	if c.Active() {
		t.Errorf("expected confirm initially inactive")
	}

	c.Activate(false)
	if !c.Active() {
		t.Errorf("expected confirm to be active after Activate")
	}

	view := c.View()
	if !strings.Contains(view, "CONFIRMATION REQUIRED") || !strings.Contains(view, "Penalty: -250 pts") {
		t.Errorf("Confirm View missing header or details: %s", view)
	}

	// Toggle right
	consumed, _ := c.Update(tea.KeyMsg{Type: tea.KeyRight})
	if !consumed {
		t.Errorf("expected key to be consumed")
	}

	// Press Enter to confirm Yes
	consumed, cmd := c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !consumed || cmd == nil {
		t.Errorf("expected enter to emit result")
	}
	resMsg := cmd()
	res, ok := resMsg.(msg.ConfirmResult)
	if !ok || !res.Yes || res.ID != 42 {
		t.Errorf("unexpected confirm result: %+v", resMsg)
	}
	if c.Active() {
		t.Errorf("expected confirm to deactivate after enter")
	}
}
