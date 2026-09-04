package scoring_test

import (
	"path/filepath"
	"testing"
	"time"

	"ieee-ctf/internal/models"
	"ieee-ctf/internal/scoring"
	"ieee-ctf/internal/store"
)

func TestHintsAndSkipsRestrictionConfig(t *testing.T) {
	yamlData := []byte(`
allow_hints: false
allow_skips: false

rounds:
  - id: 1
    name: "Round 1 — Warmup"
    points: 100
    is_active: true
    allow_hints: false
    allow_skips: false
    flag_hash: "ef2d127de37b942baad06145e54b0c619a1f22327b2ebbcfbec78f5564afe39d"
    hints:
      - type: plain
        text: "Sample plain hint"
      - type: encoded
        text: "U2FtcGxl"

  - id: 2
    name: "Round 2 — Active"
    points: 200
    is_active: true
    allow_hints: true
    allow_skips: true
    flag_hash: "ef2d127de37b942baad06145e54b0c619a1f22327b2ebbcfbec78f5564afe39d"
    hints:
      - type: plain
        text: "Round 2 hint"
`)

	rc, err := models.ParseRounds(yamlData)
	if err != nil {
		t.Fatalf("ParseRounds failed: %v", err)
	}

	if rc.HintsAllowed(0) {
		t.Errorf("expected global HintsAllowed(0) to be false")
	}
	if rc.SkipsAllowed(0) {
		t.Errorf("expected global SkipsAllowed(0) to be false")
	}

	// Setup DB & Service
	dbPath := filepath.Join(t.TempDir(), "test.db")
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

	teamID, err := db.CreateTeam("Alpha", "alpha", "hashed")
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	team, err := db.GetTeam(teamID)
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}

	flags := scoring.NewFlagValidator(256, 10, 0)
	svc := scoring.NewService(db, rc, flags, 5*time.Minute)

	// Test 1: Requesting hint on restricted round returns ErrHintsRestricted
	_, _, _, err = svc.HintChallenge(team, 1, "plain")
	if err != scoring.ErrHintsRestricted {
		t.Errorf("expected ErrHintsRestricted, got %v", err)
	}

	_, _, err = svc.DirectRedeemHint(team, 1, "plain")
	if err != scoring.ErrHintsRestricted {
		t.Errorf("expected ErrHintsRestricted on DirectRedeemHint, got %v", err)
	}

	// Test 2: Requesting skip on restricted round returns ErrSkipsRestricted
	_, _, err = svc.SkipChallenge(team, 1)
	if err != scoring.ErrSkipsRestricted {
		t.Errorf("expected ErrSkipsRestricted on SkipChallenge, got %v", err)
	}

	_, err = svc.SkipRound(team, 1)
	if err != scoring.ErrSkipsRestricted {
		t.Errorf("expected ErrSkipsRestricted on SkipRound, got %v", err)
	}

	_, err = svc.PreviewSkipCost(team.ID, 1)
	if err != scoring.ErrSkipsRestricted {
		t.Errorf("expected ErrSkipsRestricted on PreviewSkipCost, got %v", err)
	}
}
