package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestFormatPoints(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0.0, "0"},
		{100.0, "100"},
		{125.5, "125.5"},
		{-50.0, "-50"},
		{-12.3, "-12.3"},
	}
	for _, tc := range tests {
		got := FormatPoints(tc.input)
		if got != tc.want {
			t.Errorf("FormatPoints(%v) = %q; want %q", tc.input, got, tc.want)
		}
	}
}

func TestRenderBadge(t *testing.T) {
	badge := RenderBadge("ACTIVE", lipgloss.Color("#0B0F19"), lipgloss.Color("#10B981"))
	if !strings.Contains(badge, "ACTIVE") {
		t.Errorf("RenderBadge output missing label, got %q", badge)
	}
}

func TestRenderProgressBar(t *testing.T) {
	bar := RenderProgressBar(3, 6, 10)
	if !strings.Contains(bar, "50%") {
		t.Errorf("RenderProgressBar expected 50%%, got %q", bar)
	}

	barZero := RenderProgressBar(0, 0, 10)
	if !strings.Contains(barZero, "0%") {
		t.Errorf("RenderProgressBar with zero total expected 0%%, got %q", barZero)
	}

	barFull := RenderProgressBar(10, 5, 10)
	if !strings.Contains(barFull, "100%") {
		t.Errorf("RenderProgressBar capped at 100%%, got %q", barFull)
	}
}

func TestRenderStatCard(t *testing.T) {
	card := RenderStatCard("Score", "1,250 PTS", "Net Total", 24, ColorSuccess)
	if !strings.Contains(card, "SCORE") || !strings.Contains(card, "1,250 PTS") {
		t.Errorf("RenderStatCard output missing text, got %q", card)
	}
}
