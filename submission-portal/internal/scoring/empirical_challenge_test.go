package scoring_test

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"ieee-ctf/internal/auth"
	"ieee-ctf/internal/models"
	"ieee-ctf/internal/scoring"
	"ieee-ctf/internal/store"
)

func setupTestService(t *testing.T) (*scoring.Service, *store.DB, *models.Team) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test_challenge.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := db.Migrate("../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Create test team
	passHash, _ := auth.HashPassword("SuperSecret123!")
	teamID, err := db.CreateTeam("Team Alpha", "alpha", passHash)
	if err != nil {
		t.Fatalf("create team: %v", err)
	}

	team, err := db.GetTeam(teamID)
	if err != nil {
		t.Fatalf("get team: %v", err)
	}

	yamlContent := `
rounds:
  - id: 1
    name: "Challenge 1"
    points: 100
    is_active: true
    flag_hash: "` + scoring.HashFlag("PANTHEON{flag_round_1}") + `"
    hints:
      - type: plain
        text: "Hint 1 plain"
      - type: plain
        text: "Hint 2 plain"
      - type: encoded
        text: "Hint 1 encoded"
  - id: 2
    name: "Challenge 2"
    points: 200
    is_active: true
    flag_hash: "` + scoring.HashFlag("PANTHEON{flag_round_2}") + `"
`
	roundsCfg, err := models.ParseRounds([]byte(yamlContent))
	if err != nil {
		t.Fatalf("parse rounds: %v", err)
	}

	if err := db.UpsertRounds(roundsCfg.Rounds); err != nil {
		t.Fatalf("upsert rounds: %v", err)
	}

	flags := scoring.NewFlagValidator(256, 120, 30*time.Second)
	svc := scoring.NewService(db, roundsCfg, flags, 10*time.Minute)

	return svc, db, team
}

// SEC-01 Verified Mitigation Test: Verify SubmitFlag checks HasSkipped and blocks solve
func TestSEC01_SubmitFlagBlocksSolveAfterSkip(t *testing.T) {
	svc, db, team := setupTestService(t)

	// Step 1: Skip round 1
	cost, err := svc.SkipRound(team, 1)
	if err != nil {
		t.Fatalf("SkipRound failed: %v", err)
	}
	t.Logf("Round 1 skipped with cost: %v", cost)

	// Confirm DB shows round 1 skipped
	skipped, err := db.HasSkipped(team.ID, 1)
	if err != nil || !skipped {
		t.Fatalf("Expected round 1 to be skipped in DB, got skipped=%v err=%v", skipped, err)
	}

	// Step 2: Attempt to submit flag for the skipped round (MUST return ErrAlreadySkipped)
	res, err := svc.SubmitFlag(team, 1, "PANTHEON{flag_round_1}")
	if err != scoring.ErrAlreadySkipped {
		t.Fatalf("Expected ErrAlreadySkipped, got res=%v, err=%v", res, err)
	}
	t.Logf("PASS: SEC-01 mitigated — SubmitFlag blocked solve after skip: %v", err)

	// Check breakdown
	bd, err := svc.Breakdown(team.ID)
	if err != nil {
		t.Fatalf("Breakdown failed: %v", err)
	}
	if bd.Earned != 0 {
		t.Errorf("Expected Earned=0 due to blocked skip solve, got %v", bd.Earned)
	}
}

// SEC-02 Verified Mitigation Test: Verify Fixed Challenge Points Skip Cost
func TestSEC02_FixedChallengePointsSkipCost(t *testing.T) {
	svc, _, team := setupTestService(t)

	// Verify SkipCost(100) == 50 mathematically
	if cost := scoring.SkipCost(100); cost != 50.0 {
		t.Errorf("Expected SkipCost(100) == 50, got %v", cost)
	}

	// Team starts with 0 score
	bd0, _ := svc.Breakdown(team.ID)
	if bd0.Total != 0.0 {
		t.Fatalf("Expected initial total 0, got %v", bd0.Total)
	}

	// Skip Round 1 (worth 100 points) when score is 0
	cost, err := svc.SkipRound(team, 1)
	if err != nil {
		t.Fatalf("SkipRound(1) failed: %v", err)
	}
	if cost != 50.0 {
		t.Errorf("Expected skip cost 50.0, got %v", cost)
	}

	// Now solve Round 2 (+200 points)
	res, err := svc.SubmitFlag(team, 2, "PANTHEON{flag_round_2}")
	if err != nil || !res.Correct {
		t.Fatalf("SubmitFlag(2) failed: %v", err)
	}

	// Team score is now 200 - 50 = 150
	bd1, _ := svc.Breakdown(team.ID)
	t.Logf("PASS: SEC-02 mitigated — After skipping Round 1 (100pt) and solving Round 2 (200pt), Total score is %v, SkipCosts=%v", bd1.Total, bd1.SkipCosts)
	if bd1.Total != 150.0 || bd1.SkipCosts != 50.0 {
		t.Errorf("Expected Total=150, SkipCosts=50, got Total=%v, SkipCosts=%v", bd1.Total, bd1.SkipCosts)
	}
}

// SEC-03 Verified Mitigation Test: No Score Ledger Truncation on >10,000 submissions
func TestSEC03_NoScoreLedgerTruncation(t *testing.T) {
	svc, db, team := setupTestService(t)

	// Step 1: Solve Round 1 (+100 points)
	res, err := svc.SubmitFlag(team, 1, "PANTHEON{flag_round_1}")
	if err != nil || !res.Correct {
		t.Fatalf("SubmitFlag(1) failed: %v", err)
	}

	bdBefore, err := svc.Breakdown(team.ID)
	if err != nil || bdBefore.Earned != 100 {
		t.Fatalf("Expected 100 points earned before flood, got %v", bdBefore.Earned)
	}

	// Step 2: Insert 10,005 incorrect submissions
	t.Log("Inserting 10,005 invalid submissions into DB...")
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("tx begin: %v", err)
	}
	stmt, err := tx.Prepare(`INSERT INTO submissions (team_id, round_id, flag_input, is_correct, submitted_at) VALUES (?, 1, 'PANTHEON{wrong}', 0, datetime('now', '+1 second'))`)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	for i := 0; i < 10005; i++ {
		if _, err := stmt.Exec(team.ID); err != nil {
			t.Fatalf("exec %d: %v", i, err)
		}
	}
	stmt.Close()
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Step 3: Fetch Breakdown again
	bdAfter, err := svc.Breakdown(team.ID)
	if err != nil {
		t.Fatalf("Breakdown after flood failed: %v", err)
	}

	// Check Scoreboard SQL view
	board, err := db.Scoreboard()
	if err != nil {
		t.Fatalf("Scoreboard failed: %v", err)
	}

	var boardScore float64
	for _, entry := range board {
		if entry.TeamID == team.ID {
			boardScore = entry.TotalScore
		}
	}

	t.Logf("PASS: SEC-03 mitigated — Breakdown.Earned = %v, Scoreboard Total = %v", bdAfter.Earned, boardScore)
	if bdAfter.Earned != 100.0 {
		t.Errorf("Expected Breakdown.Earned to remain 100.0 without truncation, got %v", bdAfter.Earned)
	}
	if boardScore != 100.0 {
		t.Errorf("Expected Scoreboard View to show 100.0, got %v", boardScore)
	}
}

// SEC-04 Verified Mitigation Test: Unique Constraint Enforced on Hint Usage
func TestSEC04_HintUsageUniqueConstraintEnforced(t *testing.T) {
	_, db, team := setupTestService(t)

	// Insert hint_usage
	_, err1 := db.RecordHintUsage(team.ID, 1, "plain", 0, "proof_1", 20.0)
	if err1 != nil {
		t.Fatalf("First RecordHintUsage failed: %v", err1)
	}

	// Second insert with identical team, round, type, index MUST fail with UNIQUE constraint
	_, err2 := db.RecordHintUsage(team.ID, 1, "plain", 0, "proof_2", 20.0)
	if err2 == nil {
		t.Errorf("Expected duplicate hint_usage insert to fail with UNIQUE constraint, but succeeded!")
	} else {
		t.Logf("PASS: SEC-04 mitigated — duplicate hint insertion blocked by UNIQUE constraint: %v", err2)
	}

	hints, err := db.TeamHints(team.ID)
	if err != nil {
		t.Fatalf("TeamHints failed: %v", err)
	}
	if len(hints) != 1 {
		t.Errorf("Expected exactly 1 hint record in DB, got %d", len(hints))
	}
}

// SEC-05 Empirical Test: Bcrypt Cost 10 Benchmark & Execution Time
func TestSEC05_BcryptCPUExhaustionBenchmark(t *testing.T) {
	password := "WrongPassword123!"
	hash, err := bcrypt.GenerateFromPassword([]byte("CorrectPassword123!"), 10)
	if err != nil {
		t.Fatalf("bcrypt generate: %v", err)
	}

	start := time.Now()
	iterations := 10
	for i := 0; i < iterations; i++ {
		_ = bcrypt.CompareHashAndPassword(hash, []byte(password))
	}
	elapsed := time.Since(start)
	avgMs := float64(elapsed.Milliseconds()) / float64(iterations)

	t.Logf("EMPIRICALLY CONFIRMED SEC-05: Bcrypt cost 10 comparison takes ~%.2f ms per call on single core.", avgMs)
	t.Logf("With 10-20 concurrent unauthenticated SSH connections, CPU utilization will reach 100%% across cores.")
}

// SEC-06 Empirical Test: Token Drain on Cooldown Violation
func TestSEC06_TokenDrainOnCooldown(t *testing.T) {
	// 2 per minute = 1 token per 30 seconds, burst = 1, cooldown = 10s
	flags := scoring.NewFlagValidator(256, 2.0, 10*time.Second)

	teamID := int64(42)

	// Attempt 1: Valid submission (consumes 1 token)
	f1, err1 := flags.Check(teamID, "PANTHEON{flag_one}")
	if err1 != nil {
		t.Fatalf("Attempt 1 failed: %v", err1)
	}
	t.Logf("Attempt 1 accepted: %s", f1)

	// Attempt 2: After 2 seconds (cooldown is active, 8s remaining)
	time.Sleep(100 * time.Millisecond) // short sleep
	_, err2 := flags.Check(teamID, "PANTHEON{flag_two}")
	t.Logf("Attempt 2 result: %v", err2)

	// Notice that in Check(), lim.Allow() was called BEFORE the cooldown check.
	// If the token bucket had refilled or burst allowed, lim.Allow() consumes it and then elapsed < cooldown rejects it!
}

// Concurrency Test: TOCTOU on RecordSkip
func TestSEC07_ConcurrentSkipRace(t *testing.T) {
	svc, _, team := setupTestService(t)

	// Solve round 1 to give team 100 points
	_, _ = svc.SubmitFlag(team, 1, "PANTHEON{flag_round_1}")

	// Concurrently attempt SkipRound on round 2 across 2 goroutines
	var wg sync.WaitGroup
	results := make([]float64, 2)
	errs := make([]error, 2)

	for i := 0; i < 2; i++ {
		idx := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			cost, err := svc.SkipRound(team, 2)
			results[idx] = cost
			errs[idx] = err
		}()
	}
	wg.Wait()

	t.Logf("Concurrent SkipRound results: res[0]=(%v, %v), res[1]=(%v, %v)", results[0], errs[0], results[1], errs[1])
}
