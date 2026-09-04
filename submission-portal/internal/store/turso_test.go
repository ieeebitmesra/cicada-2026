package store

import (
	"os"
	"testing"
)

func TestTursoIntegration(t *testing.T) {
	url := os.Getenv("TURSO_DATABASE_URL")
	if url == "" {
		url = "libsql://ctf-testifywebdev.aws-ap-south-1.turso.io"
	}
	token := os.Getenv("TURSO_AUTH_TOKEN")
	if token == "" {
		token = os.Getenv("CTF_DB_AUTH_TOKEN")
	}
	if token == "" {
		t.Skip("skipping Turso integration test: TURSO_AUTH_TOKEN or CTF_DB_AUTH_TOKEN not set")
	}

	db, err := OpenWithAuth(url, token)
	if err != nil {
		t.Fatalf("OpenWithAuth failed: %v", err)
	}
	defer db.Close()

	if !db.IsRemote() {
		t.Errorf("expected isRemote = true, got false")
	}

	if err := db.Migrate("../../migrations"); err != nil {
		t.Fatalf("Migrate failed on Turso: %v", err)
	}
	t.Logf("Turso migrations successfully applied!")
}
