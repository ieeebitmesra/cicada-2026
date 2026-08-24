package middleware

import (
	"strings"
	"testing"
)

func TestStripANSI(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain", "plain"},
		{"\x1b[31mred\x1b[0m", "red"},
		{"\x1b]0;title\x07rest", "rest"},                 // OSC
		{"\x1bP+q544e\x1b\\after", "after"},              // DCS
		{"\x1b[2J\x1b[Hclear", "clear"},                  // CSI
		{"\x1b[?25lhidden-cursor", "hidden-cursor"},      // extended CSI
	}
	for _, c := range cases {
		if got := StripANSI(c.in); got != c.want {
			t.Errorf("StripANSI(%q) = %q want %q", c.in, got, c.want)
		}
	}
}

func TestSanitizeInput(t *testing.T) {
	in := "ok\x00\x01\x02text\nkeep\ttabs\r"
	want := "oktext\nkeep\ttabs\r"
	if got := SanitizeInput(in); got != want {
		t.Errorf("SanitizeInput = %q want %q", got, want)
	}
}

func TestValidateTextInput(t *testing.T) {
	if s, ok := ValidateTextInput("hello \x1b[31mworld", 64); !ok || s != "hello world" {
		t.Errorf("valid input rejected: %q %v", s, ok)
	}
	if _, ok := ValidateTextInput(strings.Repeat("x", 300), 64); ok {
		t.Error("over-length input accepted")
	}
	for _, bad := range []string{"`id`", "$(cmd)", "${var}", "a;b", "a|b", "a>b", "a<b"} {
		if _, ok := ValidateTextInput(bad, 64); ok {
			t.Errorf("shell metachar accepted: %q", bad)
		}
	}
}
