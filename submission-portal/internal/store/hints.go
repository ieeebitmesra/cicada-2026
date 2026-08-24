package store

import (
	"ieee-ctf/internal/models"
)

// RecordHintUsage logs a dispensed hint with its PGP proof and cost.
func (db *DB) RecordHintUsage(teamID int64, roundID int, hintType string, hintIndex int, pgpProof string, cost float64) (int64, error) {
	res, err := db.Exec(`
		INSERT INTO hint_usage (team_id, round_id, hint_type, hint_index, pgp_proof, cost_points)
		VALUES (?, ?, ?, ?, ?, ?)`,
		teamID, roundID, hintType, hintIndex, pgpProof, cost)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// CountHintsUsed returns how many hints of a type a team has used on a round.
func (db *DB) CountHintsUsed(teamID int64, roundID int, hintType string) (int, error) {
	var n int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM hint_usage WHERE team_id = ? AND round_id = ? AND hint_type = ?`,
		teamID, roundID, hintType).Scan(&n)
	return n, err
}

// TeamHints returns all hints used by a team, oldest first.
func (db *DB) TeamHints(teamID int64) ([]models.HintUsage, error) {
	rows, err := db.Query(`
		SELECT id, team_id, round_id, hint_type, hint_index, pgp_proof, cost_points, used_at
		FROM hint_usage WHERE team_id = ? ORDER BY used_at, id`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.HintUsage
	for rows.Next() {
		h := &models.HintUsage{}
		if err := rows.Scan(&h.ID, &h.TeamID, &h.RoundID, &h.HintType, &h.HintIndex,
			&h.PGPProof, &h.CostPoints, &h.UsedAt); err != nil {
			return nil, err
		}
		out = append(out, *h)
	}
	return out, rows.Err()
}

// AllHints returns every hint usage across teams (admin).
func (db *DB) AllHints() ([]models.HintUsage, error) {
	rows, err := db.Query(`
		SELECT id, team_id, round_id, hint_type, hint_index, pgp_proof, cost_points, used_at
		FROM hint_usage ORDER BY used_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.HintUsage
	for rows.Next() {
		h := &models.HintUsage{}
		if err := rows.Scan(&h.ID, &h.TeamID, &h.RoundID, &h.HintType, &h.HintIndex,
			&h.PGPProof, &h.CostPoints, &h.UsedAt); err != nil {
			return nil, err
		}
		out = append(out, *h)
	}
	return out, rows.Err()
}


