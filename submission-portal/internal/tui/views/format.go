package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// formatPoints renders a signed point value.
func formatPoints(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%d", int64(v))
	}
	return fmt.Sprintf("%.1f", v)
}

// StripANSI removes ANSI escape codes from a string for accurate measuring.
func StripANSI(s string) string {
	var b strings.Builder
	inAnsi := false
	for _, ch := range s {
		if ch == '\033' {
			inAnsi = true
			continue
		}
		if inAnsi {
			if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
				inAnsi = false
			}
			continue
		}
		b.WriteRune(ch)
	}
	return b.String()
}

// VisualWidth measures the true terminal display width, stripping ANSI and emoji variation selectors.
func VisualWidth(s string) int {
	stripped := StripANSI(s)
	stripped = strings.ReplaceAll(stripped, "\uFE0F", "")
	stripped = strings.ReplaceAll(stripped, "\uFE0E", "")
	return lipgloss.Width(stripped)
}

// SingleLine collapses multi-line text into a single space-separated line.
func SingleLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

// Truncate ensures text never auto-wraps inside bordered panels and stays strictly on a single line (Golden Rule #2).
func Truncate(s string, maxLen int) string {
	s = SingleLine(s)
	if maxLen < 1 {
		return ""
	}
	w := VisualWidth(s)
	if w <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return strings.Repeat(".", maxLen)
	}

	runes := []rune(s)
	for i := len(runes); i > 0; i-- {
		candidate := string(runes[:i]) + "…"
		if VisualWidth(candidate) <= maxLen {
			return candidate
		}
	}
	return "…"
}

