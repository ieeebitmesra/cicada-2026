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
	ID        int
	Name      string
	Points    int
	FlagHash  string // SHA-256 hex of the exact flag
	IsActive  bool
	SortOrder int
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
	ID       int       `yaml:"id"`
	Name     string    `yaml:"name"`
	Points   int       `yaml:"points"`
	IsActive bool      `yaml:"is_active"`
	FlagHash string    `yaml:"flag_hash"`
	Hints    []HintDef `yaml:"hints"`
}

// RoundsConfig is the parsed rounds.yaml document.
type RoundsConfig struct {
	Rounds []RoundDef `yaml:"rounds"`

	byID map[int]*RoundDef
}

// Def returns the definition for a round id (nil if unknown).
func (rc *RoundsConfig) Def(id int) *RoundDef {
	if rc == nil {
		return nil
	}
	return rc.byID[id]
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
		rc.byID[d.ID] = d
		for _, h := range d.Hints {
			if h.Type != "plain" && h.Type != "encoded" {
				return nil, fmt.Errorf("round %d: hint type must be plain|encoded (got %q)", d.ID, h.Type)
			}
		}
	}
	return rc, nil
}
