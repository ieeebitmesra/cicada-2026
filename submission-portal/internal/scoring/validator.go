package scoring

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"ieee-ctf/internal/middleware"

	"golang.org/x/time/rate"
)

var (
	ErrFlagFormat     = errors.New("invalid flag format (expected IEEE{...})")
	ErrFlagTooLong    = errors.New("flag too long")
	ErrRateLimited    = errors.New("too many submissions — slow down")
	ErrCooldownActive = errors.New("cooldown active between submissions")
)

// flagPattern: flags look like IEEE{...} with printable ASCII inside.
// Braces are required by the format and therefore exempt from the generic
// shell-metacharacter filter applied to other free-text inputs.
var flagPattern = regexp.MustCompile(`^IEEE\{[!-~]{4,240}}$`)

// FlagValidator enforces flag format + per-team submission rate limits.
type FlagValidator struct {
	maxLen        int
	perMinute     rate.Limit
	burst         int
	cooldown      time.Duration
	mu            sync.Mutex
	limiters      map[int64]*rate.Limiter
	lastAttempt   map[int64]time.Time
}

// NewFlagValidator builds a validator from server config values.
func NewFlagValidator(maxLen int, perMinutePerTeam float64, cooldown time.Duration) *FlagValidator {
	return &FlagValidator{
		maxLen:      maxLen,
		perMinute:   rate.Limit(perMinutePerTeam / 60.0),
		burst:       1,
		cooldown:    cooldown,
		limiters:    make(map[int64]*rate.Limiter),
		lastAttempt: make(map[int64]time.Time),
	}
}

// HashFlag returns the SHA-256 hex digest of the exact flag string.
func HashFlag(flag string) string {
	sum := sha256.Sum256([]byte(flag))
	return hex.EncodeToString(sum[:])
}

// Check validates + rate limits a submission attempt for a team.
// It returns the normalized flag on success.
func (v *FlagValidator) Check(teamID int64, raw string) (string, error) {
	flag := strings.TrimSpace(middleware.StripANSI(raw))

	if len(flag) > v.maxLen {
		return "", ErrFlagTooLong
	}
	if !flagPattern.MatchString(flag) {
		return "", ErrFlagFormat
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	lim := v.limiters[teamID]
	if lim == nil {
		lim = rate.NewLimiter(v.perMinute, v.burst)
		v.limiters[teamID] = lim
	}
	if !lim.Allow() {
		return "", ErrRateLimited
	}
	if last, ok := v.lastAttempt[teamID]; ok {
		if elapsed := time.Since(last); elapsed < v.cooldown {
			return "", fmt.Errorf("%w (%ds remaining)",
				ErrCooldownActive, int((v.cooldown - elapsed).Seconds()+0.99))
		}
	}
	v.lastAttempt[teamID] = time.Now()
	return flag, nil
}
