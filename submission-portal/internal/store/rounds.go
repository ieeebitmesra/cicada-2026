package store

import (
	"fmt"

	"ieee-ctf/internal/models"
)

// UpsertRounds syncs round definitions from rounds.yaml into the DB.
// Hints themselves live only in config; the DB stores round metadata.
func (db *DB) UpsertRounds(defs []models.RoundDef) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for i, d := range defs {
		active := 1
		if !d.IsActive {
			active = 0
		}
		_, err := tx.Exec(`
			INSERT INTO rounds (id, name, points, flag_hash, is_active, sort_order)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				name = excluded.name,
				points = excluded.points,
				flag_hash = excluded.flag_hash,
				is_active = excluded.is_active,
				sort_order = excluded.sort_order`,
			d.ID, d.Name, d.Points, d.FlagHash, active, i)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("upsert round %d: %w", d.ID, err)
		}
	}
	return tx.Commit()
}

// GetRound fetches one round by ID.
func (db *DB) GetRound(id int) (*models.Round, error) {
	r := &models.Round{}
	err := db.QueryRow(
		`SELECT id, name, points, flag_hash, is_active, sort_order FROM rounds WHERE id = ?`, id).
		Scan(&r.ID, &r.Name, &r.Points, &r.FlagHash, &r.IsActive, &r.SortOrder)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// ListRounds returns rounds ordered by sort_order. When activeOnly is true,
// only active rounds are returned.
func (db *DB) ListRounds(activeOnly bool) ([]models.Round, error) {
	q := `SELECT id, name, points, flag_hash, is_active, sort_order FROM rounds`
	if activeOnly {
		q += ` WHERE is_active = 1`
	}
	q += ` ORDER BY sort_order, id`
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Round
	for rows.Next() {
		var r models.Round
		var active bool
		if err := rows.Scan(&r.ID, &r.Name, &r.Points, &r.FlagHash, &active, &r.SortOrder); err != nil {
			return nil, err
		}
		r.IsActive = active
		out = append(out, r)
	}
	return out, rows.Err()
}

// SetRoundActive toggles a round's availability.
func (db *DB) SetRoundActive(id int, active bool) error {
	v := 0
	if active {
		v = 1
	}
	res, err := db.Exec(`UPDATE rounds SET is_active = ? WHERE id = ?`, v, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// RoundsMap returns all rounds keyed by ID.
func (db *DB) RoundsMap() (map[int]models.Round, error) {
	list, err := db.ListRounds(false)
	if err != nil {
		return nil, err
	}
	m := make(map[int]models.Round, len(list))
	for _, r := range list {
		m[r.ID] = r
	}
	return m, nil
}

// RecordSkip inserts a skip row for a team/round with its penalty cost.
func (db *DB) RecordSkip(teamID int64, roundID int, cost float64) (int64, error) {
	res, err := db.Exec(`INSERT INTO skips (team_id, round_id, cost_points) VALUES (?, ?, ?)`,
		teamID, roundID, cost)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// HasSkipped reports whether a team already skipped a round.
func (db *DB) HasSkipped(teamID int64, roundID int) (bool, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM skips WHERE team_id = ? AND round_id = ?`,
		teamID, roundID).Scan(&n)
	return n > 0, err
}

// TeamSkips returns all skips for a team.
func (db *DB) TeamSkips(teamID int64) ([]models.Skip, error) {
	rows, err := db.Query(`
		SELECT id, team_id, round_id, cost_points, skipped_at
		FROM skips WHERE team_id = ? ORDER BY skipped_at, id`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Skip
	for rows.Next() {
		s := &models.Skip{}
		if err := rows.Scan(&s.ID, &s.TeamID, &s.RoundID, &s.CostPoints, &s.SkippedAt); err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

// Scoreboard reads the leaderboard view ordered by score desc.
func (db *DB) Scoreboard() ([]models.ScoreboardEntry, error) {
	rows, err := db.Query(
		`SELECT team_id, team_name, total_score, rounds_solved FROM scoreboard ORDER BY total_score DESC, last_solve_at ASC, team_name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.ScoreboardEntry
	for rows.Next() {
		var e models.ScoreboardEntry
		if err := rows.Scan(&e.TeamID, &e.TeamName, &e.TotalScore, &e.RoundsSolved); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
