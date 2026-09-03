package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"ieee-ctf/internal/models"
	"ieee-ctf/internal/scoring"
	"ieee-ctf/internal/store"
	"ieee-ctf/internal/tui/msg"
)

func newTestEnv(t *testing.T) (*store.DB, *scoring.Service, *models.Team) {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Migrate("../../migrations"); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	id, err := db.CreateTeam("OmegaTeam", "omega", "$2a$10$abcdefghijklmnopqrstuv0123456789012345678901234567890")
	if err != nil {
		t.Fatalf("create team: %v", err)
	}
	team, err := db.GetTeam(id)
	if err != nil {
		t.Fatalf("get team: %v", err)
	}
	_ = db.CompleteRegistration(team.ID, team.SSHPass, "-----BEGIN PGP PUBLIC KEY BLOCK-----\ntest\n-----END PGP PUBLIC KEY BLOCK-----")
	team.Registered = true
	team.PGPPubkey = "-----BEGIN PGP PUBLIC KEY BLOCK-----\ntest\n-----END PGP PUBLIC KEY BLOCK-----"

	yamlContent := `
rounds:
  - id: 1
    name: "Web Infiltration"
    points: 100
    is_active: true
    flag_hash: "3b006c0ab342be3e7ef08855eec1314902a20caeebeecb700147cb9a2bc2245e"
`
	roundsConfig, err := models.ParseRounds([]byte(yamlContent))
	if err != nil {
		t.Fatalf("parse rounds: %v", err)
	}
	_ = db.UpsertRounds(roundsConfig.Rounds)
	flags := scoring.NewFlagValidator(256, 2, 0)
	svc := scoring.NewService(db, roundsConfig, flags, 5*time.Minute)

	return db, svc, team
}

func TestRootModelNavigationAndLifecycle(t *testing.T) {
	db, svc, team := newTestEnv(t)

	m := NewRootModel(db, svc, team, 110, 32, 10*time.Minute)
	if m.state != viewWelcome {
		t.Errorf("expected initial state viewWelcome, got %d", m.state)
	}

	// Test WindowSizeMsg
	mModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 35})
	m = mModel.(RootModel)
	if m.width != 120 || m.height != 35 {
		t.Errorf("expected dimensions 120x35, got %dx%d", m.width, m.height)
	}

	// Test View rendering
	v := m.View()
	if !strings.Contains(v, "PANTHEON CTF") || !strings.Contains(v, "PORTAL ACCESS") {
		t.Errorf("Expected dashboard UI to contain PANTHEON CTF and PORTAL ACCESS, got:\n%s", v)
	}

	// Navigate to Dashboard
	mModel, cmd := m.Update(msg.Navigate{Target: msg.TDashboard})
	m = mModel.(RootModel)
	if m.state != viewDashboard {
		t.Errorf("expected state viewDashboard, got %d", m.state)
	}
	if cmd != nil {
		_ = cmd()
	}

	vDash := m.View()
	if !strings.Contains(vDash, "COMMAND DECK") {
		t.Errorf("Dashboard View missing breadcrumb: %s", vDash)
	}

	// Flash notification test
	mModel, notifCmd := m.Update(msg.StatusFlash{Text: "Operation successful", Success: true})
	m = mModel.(RootModel)
	if notifCmd == nil {
		t.Errorf("expected timer command on StatusFlash")
	}
	vFlash := m.View()
	if !strings.Contains(vFlash, "SUCCESS") || !strings.Contains(vFlash, "Operation successful") {
		t.Errorf("View missing notification toast: %s", vFlash)
	}

	// Esc from Scoreboard back to Dashboard
	mModel, _ = m.Update(msg.Navigate{Target: msg.TScoreboard})
	m = mModel.(RootModel)
	if m.state != viewScoreboard {
		t.Errorf("expected state viewScoreboard, got %d", m.state)
	}
	mModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mModel.(RootModel)
	if m.state != viewDashboard {
		t.Errorf("expected Esc to return to viewDashboard, got %d", m.state)
	}
}

func TestRootModelUnregistered(t *testing.T) {
	db, svc, team := newTestEnv(t)
	team.Registered = false

	m := NewRootModel(db, svc, team, 100, 30, 10*time.Minute)
	if m.state != viewRegister {
		t.Errorf("expected unregistered team to land in viewRegister, got %d", m.state)
	}
	v := m.View()
	if !strings.Contains(v, "ACCOUNT PROVISIONING") {
		t.Errorf("View missing ACCOUNT PROVISIONING breadcrumb: %s", v)
	}
}
