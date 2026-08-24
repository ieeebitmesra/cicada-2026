package scoring

import (
	"strings"
	"testing"
	"time"

	"ieee-ctf/internal/models"
)

const validFlagHash = "" // set in TestHashFlag

func TestHashFlag(t *testing.T) {
	a := HashFlag("IEEE{test_flag}")
	b := HashFlag("IEEE{test_flag}")
	c := HashFlag("IEEE{other}")
	if a == "" || len(a) != 64 {
		t.Fatalf("hash length wrong: %q", a)
	}
	if a != b {
		t.Error("hash not deterministic")
	}
	if a == c {
		t.Error("different flags produced same hash")
	}
}

func TestValidatorFormatAndRateLimit(t *testing.T) {
	v := NewFlagValidator(256, 2, 0) // no cooldown for the test

	if _, err := v.Check(1, "not-a-flag"); err != ErrFlagFormat {
		t.Errorf("expected ErrFlagFormat, got %v", err)
	}
	if _, err := v.Check(1, "IEEE{ok_flag}"); err != nil {
		t.Errorf("valid flag rejected: %v", err)
	}
	long := "IEEE{" + strings.Repeat("a", 300) + "}"
	if _, err := v.Check(1, long); err != ErrFlagTooLong {
		t.Errorf("expected ErrFlagTooLong, got %v", err)
	}

	// burst=1 → the second immediate attempt must be limited.
	v2 := NewFlagValidator(256, 2, 0)
	if _, err := v2.Check(7, "IEEE{one1}"); err != nil {
		t.Fatal(err)
	}
	if _, err := v2.Check(7, "IEEE{two2}"); err != ErrRateLimited {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
	// other teams unaffected
	if _, err := v2.Check(8, "IEEE{three}"); err != nil {
		t.Errorf("rate limiter leaked across teams: %v", err)
	}
}

func TestChallengeParseAndVerify(t *testing.T) {
	team := &models.Team{SSHUser: "alpha"}

	now := time.Now().UTC()
	ch := BuildHintChallenge(team, 3, "plain", 0, now)
	fields := ParseChallenge(ch)
	if fields["round"] != "3" || fields["type"] != "plain" ||
		fields["index"] != "0" || fields["team"] != "alpha" || fields["ts"] == "" {
		t.Fatalf("challenge parse failed: %#v", fields)
	}

	if err := VerifyHintChallenge(ch, team, 3, "plain", 0, time.Minute); err != nil {
		t.Errorf("fresh challenge rejected: %v", err)
	}

	// Wrong hint type/index/team rejected.
	if err := VerifyHintChallenge(ch, team, 3, "encoded", 0, time.Minute); err == nil {
		t.Error("mismatched type accepted")
	}
	if err := VerifyHintChallenge(ch, team, 3, "plain", 1, time.Minute); err == nil {
		t.Error("mismatched index accepted")
	}
	if other := (&models.Team{SSHUser: "beta"}); VerifyHintChallenge(ch, other, 3, "plain", 0, time.Minute) == nil {
		t.Error("wrong team accepted")
	}

	// Stale challenge rejected.
	old := BuildHintChallenge(team, 3, "plain", 0, now.Add(-11*time.Minute))
	if err := VerifyHintChallenge(old, team, 3, "plain", 0, 10*time.Minute); err != ErrChallengeStale {
		t.Errorf("stale challenge accepted: %v", err)
	}

	if f := ParseChallenge("garbage without equals"); f != nil {
		t.Error("malformed challenge parsed")
	}
}
