// Package auth implements SSH password verification and PGP-signed hint proofs.
package auth

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"

	"ieee-ctf/internal/models"
	"ieee-ctf/internal/store"
)

var (
	authLimiterMu sync.Mutex
	authLimiters  = make(map[string]*rate.Limiter)
	failedLogins  = make(map[string]int)
	lockoutUntil  = make(map[string]time.Time)
)

const bcryptCost = 10

// HashPassword hashes a plaintext password with bcrypt.
func HashPassword(plain string) (string, error) {
	if len(plain) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt: %w", err)
	}
	return string(h), nil
}

// VerifyTeamLogin checks an SSH username/password pair against the teams table.
func VerifyTeamLogin(db *store.DB, user, password string) bool {
	return VerifyTeamLoginWithIP(db, user, password, "127.0.0.1")
}

// VerifyTeamLoginWithIP checks SSH login with per-IP rate limiting and failure lockout.
func VerifyTeamLoginWithIP(db *store.DB, user, password, remoteAddr string) bool {
	if user == "" || password == "" {
		return false
	}
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		ip = remoteAddr
	}

	authLimiterMu.Lock()
	if until, locked := lockoutUntil[ip]; locked && time.Now().Before(until) {
		authLimiterMu.Unlock()
		return false
	}
	lim, exists := authLimiters[ip]
	if !exists {
		lim = rate.NewLimiter(rate.Limit(2.0), 5) // max 2 auth/sec, burst 5
		authLimiters[ip] = lim
	}
	if !lim.Allow() {
		authLimiterMu.Unlock()
		return false
	}
	authLimiterMu.Unlock()

	team, err := db.GetTeamBySSHUser(strings.ToLower(user))
	if err != nil || team == nil {
		recordAuthFailure(ip)
		return false
	}

	if err := bcrypt.CompareHashAndPassword([]byte(team.SSHPass), []byte(password)); err != nil {
		recordAuthFailure(ip)
		return false
	}

	recordAuthSuccess(ip)
	return true
}

func recordAuthFailure(ip string) {
	authLimiterMu.Lock()
	defer authLimiterMu.Unlock()
	failedLogins[ip]++
	if failedLogins[ip] >= 10 {
		lockoutUntil[ip] = time.Now().Add(5 * time.Minute)
		failedLogins[ip] = 0
	}
}

func recordAuthSuccess(ip string) {
	authLimiterMu.Lock()
	defer authLimiterMu.Unlock()
	delete(failedLogins, ip)
	delete(lockoutUntil, ip)
}

var ErrWeakPassword = errors.New("password must be at least 8 characters")

// CompleteRegistration sets the team's new password and marks it registered.
// PGP key is no longer required for registration.
func CompleteRegistration(db *store.DB, team *models.Team, newPassword string) error {
	hash, err := HashPassword(newPassword)
	if err != nil {
		return ErrWeakPassword
	}
	return db.CompleteRegistration(team.ID, hash, "")
}

// ResetPassword generates and sets a new random password, returning the plaintext.
func ResetPassword(db *store.DB, teamID int64) (string, error) {
	const charset = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789"
	buf := make([]byte, 16)
	for i := range buf {
		// crypto/rand-free fallback is NOT acceptable; use rand below.
		buf[i] = charset[cryptRandIntn(len(charset))]
	}
	pass := string(buf)
	hash, err := HashPassword(pass)
	if err != nil {
		return "", err
	}
	if err := db.UpdatePassword(teamID, hash); err != nil {
		return "", err
	}
	return pass, nil
}
