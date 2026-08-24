// Package auth implements SSH password verification and PGP-signed hint proofs.
package auth

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"ieee-ctf/internal/models"
	"ieee-ctf/internal/store"
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
	if user == "" || password == "" {
		return false
	}
	team, err := db.GetTeamBySSHUser(strings.ToLower(user))
	if err != nil || team == nil {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(team.SSHPass), []byte(password)) == nil
}

var ErrWeakPassword = errors.New("password must be at least 8 characters")

// CompleteRegistration sets the team's new password + PGP key and marks it registered.
func CompleteRegistration(db *store.DB, team *models.Team, newPassword, armoredPubkey string) error {
	hash, err := HashPassword(newPassword)
	if err != nil {
		return ErrWeakPassword
	}
	if _, err := ParsePublicKey(armoredPubkey); err != nil {
		return fmt.Errorf("invalid PGP public key: %w", err)
	}
	return db.CompleteRegistration(team.ID, hash, strings.TrimSpace(armoredPubkey))
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
