// Package models holds shared domain types for the IEEE CTF submission portal.
package models

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Team is a participating team (one SSH account per team).
type Team struct {
	ID         int64
	Name       string
	SSHUser    string
	SSHPass    string // bcrypt hash
	PGPPubkey  string // armored public key
	Registered bool
	CreatedAt  time.Time
}

// Round is one challenge round.
type Round struct {
	ID          int
	Name        string
	Description string
	SkipText    string
	Points      int
	FlagHash    string // SHA-256 hex of the exact flag
	IsActive    bool
	LimitSolves bool
	MaxSolves   int
	SortOrder   int
}

// Submission is a single flag submission attempt.
type Submission struct {
	ID          int64
	TeamID      int64
	RoundID     int
	FlagInput   string
	IsCorrect   bool
	SubmittedAt time.Time
}

// HintUsage records one dispensed hint with its PGP proof.
type HintUsage struct {
	ID         int64
	TeamID     int64
	RoundID    int
	HintType   string // 'plain' | 'encoded'
	HintIndex  int
	PGPProof   string
	CostPoints float64
	UsedAt     time.Time
}

// Skip records a skipped round and its penalty cost.
type Skip struct {
	ID         int64
	TeamID     int64
	RoundID    int
	CostPoints float64
	SkippedAt  time.Time
}

// ScoreboardEntry is one row of the leaderboard view.
type ScoreboardEntry struct {
	TeamID       int64
	TeamName     string
	TotalScore   float64
	RoundsSolved int
}

// HintDef is a hint as defined in rounds.yaml.
type HintDef struct {
	Type string `yaml:"type"`
	Text string `yaml:"text"`
}

// RoundDef is one round as defined in rounds.yaml.
type RoundDef struct {
	ID                 int       `yaml:"id"`
	Name               string    `yaml:"name"`
	Description        string    `yaml:"description,omitempty"`
	Text               string    `yaml:"text,omitempty"`
	SkipText           string    `yaml:"skip_text,omitempty"`
	SkipURL            string    `yaml:"skip_url,omitempty"`
	Points             int       `yaml:"points"`
	IsActive           bool      `yaml:"is_active"`
	FlagHash           string    `yaml:"flag_hash"`
	AllowHints         *bool     `yaml:"allow_hints,omitempty"`
	AllowSkips         *bool     `yaml:"allow_skips,omitempty"`
	LimitSolves        *bool     `yaml:"limit_solves,omitempty"`
	LimitedSubmissions *bool     `yaml:"limited_submissions,omitempty"`
	MaxSolves          int       `yaml:"max_solves,omitempty"`
	MaxSubmissions     int       `yaml:"max_submissions,omitempty"`
	Hints              []HintDef `yaml:"hints"`
}

// GetDescription returns the round's description or text if configured.
func (rd *RoundDef) GetDescription() string {
	if rd == nil {
		return ""
	}
	if rd.Description != "" {
		return rd.Description
	}
	return rd.Text
}

// GetSkipText returns the round's skip_text or skip_url if configured.
func (rd *RoundDef) GetSkipText() string {
	if rd == nil {
		return ""
	}
	if rd.SkipText != "" {
		return rd.SkipText
	}
	return rd.SkipURL
}

// IsSolveLimited returns whether the round has a cap on successful solves.
func (rd *RoundDef) IsSolveLimited() bool {
	if rd == nil {
		return false
	}
	if rd.LimitSolves != nil {
		return *rd.LimitSolves
	}
	if rd.LimitedSubmissions != nil {
		return *rd.LimitedSubmissions
	}
	return rd.MaxSolves > 0 || rd.MaxSubmissions > 0
}

// MaxAllowedSolves returns the max number of solves allowed for this round (default 3 if limited).
func (rd *RoundDef) MaxAllowedSolves() int {
	if rd == nil {
		return 0
	}
	if rd.MaxSolves > 0 {
		return rd.MaxSolves
	}
	if rd.MaxSubmissions > 0 {
		return rd.MaxSubmissions
	}
	if rd.IsSolveLimited() {
		return 3
	}
	return 0
}

// RoundsConfig is the parsed rounds.yaml document.
type RoundsConfig struct {
	AllowHints         *bool      `yaml:"allow_hints,omitempty"`
	AllowSkips         *bool      `yaml:"allow_skips,omitempty"`
	LimitSolves        *bool      `yaml:"limit_solves,omitempty"`
	LimitedSubmissions *bool      `yaml:"limited_submissions,omitempty"`
	MaxSolves          int        `yaml:"max_solves,omitempty"`
	MaxSubmissions     int        `yaml:"max_submissions,omitempty"`
	Rounds             []RoundDef `yaml:"rounds"`

	byID map[int]*RoundDef
}

// HintsAllowed returns whether hints are permitted for the given round (or globally).
func (rc *RoundsConfig) HintsAllowed(roundID int) bool {
	if rc == nil {
		return true
	}
	if rc.AllowHints != nil && !*rc.AllowHints {
		return false
	}
	if roundID > 0 {
		if d := rc.Def(roundID); d != nil && d.AllowHints != nil {
			return *d.AllowHints
		}
	}
	if rc.AllowHints != nil {
		return *rc.AllowHints
	}
	return true
}

// SkipsAllowed returns whether skips are permitted for the given round (or globally).
func (rc *RoundsConfig) SkipsAllowed(roundID int) bool {
	if rc == nil {
		return true
	}
	if rc.AllowSkips != nil && !*rc.AllowSkips {
		return false
	}
	if roundID > 0 {
		if d := rc.Def(roundID); d != nil && d.AllowSkips != nil {
			return *d.AllowSkips
		}
	}
	if rc.AllowSkips != nil {
		return *rc.AllowSkips
	}
	return true
}

// IsSolveLimited returns whether a round is solve-limited.
func (rc *RoundsConfig) IsSolveLimited(roundID int) bool {
	if rc == nil {
		return false
	}
	if roundID > 0 {
		if d := rc.Def(roundID); d != nil {
			if d.LimitSolves != nil {
				return *d.LimitSolves
			}
			if d.LimitedSubmissions != nil {
				return *d.LimitedSubmissions
			}
			if d.MaxSolves > 0 || d.MaxSubmissions > 0 {
				return true
			}
		}
	}
	if rc.LimitSolves != nil {
		return *rc.LimitSolves
	}
	if rc.LimitedSubmissions != nil {
		return *rc.LimitedSubmissions
	}
	return rc.MaxSolves > 0 || rc.MaxSubmissions > 0
}

// MaxAllowedSolves returns the max number of solves allowed for a round (0 means unlimited).
func (rc *RoundsConfig) MaxAllowedSolves(roundID int) int {
	if rc == nil {
		return 0
	}
	if roundID > 0 {
		if d := rc.Def(roundID); d != nil {
			if d.MaxSolves > 0 {
				return d.MaxSolves
			}
			if d.MaxSubmissions > 0 {
				return d.MaxSubmissions
			}
			if d.IsSolveLimited() {
				if rc.MaxSolves > 0 {
					return rc.MaxSolves
				}
				if rc.MaxSubmissions > 0 {
					return rc.MaxSubmissions
				}
				return 3
			}
		}
	}
	if rc.MaxSolves > 0 {
		return rc.MaxSolves
	}
	if rc.MaxSubmissions > 0 {
		return rc.MaxSubmissions
	}
	if rc.IsSolveLimited(roundID) {
		return 3
	}
	return 0
}

// Def returns the definition for a round id (nil if unknown).
func (rc *RoundsConfig) Def(id int) *RoundDef {
	if rc == nil {
		return nil
	}
	return rc.byID[id]
}

// SkipText returns the configured skip_text or skip_url for a round.
func (rc *RoundsConfig) SkipText(roundID int) string {
	if rc == nil {
		return ""
	}
	if d := rc.Def(roundID); d != nil {
		return d.GetSkipText()
	}
	return ""
}

// Hint returns the hint text for (round, type, index) if it exists.
func (rc *RoundsConfig) Hint(roundID int, typ string, idx int) (string, bool) {
	d := rc.Def(roundID)
	if d == nil {
		return "", false
	}
	n := 0
	for _, h := range d.Hints {
		if h.Type == typ {
			if n == idx {
				return h.Text, true
			}
			n++
		}
	}
	return "", false
}

// CountHints counts configured hints of a type for a round.
func (rc *RoundsConfig) CountHints(roundID int, typ string) int {
	d := rc.Def(roundID)
	if d == nil {
		return 0
	}
	n := 0
	for _, h := range d.Hints {
		if h.Type == typ {
			n++
		}
	}
	return n
}

// ParseRounds decodes rounds.yaml bytes and builds the lookup index.
func ParseRounds(data []byte) (*RoundsConfig, error) {
	rc := &RoundsConfig{}
	if err := yaml.Unmarshal(data, rc); err != nil {
		return nil, err
	}
	rc.byID = make(map[int]*RoundDef, len(rc.Rounds))
	for i := range rc.Rounds {
		d := &rc.Rounds[i]
		if d.ID <= 0 {
			return nil, fmt.Errorf("round %d: id must be positive", i)
		}
		if d.Description == "" && d.Text != "" {
			d.Description = d.Text
		} else if d.Text == "" && d.Description != "" {
			d.Text = d.Description
		}
		rc.byID[d.ID] = d
		for _, h := range d.Hints {
			if h.Type != "plain" && h.Type != "encoded" {
				return nil, fmt.Errorf("round %d: hint type must be plain|encoded (got %q)", d.ID, h.Type)
			}
		}
	}
	return rc, nil
}
