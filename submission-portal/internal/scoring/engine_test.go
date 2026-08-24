package scoring

import (
	"testing"

	"ieee-ctf/internal/models"
)

func TestHintCost(t *testing.T) {
	cases := []struct {
		roundPoints int
		typ         string
		want        float64
	}{
		{100, "plain", 20},
		{100, "encoded", 10},
		{300, "plain", 60},
		{300, "encoded", 30},
		{100, "bogus", 0},
	}
	for _, c := range cases {
		if got := HintCost(c.roundPoints, c.typ); got != c.want {
			t.Errorf("HintCost(%d,%q) = %v want %v", c.roundPoints, c.typ, got, c.want)
		}
	}
}

func TestSkipCost(t *testing.T) {
	if got := SkipCost(200); got != 100 {
		t.Errorf("SkipCost(200) = %v want 100", got)
	}
	if got := SkipCost(-40); got != 20 {
		t.Errorf("SkipCost(-40) = %v want 20 (absolute)", got)
	}
	if got := SkipCost(0); got != 0 {
		t.Errorf("SkipCost(0) = %v want 0", got)
	}
}

func TestCalculateTeamScore(t *testing.T) {
	rounds := map[int]models.Round{
		1: {ID: 1, Points: 100},
		2: {ID: 2, Points: 150},
	}
	solved := []models.Submission{
		{RoundID: 1, IsCorrect: true},
		{RoundID: 2, IsCorrect: true},
		{RoundID: 3, IsCorrect: false}, // ignored
	}
	hints := []models.HintUsage{
		{CostPoints: 20}, // plain on r1
		{CostPoints: 15}, // encoded on r2
	}
	skips := []models.Skip{{CostPoints: 50}}

	// earned = 100+150 ; costs = 20+15+50 ; total = 165
	want := 250 - 85
	if got := CalculateTeamScore(solved, rounds, hints, skips); got != float64(want) {
		t.Errorf("score = %v want %v", got, want)
	}

	// Unbounded negative: heavy penalties exceed earnings.
	negative := CalculateTeamScore(
		[]models.Submission{{RoundID: 1, IsCorrect: true}},
		rounds,
		[]models.HintUsage{},
		[]models.Skip{{CostPoints: 500}},
	)
	if negative != -400 {
		t.Errorf("negative floor score = %v want -400", negative)
	}
}
