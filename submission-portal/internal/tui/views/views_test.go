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
    description: "Exploit web authentication vulnerabilities to acquire tokens"
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
    description: "Smash stack frame boundaries to hijack control flow"
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

func TestRoundDescriptionDisplay(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(t, db)
	team := mustCreateTeam(t, db, "DescTeam")

	// 1. SubmitFlagModel: description displayed in list view and HUD
	sf := NewSubmitFlag(svc, team)
	sf.SetSize(100, 30)
	_ = sf.Refresh()
	vSF := sf.View()
	if !strings.Contains(vSF, "Exploit web authentication") {
		t.Errorf("SubmitFlag view missing round description: %s", vSF)
	}

	// 2. RequestHintModel: description displayed in round selection
	rh := NewRequestHint(svc, team)
	rh.SetSize(100, 30)
	_ = rh.Refresh()
	vRH := rh.View()
	if !strings.Contains(vRH, "Exploit web authentication") {
		t.Errorf("RequestHint view missing round description: %s", vRH)
	}

	// 3. SkipRoundModel: description displayed in skip list
	sk := NewSkipRound(svc, team)
	sk.SetSize(100, 30)
	_ = sk.Refresh()
	vSK := sk.View()
	if !strings.Contains(vSK, "Exploit web authentication") {
		t.Errorf("SkipRound view missing round description: %s", vSK)
	}

	// 4. TeamStatusModel: description displayed in engagement matrix
	ts := NewTeamStatus(svc, team)
	ts.SetSize(100, 60)
	_ = ts.Refresh()
	vTS := ts.View()
	if !strings.Contains(vTS, "Exploit web authentication") {
		t.Errorf("TeamStatus view missing round description: %s", vTS)
	}
}

func TestSolveLimitViewDisplay(t *testing.T) {
	db := newTestDB(t)
	yamlContent := `
rounds:
  - id: 11
    name: "Campus Riddle Limited"
    description: "Limited to first 3 solves"
    points: 100
    is_active: true
    limit_solves: true
    max_solves: 3
    flag_hash: "3b006c0ab342be3e7ef08855eec1314902a20caeebeecb700147cb9a2bc2245e"
`
	roundsConfig, err := models.ParseRounds([]byte(yamlContent))
	if err != nil {
		t.Fatalf("ParseRounds: %v", err)
	}
	if err := db.UpsertRounds(roundsConfig.Rounds); err != nil {
		t.Fatalf("UpsertRounds: %v", err)
	}

	teamA := mustCreateTeam(t, db, "Alpha")
	teamB := mustCreateTeam(t, db, "Beta")
	teamC := mustCreateTeam(t, db, "Gamma")
	teamD := mustCreateTeam(t, db, "Delta")

	flags := scoring.NewFlagValidator(256, 10, 0)
	svc := scoring.NewService(db, roundsConfig, flags, 5*time.Minute)

	// Submit 1 solve by teamA
	_, err = svc.SubmitFlag(teamA, 11, "PANTHEON{valid_token_string_here_123}")
	// Hash in config was 3b00... so let's directly record correct submission in DB for teamA
	_, _ = db.RecordSubmission(teamA.ID, 11, "flag", true)

	// Now check view for teamB: should show 2/3 LEFT
	sf := NewSubmitFlag(svc, teamB)
	sf.SetSize(100, 30)
	_ = sf.Refresh()
	vSF := sf.View()
	if !strings.Contains(vSF, "2/3 LEFT") && !strings.Contains(vSF, "2/3 solves left") {
		t.Errorf("SubmitFlag view missing remaining solve slots: %s", vSF)
	}

	// Check TeamStatus for teamB
	ts := NewTeamStatus(svc, teamB)
	ts.SetSize(100, 60)
	_ = ts.Refresh()
	vTS := ts.View()
	if !strings.Contains(vTS, "2/3 LEFT") {
		t.Errorf("TeamStatus view missing 2/3 LEFT badge: %s", vTS)
	}

	// Now record 2 more solves to fill the quota (total 3)
	_, _ = db.RecordSubmission(teamB.ID, 11, "flag", true)
	_, _ = db.RecordSubmission(teamC.ID, 11, "flag", true)

	// Check view for teamD: should show QUOTA FULL
	sfD := NewSubmitFlag(svc, teamD)
	sfD.SetSize(100, 30)
	_ = sfD.Refresh()
	vSFQuota := sfD.View()
	if !strings.Contains(vSFQuota, "QUOTA FULL") {
		t.Errorf("SubmitFlag view missing QUOTA FULL badge when limit reached: %s", vSFQuota)
	}

	tsD := NewTeamStatus(svc, teamD)
	tsD.SetSize(100, 60)
	_ = tsD.Refresh()
	vTSQuota := tsD.View()
	if !strings.Contains(vTSQuota, "QUOTA FULL") {
		t.Errorf("TeamStatus view missing QUOTA FULL badge when limit reached: %s", vTSQuota)
	}
}

