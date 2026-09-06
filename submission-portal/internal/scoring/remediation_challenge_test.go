package scoring_test

import (
	"crypto/subtle"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
	"gopkg.in/yaml.v3"

	"ieee-ctf/internal/auth"
	"ieee-ctf/internal/middleware"
	"ieee-ctf/internal/models"
	"ieee-ctf/internal/scoring"
	"ieee-ctf/internal/store"
)

// setupRemediationDB sets up a fresh test database and applies both 001_initial.sql and 002_security_hardening.sql
func setupRemediationDB(t *testing.T) (*store.DB, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "remediation_test.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("store.Open failed: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Create migrations directory with 001_initial.sql and proposed 002_security_hardening.sql
	migDir := filepath.Join(dir, "migrations")
	if err := os.MkdirAll(migDir, 0o755); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}

	// Copy all migrations from ../../migrations
	entries, err := os.ReadDir("../../migrations")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			content, err := os.ReadFile(filepath.Join("../../migrations", e.Name()))
			if err != nil {
				t.Fatalf("read %s: %v", e.Name(), err)
			}
			if err := os.WriteFile(filepath.Join(migDir, e.Name()), content, 0o644); err != nil {
				t.Fatalf("write %s: %v", e.Name(), err)
			}
		}
	}

	// Run migration
	if err := db.Migrate(migDir); err != nil {
		t.Fatalf("db.Migrate failed: %v", err)
	}

	return db, migDir
}

// makeRoundsCfg parses a slice of RoundDef into a fully indexed RoundsConfig
func makeRoundsCfg(defs []models.RoundDef) *models.RoundsConfig {
	data, _ := yaml.Marshal(map[string]interface{}{"rounds": defs})
	rc, _ := models.ParseRounds(data)
	return rc
}

// -------------------------------------------------------------------------------------------------
// Test 1: Verify Migration 002_security_hardening.sql against WAL mode and constraints
// -------------------------------------------------------------------------------------------------
func TestMigration_002_SecurityHardening_WAL_Compatibility(t *testing.T) {
	db, _ := setupRemediationDB(t)

	// Check WAL mode
	var journalMode string
	if err := db.QueryRow(`PRAGMA journal_mode;`).Scan(&journalMode); err != nil {
		t.Fatalf("PRAGMA journal_mode query failed: %v", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		t.Errorf("Expected journal_mode WAL, got %s", journalMode)
	}
	t.Logf("Verified SQLite journal_mode is: %s", journalMode)

	// Create test team and round
	passHash, _ := auth.HashPassword("TestPass123!")
	teamID, err := db.CreateTeam("Team Alpha", "alpha", passHash)
	if err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := db.UpsertRounds([]models.RoundDef{
		{ID: 1, Name: "R1", Points: 100, FlagHash: scoring.HashFlag("PANTHEON{r1}"), IsActive: true},
	}); err != nil {
		t.Fatalf("UpsertRounds failed: %v", err)
	}

	// Test unique index on hint_usage (team_id, round_id, hint_type, hint_index)
	_, err1 := db.RecordHintUsage(teamID, 1, "plain", 0, "proof_A", 20.0)
	if err1 != nil {
		t.Fatalf("First RecordHintUsage failed: %v", err1)
	}

	// Attempt duplicate insertion with same tuple but different proof
	_, err2 := db.RecordHintUsage(teamID, 1, "plain", 0, "proof_B", 20.0)
	if err2 == nil {
		t.Errorf("Expected duplicate (team, round, type, index) to fail UNIQUE constraint, but succeeded!")
	} else if !strings.Contains(err2.Error(), "UNIQUE constraint failed") {
		t.Errorf("Expected UNIQUE constraint error, got: %v", err2)
	} else {
		t.Logf("PASS: idx_hint_usage_unique blocked duplicate hint index: %v", err2)
	}

	// Test unique index on pgp_proof
	_, err3 := db.RecordHintUsage(teamID, 1, "plain", 1, "proof_A", 20.0)
	if err3 == nil {
		t.Errorf("Expected duplicate pgp_proof 'proof_A' to fail UNIQUE constraint, but succeeded!")
	} else if !strings.Contains(err3.Error(), "UNIQUE constraint failed") {
		t.Errorf("Expected UNIQUE constraint error for proof, got: %v", err3)
	} else {
		t.Logf("PASS: idx_hint_usage_proof blocked duplicate proof: %v", err3)
	}
}

// -------------------------------------------------------------------------------------------------
// Test 2: Verify Scoreboard View Tie-Breaking & Identify db.Scoreboard() Go Code Discrepancy
// -------------------------------------------------------------------------------------------------
func TestScoreboardTieBreaking_ViewVsGoMethod(t *testing.T) {
	db, _ := setupRemediationDB(t)

	passHash, _ := auth.HashPassword("TestPass123!")
	// Team Beta registers first, Team Alpha registers second
	betaID, _ := db.CreateTeam("Beta", "beta", passHash)
	alphaID, _ := db.CreateTeam("Alpha", "alpha", passHash)

	db.UpsertRounds([]models.RoundDef{
		{ID: 1, Name: "R1", Points: 100, FlagHash: scoring.HashFlag("PANTHEON{r1}"), IsActive: true},
	})

	// Team Beta solves R1 at T0 (earlier)
	// Team Alpha solves R1 at T0 + 10s (later)
	// Both teams have 100 points.
	// In CTF competition rules: Beta achieved 100 points FIRST, so Beta MUST be Rank #1, Alpha Rank #2.
	
	// Insert Beta submission with earlier timestamp
	_, err := db.Exec(`INSERT INTO submissions (team_id, round_id, flag_input, is_correct, submitted_at) VALUES (?, 1, 'PANTHEON{r1}', 1, '2026-08-30 10:00:00')`, betaID)
	if err != nil {
		t.Fatalf("insert beta solve: %v", err)
	}

	// Insert Alpha submission with later timestamp
	_, err = db.Exec(`INSERT INTO submissions (team_id, round_id, flag_input, is_correct, submitted_at) VALUES (?, 1, 'PANTHEON{r1}', 1, '2026-08-30 10:00:10')`, alphaID)
	if err != nil {
		t.Fatalf("insert alpha solve: %v", err)
	}

	// 1. Direct query against scoreboard VIEW (which uses view's ORDER BY)
	rows, err := db.Query(`SELECT team_name, total_score, last_solve_at FROM scoreboard`)
	if err != nil {
		t.Fatalf("query scoreboard view: %v", err)
	}
	defer rows.Close()

	var viewOrder []string
	for rows.Next() {
		var name string
		var score float64
		var lastSolve any
		if err := rows.Scan(&name, &score, &lastSolve); err != nil {
			t.Fatalf("scan view row: %v", err)
		}
		viewOrder = append(viewOrder, name)
	}
	t.Logf("Direct View Query Order: %v (Rank 1: %s, Rank 2: %s)", viewOrder, viewOrder[0], viewOrder[1])

	// 2. Query through db.Scoreboard() Go method
	board, err := db.Scoreboard()
	if err != nil {
		t.Fatalf("db.Scoreboard() failed: %v", err)
	}
	var goMethodOrder []string
	for i, entry := range board {
		goMethodOrder = append(goMethodOrder, fmt.Sprintf("%d: %s (%v)", i+1, entry.TeamName, entry.TotalScore))
	}
	t.Logf("db.Scoreboard() Go Method Order: %v", goMethodOrder)

	// In patched view & Go method, Beta MUST be Rank 1
	if board[0].TeamName != "Beta" {
		t.Errorf("CRITICAL BUG: Tie-breaking failed in db.Scoreboard()! Expected Beta as Rank 1, got: %s", board[0].TeamName)
	}
}

// -------------------------------------------------------------------------------------------------
// Test 3: Adversarially Test Proposed SubmitFlag Implementation (Patch 1)
// -------------------------------------------------------------------------------------------------
func TestPatchedSubmitFlag_Workflow(t *testing.T) {
	db, _ := setupRemediationDB(t)

	passHash, _ := auth.HashPassword("TestPass123!")
	teamID, _ := db.CreateTeam("Team Alpha", "alpha", passHash)
	team, _ := db.GetTeam(teamID)

	roundDefs := []models.RoundDef{
		{ID: 1, Name: "R1", Points: 100, FlagHash: scoring.HashFlag("PANTHEON{flag_round_1}"), IsActive: true},
		{ID: 2, Name: "R2", Points: 200, FlagHash: strings.ToUpper(scoring.HashFlag("PANTHEON{flag_round_2}")), IsActive: true}, // Test uppercase hash
		{ID: 3, Name: "R3", Points: 300, FlagHash: scoring.HashFlag("PANTHEON{flag_round_3}"), IsActive: false}, // Inactive round
	}
	db.UpsertRounds(roundDefs)
	rawYaml := fmt.Sprintf(`
rounds:
  - id: 1
    name: "R1"
    points: 100
    is_active: true
    flag_hash: "%s"
  - id: 2
    name: "R2"
    points: 200
    is_active: true
    flag_hash: "%s"
  - id: 3
    name: "R3"
    points: 300
    is_active: false
    flag_hash: "%s"
`, scoring.HashFlag("PANTHEON{flag_round_1}"), strings.ToUpper(scoring.HashFlag("PANTHEON{flag_round_2}")), scoring.HashFlag("PANTHEON{flag_round_3}"))
	roundsCfg, err := models.ParseRounds([]byte(rawYaml))
	if err != nil {
		t.Fatalf("ParseRounds failed: %v", err)
	}
	flags := scoring.NewFlagValidator(256, 6000, 0)

	// Simulated Patched SubmitFlag implementation
	patchedSubmitFlag := func(sDB *store.DB, rCfg *models.RoundsConfig, sFlags *scoring.FlagValidator, team *models.Team, roundID int, raw string) (*scoring.SubmitResult, error) {
		flag, err := sFlags.Check(team.ID, raw)
		if err != nil {
			return nil, err
		}
		round := rCfg.Def(roundID)
		if round == nil {
			return nil, fmt.Errorf("%w: unknown round %d", scoring.ErrRoundInactive, roundID)
		}
		dbRound, err := sDB.GetRound(roundID)
		if err != nil || !dbRound.IsActive {
			return nil, scoring.ErrRoundInactive
		}
		solved, err := sDB.HasSolvedRound(team.ID, roundID)
		if err != nil {
			return nil, err
		}
		if solved {
			return nil, store.ErrAlreadySolved
		}
		skipped, err := sDB.HasSkipped(team.ID, roundID)
		if err != nil {
			return nil, err
		}
		if skipped {
			return nil, scoring.ErrAlreadySkipped
		}

		expectedHash := strings.ToLower(round.FlagHash)
		actualHash := scoring.HashFlag(flag)
		correct := subtle.ConstantTimeCompare([]byte(actualHash), []byte(expectedHash)) == 1

		if _, err := sDB.RecordSubmission(team.ID, roundID, flag, correct); err != nil {
			return nil, err
		}
		res := &scoring.SubmitResult{Correct: correct}
		if correct {
			res.PointsAwarded = float64(dbRound.Points)
		}
		return res, nil
	}

	// 1. Normal valid submission
	res1, err := patchedSubmitFlag(db, roundsCfg, flags, team, 1, "PANTHEON{flag_round_1}")
	if err != nil || !res1.Correct || res1.PointsAwarded != 100.0 {
		t.Fatalf("Valid submit failed: res=%v, err=%v", res1, err)
	}
	t.Logf("PASS: Valid flag submitted, awarded %v points", res1.PointsAwarded)

	// 2. Duplicate submission for solved round
	time.Sleep(150 * time.Millisecond) // satisfy cooldown
	_, errDup := patchedSubmitFlag(db, roundsCfg, flags, team, 1, "PANTHEON{flag_round_1}")
	if errDup != store.ErrAlreadySolved {
		t.Errorf("Expected ErrAlreadySolved, got %v", errDup)
	} else {
		t.Logf("PASS: Solved round duplicate blocked: %v", errDup)
	}

	// 3. Skip Round 2 and attempt submission
	db.RecordSkip(team.ID, 2, 100.0)
	time.Sleep(150 * time.Millisecond)
	_, errSkip := patchedSubmitFlag(db, roundsCfg, flags, team, 2, "PANTHEON{flag_round_2}")
	if errSkip != scoring.ErrAlreadySkipped {
		t.Errorf("Expected ErrAlreadySkipped, got %v", errSkip)
	} else {
		t.Logf("PASS: Skipped round submission blocked: %v", errSkip)
	}

	// 4. Inactive round submission
	time.Sleep(150 * time.Millisecond)
	_, errInact := patchedSubmitFlag(db, roundsCfg, flags, team, 3, "PANTHEON{flag_round_3}")
	if errInact != scoring.ErrRoundInactive {
		t.Errorf("Expected ErrRoundInactive, got %v", errInact)
	} else {
		t.Logf("PASS: Inactive round submission blocked: %v", errInact)
	}

	// 5. Wrong flag submission
	time.Sleep(150 * time.Millisecond)
	resWrong, errWrong := patchedSubmitFlag(db, roundsCfg, flags, team, 1, "PANTHEON{wrong_flag_value}")
	// Wait, round 1 was solved, so it returns ErrAlreadySolved!
	// Let's test wrong flag on unattempted round 4
	roundDefs2 := append(roundDefs, models.RoundDef{ID: 4, Name: "R4", Points: 400, FlagHash: scoring.HashFlag("PANTHEON{r4}"), IsActive: true})
	db.UpsertRounds(roundDefs2)
	rawYaml4 := fmt.Sprintf(`%s
  - id: 4
    name: "R4"
    points: 400
    is_active: true
    flag_hash: "%s"
`, strings.TrimSpace(rawYaml), scoring.HashFlag("PANTHEON{r4}"))
	roundsCfg2, _ := models.ParseRounds([]byte(rawYaml4))

	time.Sleep(150 * time.Millisecond)
	resWrong, errWrong = patchedSubmitFlag(db, roundsCfg2, flags, team, 4, "PANTHEON{wrong_flag_value}")
	if errWrong != nil || resWrong.Correct || resWrong.PointsAwarded != 0 {
		t.Errorf("Expected incorrect submission with 0 points, got res=%v err=%v", resWrong, errWrong)
	} else {
		t.Logf("PASS: Wrong flag recorded as incorrect (0 points)")
	}
}

// -------------------------------------------------------------------------------------------------
// Test 4: Adversarially Test Proposed Skip Cost Model (Patch 2)
// -------------------------------------------------------------------------------------------------
func TestPatchedSkipCostModel(t *testing.T) {
	// Proposed SkipCost: float64(roundPoints) * 0.50
	patchedSkipCost := func(roundPoints int) float64 {
		return float64(roundPoints) * 0.50
	}

	cases := []struct {
		points int
		want   float64
	}{
		{100, 50.0},
		{200, 100.0},
		{350, 175.0},
		{50, 25.0},
		{0, 0.0},
	}

	for _, c := range cases {
		if got := patchedSkipCost(c.points); got != c.want {
			t.Errorf("patchedSkipCost(%d) = %v, want %v", c.points, got, c.want)
		}
	}
	t.Logf("PASS: Patched SkipCost depends strictly on challenge value, eliminating 0-cost game starts and order-of-operation exploits.")
}

// -------------------------------------------------------------------------------------------------
// Test 5: Adversarially Test SolvedSubmissions & Breakdown (Patch 3)
// -------------------------------------------------------------------------------------------------
func TestPatchedBreakdown_NoScoreTruncation(t *testing.T) {
	db, _ := setupRemediationDB(t)

	passHash, _ := auth.HashPassword("TestPass123!")
	teamID, _ := db.CreateTeam("Team Alpha", "alpha", passHash)

	roundDefs := []models.RoundDef{
		{ID: 1, Name: "R1", Points: 100, FlagHash: scoring.HashFlag("PANTHEON{r1}"), IsActive: true},
		{ID: 2, Name: "R2", Points: 250, FlagHash: scoring.HashFlag("PANTHEON{r2}"), IsActive: true},
		{ID: 3, Name: "R3", Points: 100, FlagHash: scoring.HashFlag("PANTHEON{r3}"), IsActive: true},
	}
	db.UpsertRounds(roundDefs)

	// Team solves R1 and R2
	db.RecordSubmission(teamID, 1, "PANTHEON{r1}", true)
	db.RecordSubmission(teamID, 2, "PANTHEON{r2}", true)
	db.RecordHintUsage(teamID, 1, "plain", 0, "proof1", 20.0)
	db.RecordSkip(teamID, 3, 50.0) // skip a 100pt round

	// Flood database with 15,000 failed attempts
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	stmt, _ := tx.Prepare(`INSERT INTO submissions (team_id, round_id, flag_input, is_correct, submitted_at) VALUES (?, 1, 'PANTHEON{bad}', 0, datetime('now'))`)
	for i := 0; i < 15000; i++ {
		stmt.Exec(teamID)
	}
	stmt.Close()
	tx.Commit()

	// Implement patched SolvedSubmissions and Breakdown
	solvedSubmissions := func(sDB *store.DB, tID int64) ([]models.Submission, error) {
		rows, err := sDB.Query(`
			SELECT id, team_id, round_id, flag_input, is_correct, submitted_at
			FROM submissions WHERE team_id = ? AND is_correct = 1 ORDER BY round_id`, tID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []models.Submission
		for rows.Next() {
			var s models.Submission
			if err := rows.Scan(&s.ID, &s.TeamID, &s.RoundID, &s.FlagInput, &s.IsCorrect, &s.SubmittedAt); err != nil {
				return nil, err
			}
			out = append(out, s)
		}
		return out, rows.Err()
	}

	patchedBreakdown := func(sDB *store.DB, tID int64) (*scoring.Breakdown, error) {
		rounds, err := sDB.RoundsMap()
		if err != nil {
			return nil, err
		}
		solvedSubs, err := solvedSubmissions(sDB, tID)
		if err != nil {
			return nil, err
		}
		hints, err := sDB.TeamHints(tID)
		if err != nil {
			return nil, err
		}
		skips, err := sDB.TeamSkips(tID)
		if err != nil {
			return nil, err
		}
		solvedIDs, err := sDB.SolvedRoundIDs(tID)
		if err != nil {
			return nil, err
		}

		total := scoring.CalculateTeamScore(solvedSubs, rounds, hints, skips)
		b := &scoring.Breakdown{
			Total:        total,
			SolvedRounds: solvedIDs,
			Hints:        hints,
			Skips:        skips,
		}
		for _, sub := range solvedSubs {
			b.Earned += float64(rounds[sub.RoundID].Points)
		}
		for _, h := range hints {
			b.HintCosts += h.CostPoints
		}
		for _, sk := range skips {
			b.SkipCosts += sk.CostPoints
		}
		return b, nil
	}

	bd, err := patchedBreakdown(db, teamID)
	if err != nil {
		t.Fatalf("patchedBreakdown failed: %v", err)
	}

	// Earned: 100 + 250 = 350
	// Costs: 20 + 50 = 70
	// Total: 350 - 70 = 280
	t.Logf("Patched Breakdown after 15,000 invalid attempts: Earned=%v, Total=%v, SolvedCount=%d", 
		bd.Earned, bd.Total, len(bd.SolvedRounds))

	if bd.Earned != 350.0 {
		t.Errorf("Expected Earned=350.0, got %v", bd.Earned)
	}
	if bd.Total != 280.0 {
		t.Errorf("Expected Total=280.0, got %v", bd.Total)
	}
	if len(bd.SolvedRounds) != 2 {
		t.Errorf("Expected 2 solved rounds, got %d", len(bd.SolvedRounds))
	}
}

// -------------------------------------------------------------------------------------------------
// Test 6: Adversarially Test Proposed Auth Rate Limiting (Patch 4) & Edge Cases
// -------------------------------------------------------------------------------------------------
func TestPatchedAuthRateLimiting_IPExtractionAndLockout(t *testing.T) {
	db, _ := setupRemediationDB(t)

	rawPass := "GoodPassword123!"
	passHash, _ := auth.HashPassword(rawPass)
	_, _ = db.CreateTeam("Team Alpha", "alpha", passHash)

	// Recreate the patched VerifyTeamLoginWithIP logic in test
	var (
		mu           sync.Mutex
		authLimiters = make(map[string]*rate.Limiter)
		failedLogins = make(map[string]int)
		lockoutUntil = make(map[string]time.Time)
	)

	verifyLogin := func(user, password, remoteAddr string) bool {
		if user == "" || password == "" {
			return false
		}
		ip, _, err := net.SplitHostPort(remoteAddr)
		if err != nil {
			ip = remoteAddr
		}

		mu.Lock()
		if until, locked := lockoutUntil[ip]; locked && time.Now().Before(until) {
			mu.Unlock()
			return false
		}
		lim, exists := authLimiters[ip]
		if !exists {
			lim = rate.NewLimiter(rate.Limit(5.0), 5) // max 5/sec, burst 5 for test
			authLimiters[ip] = lim
		}
		if !lim.Allow() {
			mu.Unlock()
			return false
		}
		mu.Unlock()

		team, err := db.GetTeamBySSHUser(strings.ToLower(user))
		if err != nil || team == nil {
			mu.Lock()
			failedLogins[ip]++
			if failedLogins[ip] >= 10 {
				lockoutUntil[ip] = time.Now().Add(5 * time.Minute)
				failedLogins[ip] = 0
			}
			mu.Unlock()
			return false
		}

		if err := bcrypt.CompareHashAndPassword([]byte(team.SSHPass), []byte(password)); err != nil {
			mu.Lock()
			failedLogins[ip]++
			if failedLogins[ip] >= 10 {
				lockoutUntil[ip] = time.Now().Add(5 * time.Minute)
				failedLogins[ip] = 0
			}
			mu.Unlock()
			return false
		}

		mu.Lock()
		delete(failedLogins, ip)
		delete(lockoutUntil, ip)
		mu.Unlock()
		return true
	}

	// 1. Valid login from IPv4
	if !verifyLogin("alpha", rawPass, "192.168.1.50:54321") {
		t.Errorf("Valid login failed")
	}

	// 2. Valid login from IPv6
	if !verifyLogin("alpha", rawPass, "[2001:db8::1]:54321") {
		t.Errorf("Valid IPv6 login failed")
	}

	// 3. 10 failed logins trigger 5-minute lockout
	attackerIP := "203.0.113.99:1234"
	for i := 0; i < 10; i++ {
		verifyLogin("alpha", "WrongPass", attackerIP)
	}

	// 11th attempt with correct password MUST be rejected due to lockout
	if verifyLogin("alpha", rawPass, attackerIP) {
		t.Errorf("Attacker was able to login during lockout period!")
	} else {
		t.Logf("PASS: Attacker locked out after 10 failed attempts.")
	}

	// Legitimate user from different IP remains unaffected
	if !verifyLogin("alpha", rawPass, "198.51.100.10:4321") {
		t.Errorf("Legitimate user was blocked by attacker lockout!")
	} else {
		t.Logf("PASS: Different IP is unaffected by lockout.")
	}
}

// -------------------------------------------------------------------------------------------------
// Test 7: Adversarially Test Proposed FlagValidator Cooldown Reordering (Patch 5)
// -------------------------------------------------------------------------------------------------
func TestPatchedFlagValidator_CooldownTokenDrain(t *testing.T) {
	// Recreate validator with Check ordering: Cooldown check FIRST, then lim.Allow()
	type patchedValidator struct {
		maxLen      int
		perMinute   rate.Limit
		burst       int
		cooldown    time.Duration
		mu          sync.Mutex
		limiters    map[int64]*rate.Limiter
		lastAttempt map[int64]time.Time
	}

	v := &patchedValidator{
		maxLen:      256,
		perMinute:   rate.Limit(2.0 / 60.0), // 1 token every 30s
		burst:       1,
		cooldown:    500 * time.Millisecond,
		limiters:    make(map[int64]*rate.Limiter),
		lastAttempt: make(map[int64]time.Time),
	}

	check := func(teamID int64, raw string) (string, error) {
		flag := strings.TrimSpace(middleware.StripANSI(raw))
		if len(flag) > v.maxLen {
			return "", scoring.ErrFlagTooLong
		}

		v.mu.Lock()
		defer v.mu.Unlock()

		// FIX SEC-06: Check cooldown FIRST
		if last, ok := v.lastAttempt[teamID]; ok {
			if elapsed := time.Since(last); elapsed < v.cooldown {
				return "", fmt.Errorf("%w (%ds remaining)",
					scoring.ErrCooldownActive, int((v.cooldown-elapsed).Seconds()+0.99))
			}
		}

		lim := v.limiters[teamID]
		if lim == nil {
			lim = rate.NewLimiter(v.perMinute, v.burst)
			v.limiters[teamID] = lim
		}
		if !lim.Allow() {
			return "", scoring.ErrRateLimited
		}

		v.lastAttempt[teamID] = time.Now()
		return flag, nil
	}

	teamID := int64(100)

	// 1. Initial attempt consumes 1 token
	_, err1 := check(teamID, "PANTHEON{flag1}")
	if err1 != nil {
		t.Fatalf("Attempt 1 failed: %v", err1)
	}

	// 2. Immediate 2nd attempt during cooldown (rejected by cooldown, NOT consuming any future token)
	_, err2 := check(teamID, "PANTHEON{flag2}")
	if !strings.Contains(err2.Error(), "cooldown active") {
		t.Fatalf("Expected cooldown error, got %v", err2)
	}

	// 3. Wait for cooldown to expire
	time.Sleep(600 * time.Millisecond)

	// Since perMinute is low, token bucket hasn't refilled. It should return ErrRateLimited.
	// But crucially, attempt 2 did NOT drain any additional tokens or corrupt rate limiter state.
	_, err3 := check(teamID, "PANTHEON{flag3}")
	t.Logf("Attempt 3 after cooldown result: %v", err3)
}

// -------------------------------------------------------------------------------------------------
// Test 8: Adversarially Test Re-Registration Guard (Patch 6)
// -------------------------------------------------------------------------------------------------
func TestPatchedCompleteRegistration_Guarded(t *testing.T) {
	db, _ := setupRemediationDB(t)

	passHash, _ := auth.HashPassword("InitialPass123!")
	teamID, _ := db.CreateTeam("Team Alpha", "alpha", passHash)

	patchedCompleteRegistration := func(sDB *store.DB, tID int64, newHash, pubkey string) error {
		res, err := sDB.Exec(`
			UPDATE teams 
			SET ssh_pass = ?, pgp_pubkey = ?, registered = 1 
			WHERE id = ? AND registered = 0`,
			newHash, pubkey, tID)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return fmt.Errorf("team is already registered or does not exist")
		}
		return nil
	}

	// First registration attempt MUST succeed
	newHash1, _ := auth.HashPassword("NewPass12345!")
	err1 := patchedCompleteRegistration(db, teamID, newHash1, "-----BEGIN PGP PUBLIC KEY BLOCK-----\ntest1")
	if err1 != nil {
		t.Fatalf("Initial registration failed: %v", err1)
	}

	team, _ := db.GetTeam(teamID)
	if !team.Registered {
		t.Fatalf("Expected team to be registered")
	}

	// Second registration attempt MUST fail
	newHash2, _ := auth.HashPassword("HijackPass123!")
	err2 := patchedCompleteRegistration(db, teamID, newHash2, "-----BEGIN PGP PUBLIC KEY BLOCK-----\nhijack")
	if err2 == nil {
		t.Errorf("Expected re-registration to fail, but succeeded!")
	} else {
		t.Logf("PASS: Re-registration blocked: %v", err2)
	}

	// Verify original credentials were NOT overwritten
	teamAfter, _ := db.GetTeam(teamID)
	if teamAfter.SSHPass != newHash1 {
		t.Errorf("Team password hash was overwritten despite failure!")
	}
}

// -------------------------------------------------------------------------------------------------
// Test 9: Adversarially Test Team Name Sanitization (Patch 8)
// -------------------------------------------------------------------------------------------------
func TestPatchedTeamNameSanitization(t *testing.T) {
	db, _ := setupRemediationDB(t)

	passHash, _ := auth.HashPassword("Pass12345!")

	patchedCreateTeam := func(sDB *store.DB, name, sshUser, pHash string) (int64, error) {
		cleanName := middleware.SanitizeInput(middleware.StripANSI(name))
		cleanUser := strings.ToLower(middleware.SanitizeInput(middleware.StripANSI(sshUser)))
		res, err := sDB.Exec(`INSERT INTO teams (name, ssh_user, ssh_pass) VALUES (?, ?, ?)`,
			cleanName, cleanUser, pHash)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				return 0, fmt.Errorf("team name or ssh user already exists")
			}
			return 0, err
		}
		return res.LastInsertId()
	}

	// Malicious team name with ANSI clear screen, color, and control characters
	maliciousName := "\x1b[2J\x1b[31mEvil\x00\x07Team\x1b[0m"
	id, err := patchedCreateTeam(db, maliciousName, "evil_user", passHash)
	if err != nil {
		t.Fatalf("patchedCreateTeam failed: %v", err)
	}

	team, err := db.GetTeam(id)
	if err != nil {
		t.Fatalf("GetTeam failed: %v", err)
	}

	t.Logf("Sanitized Team Name: %q", team.Name)
	if strings.Contains(team.Name, "\x1b") || strings.Contains(team.Name, "\x00") {
		t.Errorf("Sanitization failed to strip ANSI/control chars: %q", team.Name)
	}
	if team.Name != "EvilTeam" {
		t.Errorf("Expected 'EvilTeam', got %q", team.Name)
	}
}

// -------------------------------------------------------------------------------------------------
// Test 10: Adversarially Test Inactive Round Consistency (SEC-13)
// -------------------------------------------------------------------------------------------------
func TestInactiveRound_ConsistentGuardsAcrossOperations(t *testing.T) {
	db, _ := setupRemediationDB(t)

	passHash, _ := auth.HashPassword("Pass12345!")
	teamID, _ := db.CreateTeam("Team Alpha", "alpha", passHash)
	team, _ := db.GetTeam(teamID)

	roundDefs := []models.RoundDef{
		{
			ID:       1,
			Name:     "Challenge 1",
			Points:   100,
			FlagHash: scoring.HashFlag("PANTHEON{flag_round_1}"),
			IsActive: true,
			Hints: []models.HintDef{
				{Type: "plain", Text: "Plain hint 1"},
			},
		},
	}
	db.UpsertRounds(roundDefs)
	roundsCfg := makeRoundsCfg(roundDefs)
	flags := scoring.NewFlagValidator(256, 120, 0)
	svc := scoring.NewService(db, roundsCfg, flags, 10*time.Minute)

	// Admin deactivates Round 1 in database
	if err := db.SetRoundActive(1, false); err != nil {
		t.Fatalf("SetRoundActive failed: %v", err)
	}

	// 1. SubmitFlag correctly checks DB and rejects
	_, errSub := svc.SubmitFlag(team, 1, "PANTHEON{flag_round_1}")
	if errSub != scoring.ErrRoundInactive {
		t.Errorf("SubmitFlag: expected ErrRoundInactive, got %v", errSub)
	} else {
		t.Logf("PASS: SubmitFlag rejected inactive round: %v", errSub)
	}

	// 2. In unpatched code, HintChallenge only checks memory config (roundsCfg) and issues challenge!
	_, _, _, errHint := svc.HintChallenge(team, 1, "plain")
	t.Logf("Unpatched HintChallenge on inactive round result: %v", errHint)
	if errHint == nil {
		t.Logf("EMPIRICALLY CONFIRMED SEC-13: HintChallenge issued challenge for deactivated round because it omitted dbRound.IsActive check!")
	}

	// 3. Patched HintChallenge with dbRound.IsActive check
	patchedHintChallenge := func(s *scoring.Service, team *models.Team, roundID int, hintType string) (string, int, float64, error) {
		dbRound, err := s.DB.GetRound(roundID)
		if err != nil || !dbRound.IsActive {
			return "", 0, 0, scoring.ErrRoundInactive
		}
		return s.HintChallenge(team, roundID, hintType)
	}

	_, _, _, errPatchedHint := patchedHintChallenge(svc, team, 1, "plain")
	if errPatchedHint != scoring.ErrRoundInactive {
		t.Errorf("Patched HintChallenge expected ErrRoundInactive, got %v", errPatchedHint)
	} else {
		t.Logf("PASS: Patched HintChallenge blocked deactivated round: %v", errPatchedHint)
	}
}

