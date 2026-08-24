package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"ieee-ctf/internal/models"
)

// CreateTeam inserts a new team with a bcrypt password hash.
func (db *DB) CreateTeam(name, sshUser, passHash string) (int64, error) {
	res, err := db.Exec(`INSERT INTO teams (name, ssh_user, ssh_pass) VALUES (?, ?, ?)`,
		name, sshUser, passHash)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return 0, fmt.Errorf("team name or ssh user already exists")
		}
		return 0, err
	}
	return res.LastInsertId()
}

// GetTeamBySSHUser fetches a team by its SSH username.
func (db *DB) GetTeamBySSHUser(user string) (*models.Team, error) {
	return db.scanTeam(db.QueryRow(`
		SELECT id, name, ssh_user, ssh_pass, pgp_pubkey, registered, created_at
		FROM teams WHERE ssh_user = ?`, user))
}

// GetTeamByName fetches a team by display name.
func (db *DB) GetTeamByName(name string) (*models.Team, error) {
	return db.scanTeam(db.QueryRow(`
		SELECT id, name, ssh_user, ssh_pass, pgp_pubkey, registered, created_at
		FROM teams WHERE name = ?`, name))
}

// GetTeam fetches a team by ID.
func (db *DB) GetTeam(id int64) (*models.Team, error) {
	return db.scanTeam(db.QueryRow(`
		SELECT id, name, ssh_user, ssh_pass, pgp_pubkey, registered, created_at
		FROM teams WHERE id = ?`, id))
}

// ListTeams returns all teams ordered by name.
func (db *DB) ListTeams() ([]models.Team, error) {
	rows, err := db.Query(`
		SELECT id, name, ssh_user, ssh_pass, pgp_pubkey, registered, created_at
		FROM teams ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Team
	for rows.Next() {
		t, err := scanTeamRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// UpdatePassword replaces a team's password hash.
func (db *DB) UpdatePassword(teamID int64, passHash string) error {
	_, err := db.Exec(`UPDATE teams SET ssh_pass = ? WHERE id = ?`, passHash, teamID)
	return err
}

// CompleteRegistration sets the new password hash + PGP pubkey and marks registered.
func (db *DB) CompleteRegistration(teamID int64, newPassHash, armoredPubkey string) error {
	_, err := db.Exec(`UPDATE teams SET ssh_pass = ?, pgp_pubkey = ?, registered = 1 WHERE id = ?`,
		newPassHash, armoredPubkey, teamID)
	return err
}

// DeleteTeam removes a team and its dependent records.
func (db *DB) DeleteTeam(teamID int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for _, stmt := range []string{
		`DELETE FROM hint_usage WHERE team_id = ?`,
		`DELETE FROM skips      WHERE team_id = ?`,
		`DELETE FROM submissions WHERE team_id = ?`,
		`DELETE FROM teams       WHERE id = ?`,
	} {
		if _, err := tx.Exec(stmt, teamID); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) scanTeam(row *sql.Row) (*models.Team, error) {
	t := &models.Team{}
	var registered bool
	var pubkey string
	if err := row.Scan(&t.ID, &t.Name, &t.SSHUser, &t.SSHPass, &pubkey, &registered, &t.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	t.PGPPubkey = pubkey
	t.Registered = registered
	return t, nil
}

func scanTeamRows(rows *sql.Rows) (*models.Team, error) {
	t := &models.Team{}
	var registered bool
	var pubkey string
	if err := rows.Scan(&t.ID, &t.Name, &t.SSHUser, &t.SSHPass, &pubkey, &registered, &t.CreatedAt); err != nil {
		return nil, err
	}
	t.PGPPubkey = pubkey
	t.Registered = registered
	return t, nil
}
