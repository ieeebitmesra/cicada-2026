package store

import (
	"path/filepath"
	"testing"

	"ieee-ctf/internal/models"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Migrate("../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func mustCreateTeam(t *testing.T, db *DB, name string) *models.Team {
	t.Helper()
	id, err := db.CreateTeam(name, name+"-user", "$2a$10$abcdefghijklmnopqrstuv0123456789012345678901234567890")
	if err != nil {
		t.Fatal(err)
	}
	team, err := db.GetTeam(id)
	if err != nil {
		t.Fatal(err)
	}
	return team
}

func TestSubmissionDuplicateGuard(t *testing.T) {
	db := newTestDB(t)
	team := mustCreateTeam(t, db, "alpha")

	rounds := []models.RoundDef{
		{ID: 1, Name: "R1", Points: 100, FlagHash: "aa", IsActive: true},
	}
	if err := db.UpsertRounds(rounds); err != nil {
		t.Fatal(err)
	}

	if _, err := db.RecordSubmission(team.ID, 1, "IEEE{x}", true); err != nil {
		t.Fatalf("first correct submission: %v", err)
	}
	if _, err := db.RecordSubmission(team.ID, 1, "IEEE{x}", true); err != ErrAlreadySolved {
		t.Fatalf("duplicate correct allowed: %v", err)
	}

	solved, err := db.HasSolvedRound(team.ID, 1)
	if err != nil || !solved {
		t.Fatalf("HasSolvedRound = %v %v", solved, err)
	}
}

func TestScoreboardViewAndSkips(t *testing.T) {
	db := newTestDB(t)
	a := mustCreateTeam(t, db, "alpha")
	b := mustCreateTeam(t, db, "beta")

	if err := db.UpsertRounds([]models.RoundDef{
		{ID: 1, Name: "R1", Points: 100, FlagHash: "aa", IsActive: true},
		{ID: 2, Name: "R2", Points: 200, FlagHash: "bb", IsActive: true},
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := db.RecordSubmission(a.ID, 1, "f", true); err != nil {
		t.Fatal(err)
	}
	if _, err := db.RecordSubmission(a.ID, 2, "f", true); err != nil {
		t.Fatal(err)
	}
	if _, err := db.RecordHintUsage(a.ID, 1, "plain", 0, "proof", 20); err != nil {
		t.Fatal(err)
	}
	if _, err := db.RecordSkip(b.ID, 2, 0); err != nil {
		t.Fatal(err)
	}

	board, err := db.Scoreboard()
	if err != nil {
		t.Fatal(err)
	}
	if len(board) != 2 {
		t.Fatalf("scoreboard rows = %d want 2", len(board))
	}
	// alpha: 300 earned - 20 hint = 280 ; beta: 0 (skip cost 0 in this test row)
	if board[0].TeamName != "alpha" || board[0].TotalScore != 280 || board[0].RoundsSolved != 2 {
		t.Errorf("unexpected top entry: %+v", board[0])
	}

	skipped, err := db.HasSkipped(b.ID, 2)
	if err != nil || !skipped {
		t.Fatalf("HasSkipped = %v %v", skipped, err)
	}
	if again, err := db.RecordSkip(b.ID, 2, 5); err == nil {
		t.Fatalf("double skip allowed: %v", again)
	}
}

func TestRegistrationUpdate(t *testing.T) {
	db := newTestDB(t)
	team := mustCreateTeam(t, db, "gamma")

	if team.Registered {
		t.Fatal("new team should be unregistered")
	}
	if err := db.CompleteRegistration(team.ID, "newhash", "-----BEGIN PGP PUBLIC KEY BLOCK-----"); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetTeamBySSHUser("gamma-user")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Registered || got.PGPPubkey == "" || got.SSHPass != "newhash" {
		t.Errorf("registration not persisted: %+v", got)
	}
}
