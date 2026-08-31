package scoring

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"ieee-ctf/internal/auth"
	"ieee-ctf/internal/models"
	"ieee-ctf/internal/store"
)

// ErrRoundInactive / ErrNotRegistered are game-flow guards.
var (
	ErrRoundInactive  = errors.New("round is not active")
	ErrAlreadySkipped = errors.New("round already skipped")
	ErrSolvedNoSkip   = errors.New("round already solved — nothing to skip")
)

// Service wires the store, rounds config and validators into game operations.
type Service struct {
	DB           *store.DB
	Rounds       *models.RoundsConfig
	Flags        *FlagValidator
	HintValidity time.Duration
}

// NewService constructs the game service.
func NewService(db *store.DB, rounds *models.RoundsConfig, flags *FlagValidator, hintValidity time.Duration) *Service {
	return &Service{DB: db, Rounds: rounds, Flags: flags, HintValidity: hintValidity}
}

// SubmitResult describes the outcome of a flag submission.
type SubmitResult struct {
	Correct       bool
	PointsAwarded float64
}

// SubmitFlag validates and records a flag attempt for a round.
func (s *Service) SubmitFlag(team *models.Team, roundID int, raw string) (*SubmitResult, error) {
	flag, err := s.Flags.Check(team.ID, raw)
	if err != nil {
		return nil, err
	}

	round := s.Rounds.Def(roundID)
	if round == nil {
		return nil, fmt.Errorf("%w: unknown round %d", ErrRoundInactive, roundID)
	}
	dbRound, err := s.DB.GetRound(roundID)
	if err != nil || !dbRound.IsActive {
		return nil, ErrRoundInactive
	}

	solved, err := s.DB.HasSolvedRound(team.ID, roundID)
	if err != nil {
		return nil, err
	}
	if solved {
		return nil, store.ErrAlreadySolved
	}

	// FIX SEC-01: Check if round was skipped
	skipped, err := s.DB.HasSkipped(team.ID, roundID)
	if err != nil {
		return nil, err
	}
	if skipped {
		return nil, ErrAlreadySkipped
	}

	// FIX SEC-11: Constant-time hash comparison
	expectedHash := strings.ToLower(round.FlagHash)
	actualHash := HashFlag(flag)
	correct := subtle.ConstantTimeCompare([]byte(actualHash), []byte(expectedHash)) == 1

	if _, err := s.DB.RecordSubmission(team.ID, roundID, flag, correct); err != nil {
		return nil, err
	}

	res := &SubmitResult{Correct: correct}
	if correct {
		res.PointsAwarded = float64(dbRound.Points)
	}
	return res, nil
}

// NextHintIndex returns the index of the next dispenseable hint of a type.
func (s *Service) NextHintIndex(teamID int64, roundID int, hintType string) (int, error) {
	used, err := s.DB.CountHintsUsed(teamID, roundID, hintType)
	if err != nil {
		return 0, err
	}
	total := s.Rounds.CountHints(roundID, hintType)
	if used >= total {
		return 0, ErrNoHintsLeft
	}
	return used, nil
}

// HintChallenge issues a challenge for the next available hint of a type.
func (s *Service) HintChallenge(team *models.Team, roundID int, hintType string) (challenge string, index int, cost float64, err error) {
	// FIX SEC-13: Validate round is active in database
	dbRound, err := s.DB.GetRound(roundID)
	if err != nil || !dbRound.IsActive {
		return "", 0, 0, ErrRoundInactive
	}

	index, err = s.NextHintIndex(team.ID, roundID, hintType)
	if err != nil {
		return "", 0, 0, err
	}
	round := s.Rounds.Def(roundID)
	if round == nil {
		return "", 0, 0, ErrRoundInactive
	}

	// FIX SEC-09: Add cryptographic nonce
	nonce := auth.RandomChallengeToken()
	challenge = BuildHintChallenge(team, roundID, hintType, index, nonce, time.Now())
	return challenge, index, HintCost(round.Points, hintType), nil
}

// RedeemHint verifies the clearsigned proof and dispenses hint text.
func (s *Service) RedeemHint(team *models.Team, roundID int, hintType string, signed string) (text string, cost float64, err error) {
	if len(signed) > 8192 {
		return "", 0, errors.New("PGP message too long")
	}

	// FIX SEC-13: Validate round is active in database
	dbRound, err := s.DB.GetRound(roundID)
	if err != nil || !dbRound.IsActive {
		return "", 0, ErrRoundInactive
	}

	index, err := s.NextHintIndex(team.ID, roundID, hintType)
	if err != nil {
		return "", 0, err
	}

	body, err := auth.VerifyClearsign(team.PGPPubkey, signed)
	if err != nil {
		return "", 0, err
	}
	if err := VerifyHintChallenge(body, team, roundID, hintType, index, s.HintValidity); err != nil {
		return "", 0, err
	}

	text, ok := s.Rounds.Hint(roundID, hintType, index)
	if !ok {
		return "", 0, ErrNoHintsLeft
	}

	round := s.Rounds.Def(roundID)
	cost = HintCost(round.Points, hintType)
	if _, err := s.DB.RecordHintUsage(team.ID, roundID, hintType, index, signed, cost); err != nil {
		return "", 0, err
	}
	return text, cost, nil
}

// PreviewSkipCost computes what skipping a round would cost right now.
func (s *Service) PreviewSkipCost(teamID int64, roundID int) (float64, error) {
	round := s.Rounds.Def(roundID)
	if round == nil {
		return 0, ErrRoundInactive
	}
	return SkipCost(round.Points), nil
}

// SkipRound applies the skip penalty and records it.
func (s *Service) SkipRound(team *models.Team, roundID int) (cost float64, err error) {
	round := s.Rounds.Def(roundID)
	if round == nil {
		return 0, ErrRoundInactive
	}
	// FIX SEC-13: Validate round is active in database
	dbRound, err := s.DB.GetRound(roundID)
	if err != nil || !dbRound.IsActive {
		return 0, ErrRoundInactive
	}

	solved, err := s.DB.HasSolvedRound(team.ID, roundID)
	if err != nil {
		return 0, err
	}
	if solved {
		return 0, ErrSolvedNoSkip
	}
	skipped, err := s.DB.HasSkipped(team.ID, roundID)
	if err != nil {
		return 0, err
	}
	if skipped {
		return 0, ErrAlreadySkipped
	}

	// FIX SEC-02 & SEC-07: Fixed cost based on challenge points
	cost = SkipCost(round.Points)
	if _, err := s.DB.RecordSkip(team.ID, roundID, cost); err != nil {
		return 0, err
	}
	return cost, nil
}

// Breakdown is a team's full score decomposition.
type Breakdown struct {
	Earned       float64
	HintCosts    float64
	SkipCosts    float64
	Total        float64
	SolvedRounds []int
	Hints        []models.HintUsage
	Skips        []models.Skip
}

// Breakdown fetches all components of a team's score.
func (s *Service) Breakdown(teamID int64) (*Breakdown, error) {
	rounds, err := s.DB.RoundsMap()
	if err != nil {
		return nil, err
	}
	// FIX SEC-03: Use SolvedSubmissions to prevent score truncation on 10k+ attempts
	solvedSubs, err := s.DB.SolvedSubmissions(teamID)
	if err != nil {
		return nil, err
	}
	hints, err := s.DB.TeamHints(teamID)
	if err != nil {
		return nil, err
	}
	skips, err := s.DB.TeamSkips(teamID)
	if err != nil {
		return nil, err
	}
	solvedIDs, err := s.DB.SolvedRoundIDs(teamID)
	if err != nil {
		return nil, err
	}

	total := CalculateTeamScore(solvedSubs, rounds, hints, skips)
	b := &Breakdown{
		Total:        total,
		SolvedRounds: solvedIDs,
		Hints:        hints,
		Skips:        skips,
	}
	for _, sub := range solvedSubs {
		b.Earned += float64(rounds[sub.RoundID].Points)
	}
	for _, h := range hints {
		b.HintCosts += h.CostPoints
	}
	for _, sk := range skips {
		b.SkipCosts += sk.CostPoints
	}
	return b, nil
}
