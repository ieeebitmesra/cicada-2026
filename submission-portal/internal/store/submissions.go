package store

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"ieee-ctf/internal/models"
)

// ErrAlreadySolved is returned when a team re-submits a correct flag for a
// round it already solved.
var ErrAlreadySolved = errors.New("round already solved")

// RecordSubmission logs a flag attempt. Returns ErrAlreadySolved if the team
// already has a correct submission for the round (guarded by a partial unique
// index at the DB level as well).
func (db *DB) RecordSubmission(teamID int64, roundID int, flagInput string, isCorrect bool) (int64, error) {
	if isCorrect {
		var solved int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM submissions WHERE team_id = ? AND round_id = ? AND is_correct = 1`,
			teamID, roundID).Scan(&solved); err != nil {
			return 0, err
		}
		if solved > 0 {
			return 0, ErrAlreadySolved
		}
	}
	res, err := db.Exec(
		`INSERT INTO submissions (team_id, round_id, flag_input, is_correct) VALUES (?, ?, ?, ?)`,
		teamID, roundID, flagInput, isCorrect)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return 0, ErrAlreadySolved
		}
		return 0, err
	}
	return res.LastInsertId()
}

// HasSolvedRound reports whether the team has a correct submission for the round.
func (db *DB) HasSolvedRound(teamID int64, roundID int) (bool, error) {
	var n int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM submissions WHERE team_id = ? AND round_id = ? AND is_correct = 1`,
		teamID, roundID).Scan(&n)
	return n > 0, err
}

// SolvedRoundIDs returns the sorted list of round ids solved by the team.
func (db *DB) SolvedRoundIDs(teamID int64) ([]int, error) {
	rows, err := db.Query(
		`SELECT DISTINCT round_id FROM submissions WHERE team_id = ? AND is_correct = 1 ORDER BY round_id`,
		teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// SolvedSubmissions returns ALL correct submissions for a team without truncation.
func (db *DB) SolvedSubmissions(teamID int64) ([]models.Submission, error) {
	rows, err := db.Query(`
		SELECT id, team_id, round_id, flag_input, is_correct, submitted_at
		FROM submissions WHERE team_id = ? AND is_correct = 1 ORDER BY round_id`,
		teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Submission
	for rows.Next() {
		s, err := scanSubmission(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

// TeamSubmissions returns recent submissions for a team (newest first), capped.
func (db *DB) TeamSubmissions(teamID int64, limit int) ([]models.Submission, error) {
	rows, err := db.Query(`
		SELECT id, team_id, round_id, flag_input, is_correct, submitted_at
		FROM submissions WHERE team_id = ? ORDER BY submitted_at DESC, id DESC LIMIT ?`,
		teamID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Submission
	for rows.Next() {
		s, err := scanSubmission(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

// SubmissionsSince counts a team's submissions since t (rate limiting aid /
// auditing even though the live limiter is in-memory).
func (db *DB) SubmissionsSince(teamID int64, t time.Time) (int, error) {
	var n int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM submissions WHERE team_id = ? AND submitted_at >= ?`,
		teamID, t.UTC()).Scan(&n)
	return n, err
}

func scanSubmission(rows *sql.Rows) (*models.Submission, error) {
	s := &models.Submission{}
	if err := rows.Scan(&s.ID, &s.TeamID, &s.RoundID, &s.FlagInput, &s.IsCorrect, &s.SubmittedAt); err != nil {
		return nil, err
	}
	return s, nil
}
