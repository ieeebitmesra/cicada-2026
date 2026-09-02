package scoring

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ieee-ctf/internal/models"
)

// Hint challenge format (one key=value per line so players can sign it verbatim):
//
//	HINT-REQ
//	round=3
//	type=plain
//	index=0
//	ts=2026-08-24T10:30:00Z
//	team=alpha
//
// The player must PGP-clearsign this exact message. The signature proves intent
// (non-repudiation) and the timestamp prevents replaying old challenges.

var (
	ErrNoHintsLeft   = errors.New("no hints of that type remain for this round")
	ErrBadChallenge  = errors.New("challenge does not match the issued request")
	ErrChallengeStale = errors.New("challenge timestamp outside validity window")
)

// BuildHintChallenge creates the string the player must clearsign.
func BuildHintChallenge(team *models.Team, roundID int, hintType string, hintIndex int, nonce string, now time.Time) string {
	return fmt.Sprintf("HINT-REQ\nround=%d\ntype=%s\nindex=%d\nnonce=%s\nts=%s\nteam=%s",
		roundID, hintType, hintIndex, nonce, now.UTC().Format(time.RFC3339), team.SSHUser)
}

// VerifyHintChallenge validates a signed challenge body against expectations.
// body is the plaintext recovered from the clearsigned message.
func VerifyHintChallenge(body string, team *models.Team, roundID int, hintType string,
	hintIndex int, validity time.Duration) error {

	fields := ParseChallenge(body)
	if fields == nil {
		return ErrBadChallenge
	}
	if fields["team"] != team.SSHUser {
		return ErrBadChallenge
	}
	if fields["round"] != strconv.Itoa(roundID) ||
		fields["type"] != hintType ||
		fields["index"] != strconv.Itoa(hintIndex) {
		return ErrBadChallenge
	}
	if fields["nonce"] == "" {
		return ErrBadChallenge
	}
	ts, err := time.Parse(time.RFC3339, fields["ts"])
	if err != nil {
		return ErrBadChallenge
	}
	if d := time.Since(ts); d > validity || d < -validity {
		return ErrChallengeStale
	}
	return nil
}

// BuildSkipChallenge creates the string the player must clearsign to bypass a round.
func BuildSkipChallenge(team *models.Team, roundID int, nonce string, now time.Time) string {
	return fmt.Sprintf("SKIP-REQ\nround=%d\nnonce=%s\nts=%s\nteam=%s",
		roundID, nonce, now.UTC().Format(time.RFC3339), team.SSHUser)
}

// VerifySkipChallenge validates a signed skip challenge body against expectations.
func VerifySkipChallenge(body string, team *models.Team, roundID int, validity time.Duration) error {
	fields := ParseSkipChallenge(body)
	if fields == nil {
		return ErrBadChallenge
	}
	if fields["team"] != team.SSHUser {
		return ErrBadChallenge
	}
	if fields["round"] != strconv.Itoa(roundID) {
		return ErrBadChallenge
	}
	if fields["nonce"] == "" {
		return ErrBadChallenge
	}
	ts, err := time.Parse(time.RFC3339, fields["ts"])
	if err != nil {
		return ErrBadChallenge
	}
	if d := time.Since(ts); d > validity || d < -validity {
		return ErrChallengeStale
	}
	return nil
}

// ParseChallenge parses "k=v" lines into a map; returns nil on malformed input.
func ParseChallenge(body string) map[string]string {
	out := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "HINT-REQ" { // marker line
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok || k == "" || v == "" {
			return nil
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	if len(out) < 5 { // round, type, index, nonce, ts, team — marker excluded
		return nil
	}
	return out
}

// ParseSkipChallenge parses "k=v" lines into a map; returns nil on malformed input.
func ParseSkipChallenge(body string) map[string]string {
	out := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "SKIP-REQ" { // marker line
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok || k == "" || v == "" {
			return nil
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	if len(out) < 4 { // round, nonce, ts, team
		return nil
	}
	return out
}
