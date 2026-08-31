// Package scoring implements CTF scoring rules and the game operations
// (submit flag, request hint, skip round).
package scoring

import "ieee-ctf/internal/models"

// CalculateTeamScore totals earned round points minus hint and skip costs.
func CalculateTeamScore(
	solved []models.Submission,
	rounds map[int]models.Round,
	hints []models.HintUsage,
	skips []models.Skip,
) float64 {
	var score float64
	for _, s := range solved {
		if s.IsCorrect {
			score += float64(rounds[s.RoundID].Points)
		}
	}
	for _, h := range hints {
		score -= h.CostPoints
	}
	for _, sk := range skips {
		score -= sk.CostPoints
	}
	return score
}

// HintCost returns the cost of one hint:
//   - plain   = 20% of the round's points
//   - encoded = 10% of the round's points
func HintCost(roundPoints int, hintType string) float64 {
	switch hintType {
	case "plain":
		return float64(roundPoints) * 0.20
	case "encoded":
		return float64(roundPoints) * 0.10
	}
	return 0
}

// SkipCost calculates penalty based on Round Points (50% of the challenge value).
// Basing on round value guarantees non-zero penalty.
func SkipCost(roundPoints int) float64 {
	return float64(roundPoints) * 0.50
}
