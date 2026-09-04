package views

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

func newTestDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Migrate("../../../migrations"); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}
	return db
}

func mustCreateTeam(t *testing.T, db *store.DB, name string) *models.Team {
	t.Helper()
	id, err := db.CreateTeam(name, name+"-user", "$2a$10$abcdefghijklmnopqrstuv0123456789012345678901234567890")
	if err != nil {
		t.Fatalf("failed to create team: %v", err)
	}
	team, err := db.GetTeam(id)
	if err != nil {
		t.Fatalf("failed to get team: %v", err)
	}
	return team
}

func newTestService(t *testing.T, db *store.DB) *scoring.Service {
	t.Helper()
	yamlContent := `
rounds:
  - id: 1
    name: "Web Infiltration"
    points: 100
    is_active: true
    flag_hash: "3b006c0ab342be3e7ef08855eec1314902a20caeebeecb700147cb9a2bc2245e"
    hints:
      - type: plain
        text: "Check cookies"
      - type: encoded
        text: "SW5qZWN0IFNRTA=="
  - id: 2
    name: "Buffer Overflow"
    points: 200
    is_active: true
    flag_hash: "4b006c0ab342be3e7ef08855eec1314902a20caeebeecb700147cb9a2bc2245e"
`
	roundsConfig, err := models.ParseRounds([]byte(yamlContent))
	if err != nil {
		t.Fatalf("failed to parse rounds yaml: %v", err)
	}
	if err := db.UpsertRounds(roundsConfig.Rounds); err != nil {
		t.Fatalf("failed to upsert rounds: %v", err)
	}
	flags := scoring.NewFlagValidator(256, 2, 0)
	return scoring.NewService(db, roundsConfig, flags, 5*time.Minute)
}

func TestWelcomeModel(t *testing.T) {
	m := NewWelcome()
	m.SetSize(110, 30)

	v := m.View()
	if !strings.Contains(v, "PROTOCOL & BRIEFING") || !strings.Contains(v, "SCORING & PENALTIES") {
		t.Errorf("Welcome View missing briefing sections: %s", v)
	}

	// Test enter navigation
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected command on Enter")
	}
	res := cmd()
	nav, ok := res.(msg.Navigate)
	if !ok || nav.Target != msg.TDashboard {
		t.Errorf("expected Navigate to TDashboard, got %+v", res)
	}
}

func TestDashboardModel(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(t, db)
	team := mustCreateTeam(t, db, "TeamZero")

	d := NewDashboard(svc, team)
	d.SetSize(110, 30)
	cmd := d.Refresh()
	if cmd != nil {
		_ = cmd()
	}

	v := d.View()
	if !strings.Contains(v, "TOTAL SCORE") || !strings.Contains(v, "TACTICAL HUD") || !strings.Contains(v, "Submit Flag") {
		t.Errorf("Dashboard View missing cards or menu: %s", v)
	}

	// Test narrow view
	d.SetSize(70, 25)
	vNarrow := d.View()
	if len(vNarrow) == 0 {
		t.Errorf("Dashboard narrow view empty")
	}
}

func TestSubmitFlagModel(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(t, db)
	team := mustCreateTeam(t, db, "TestAlpha")

	m := NewSubmitFlag(svc, team)
	m.SetSize(100, 30)
	_ = m.Refresh()

	v := m.View()
	if !strings.Contains(v, "SELECT TARGET CHALLENGE") || !strings.Contains(v, "Web Infiltration") {
		t.Errorf("SubmitFlag View missing target rounds: %s", v)
	}
}

func TestRequestHintModel(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(t, db)
	team := mustCreateTeam(t, db, "HintTeam")

	m := NewRequestHint(svc, team)
	m.SetSize(100, 30)
	_ = m.Refresh()

	v := m.View()
	if !strings.Contains(v, "1. SELECT ROUND") || !strings.Contains(v, "Web Infiltration") {
		t.Errorf("RequestHint View missing wizard steps or round: %s", v)
	}

	// Dispense hint and verify persistence after Refresh
	_, _, err := svc.DirectRedeemHint(team, 1, "plain")
	if err != nil {
		t.Fatalf("DirectRedeemHint failed: %v", err)
	}

	_ = m.Refresh()
	vAfter := m.View()
	if !strings.Contains(vAfter, "INTEL") {
		t.Errorf("RequestHint View missing unlocked intel indicator after refresh: %s", vAfter)
	}
}

func TestSkipRoundModel(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(t, db)
	team := mustCreateTeam(t, db, "SkipTeam")

	m := NewSkipRound(svc, team)
	m.SetSize(100, 30)
	_ = m.Refresh()

	v := m.View()
	if !strings.Contains(v, "STRATEGIC BYPASS") || !strings.Contains(v, "Web Infiltration") {
		t.Errorf("SkipRound View missing bypass menu: %s", v)
	}
}

func TestScoreboardModel(t *testing.T) {
	db := newTestDB(t)
	team := mustCreateTeam(t, db, "TopTeam")

	m := NewScoreboard(db, team)
	m.SetSize(100, 30)
	_ = m.Refresh()

	v := m.View()
	if !strings.Contains(v, "TEAM IDENTITY") || !strings.Contains(v, "Live Auto-Sync Active") {
		t.Errorf("Scoreboard View missing table or sync text: %s", v)
	}
}

func TestTeamStatusModel(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(t, db)
	team := mustCreateTeam(t, db, "StatusTeam")

	// Unlock a hint first
	_, _, err := svc.DirectRedeemHint(team, 1, "plain")
	if err != nil {
		t.Fatalf("DirectRedeemHint failed: %v", err)
	}

	m := NewTeamStatus(svc, team)
	m.SetSize(100, 60)
	_ = m.Refresh()

	v := m.View()
	if !strings.Contains(v, "CHALLENGE ENGAGEMENT MATRIX") || !strings.Contains(v, "FINANCIAL BOUNTY LEDGER") {
		t.Errorf("TeamStatus View missing matrix or ledger: %s", v)
	}
	if !strings.Contains(v, "UNLOCKED HINTS ARCHIVE") || !strings.Contains(v, "Check cookies") {
		t.Errorf("TeamStatus View missing unlocked hint clue text: %s", v)
	}
}

func TestRegisterModel(t *testing.T) {
	db := newTestDB(t)
	team := mustCreateTeam(t, db, "RegTeam")

	m := NewRegister(db, team)
	m.SetSize(90, 30)

	v := m.View()
	if !strings.Contains(v, "INITIAL ACCOUNT PROVISIONING") || !strings.Contains(v, "Password") {
		t.Errorf("Register View missing provisioning card: %s", v)
	}

	// Validation checks
	if err := validateRegistration("short", "short"); err == nil {
		t.Errorf("expected error on short password")
	}
	if err := validateRegistration("password123", "mismatch123"); err == nil {
		t.Errorf("expected error on password mismatch")
	}
}
