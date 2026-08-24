package middleware

import (
	"regexp"
	"strings"
	"unicode"
)

// Strip ALL ANSI/CSI/OSC/DCS escape sequences.
var ansiRegex = regexp.MustCompile(
	`\x1b\[[0-9;<=>?]*[@-~]` + // CSI sequences (incl. private modes like ?25l)
		`|\x1b\].*?\x07` + // OSC sequences (BEL terminated)
		`|\x1b\].*?\x1b\\` + // OSC sequences (ST terminated)
		`|\x1bP.*?\x1b\\`, // DCS sequences
)

// StripANSI removes terminal escape sequences from input.
func StripANSI(input string) string {
	return ansiRegex.ReplaceAllString(input, "")
}

// SanitizeInput removes control characters (keeping \n \r \t).
func SanitizeInput(input string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, input)
}

// ValidateTextInput strips ANSI + control chars, enforces maxLen and blocks
// shell metacharacters. Returns the sanitized string and whether it is valid.
func ValidateTextInput(input string, maxLen int) (string, bool) {
	sanitized := SanitizeInput(StripANSI(input))
	if len(sanitized) > maxLen {
		return "", false
	}
	// Block shell metacharacters.
	if strings.ContainsAny(sanitized, "`$(){}[]|;&><") {
		return "", false
	}
	return sanitized, true
}
