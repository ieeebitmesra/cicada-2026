package scoring_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"ieee-ctf/internal/models"
	"ieee-ctf/internal/scoring"
	"ieee-ctf/internal/store"
)

func TestSolveLimitEnforcement(t *testing.T) {
	yamlData := []byte(`
allow_hints: true
allow_skips: true

rounds:
  - id: 11
    name: "Round 11 — Limited Campus Riddle"
    description: "Campus Riddle with 3-solve limit"
    points: 100
    is_active: true
    allow_hints: false
    allow_skips: false
    limit_solves: true
    max_solves: 3
    flag_hash: "` + scoring.HashFlag("PANTHEON{campus_riddle_11}") + `"
    hints: []

  - id: 12
    name: "Round 12 — Unlimited Challenge"
    description: "Standard challenge with no solve limit"
    points: 200
    is_active: true
    allow_hints: false
    allow_skips: false
    flag_hash: "` + scoring.HashFlag("PANTHEON{unlimited_round_12}") + `"
    hints: []
`)

	rc, err := models.ParseRounds(yamlData)
	if err != nil {
		t.Fatalf("ParseRounds failed: %v", err)
	}

	if !rc.IsSolveLimited(11) {
		t.Fatalf("expected round 11 to be solve limited")
	}
	if rc.MaxAllowedSolves(11) != 3 {
		t.Fatalf("expected round 11 max allowed solves to be 3, got %d", rc.MaxAllowedSolves(11))
	}
	if rc.IsSolveLimited(12) {
		t.Fatalf("expected round 12 to not be solve limited")
	}

	// Setup DB & Service
	dbPath := filepath.Join(t.TempDir(), "test_limits.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer db.Close()

	if err := db.Migrate("../../migrations"); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	if err := db.UpsertRounds(rc.Rounds); err != nil {
		t.Fatalf("UpsertRounds: %v", err)
	}

	// Create 5 teams
	var teams []*models.Team
	for i := 1; i <= 5; i++ {
		teamID, err := db.CreateTeam("Team "+string(rune('A'+i-1)), "team"+string(rune('a'+i-1)), "hashed")
		if err != nil {
			t.Fatalf("CreateTeam %d: %v", i, err)
		}
		team, err := db.GetTeam(teamID)
		if err != nil {
			t.Fatalf("GetTeam %d: %v", i, err)
		}
		teams = append(teams, team)
	}

	flags := scoring.NewFlagValidator(256, 6000, 0)
	svc := scoring.NewService(db, rc, flags, 5*time.Minute)

	// Step 1: Incorrect submission — should not consume quota
	teamWrongID, _ := db.CreateTeam("Team Wrong", "wrong", "hashed")
	teamWrong, _ := db.GetTeam(teamWrongID)
	resWrong, err := svc.SubmitFlag(teamWrong, 11, "PANTHEON{wrong_flag}")
	if err != nil {
		t.Fatalf("SubmitFlag wrong: %v", err)
	}
	if resWrong.Correct {
		t.Fatalf("expected wrong submission to not be correct")
	}

	count, err := db.CountSolvesForRound(11)
	if err != nil {
		t.Fatalf("CountSolvesForRound: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 solves after wrong attempt, got %d", count)
	}

	// Step 2: First 3 teams submit correct flags — all succeed and receive 100 points
	for i := 0; i < 3; i++ {
		res, err := svc.SubmitFlag(teams[i], 11, "PANTHEON{campus_riddle_11}")
		if err != nil {
			t.Fatalf("Team %d correct submit failed: %v", i+1, err)
		}
		if !res.Correct || res.PointsAwarded != 100 {
			t.Fatalf("Team %d: expected correct=true, points=100, got correct=%v points=%v", i+1, res.Correct, res.PointsAwarded)
		}
	}

	// Verify solve count is 3
	count, err = db.CountSolvesForRound(11)
	if err != nil {
		t.Fatalf("CountSolvesForRound: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 solves, got %d", count)
	}

	solvesMap, err := db.SolvesCountMap()
	if err != nil {
		t.Fatalf("SolvesCountMap: %v", err)
	}
	if solvesMap[11] != 3 {
		t.Fatalf("expected solvesMap[11] == 3, got %d", solvesMap[11])
	}

	// Step 3: Team 4 attempts to submit correct flag — must be rejected with ErrMaxSolvesReached
	_, err = svc.SubmitFlag(teams[3], 11, "PANTHEON{campus_riddle_11}")
	if !errors.Is(err, scoring.ErrMaxSolvesReached) {
		t.Fatalf("Team 4: expected ErrMaxSolvesReached, got %v", err)
	}

	// Step 4: Team 5 also rejected
	_, err = svc.SubmitFlag(teams[4], 11, "PANTHEON{campus_riddle_11}")
	if !errors.Is(err, scoring.ErrMaxSolvesReached) {
		t.Fatalf("Team 5: expected ErrMaxSolvesReached, got %v", err)
	}

	// Step 5: Test unlimited round 12 allows all 5 teams to solve
	svc2 := scoring.NewService(db, rc, scoring.NewFlagValidator(256, 6000, 0), 5*time.Minute)
	for i := 0; i < 5; i++ {
		res, err := svc2.SubmitFlag(teams[i], 12, "PANTHEON{unlimited_round_12}")
		if err != nil {
			t.Fatalf("Team %d solve round 12 failed: %v", i+1, err)
		}
		if !res.Correct || res.PointsAwarded != 200 {
			t.Fatalf("Team %d solve round 12: expected correct=true, points=200, got %v, %v", i+1, res.Correct, res.PointsAwarded)
		}
	}
}
