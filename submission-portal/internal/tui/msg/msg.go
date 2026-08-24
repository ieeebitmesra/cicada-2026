// Package msg defines messages shared between the root model and TUI views.
package msg

// Target identifies a routable view state.
type Target int

const (
	TWelcome Target = iota
	TRegister
	TDashboard
	TSubmitFlag
	TRequestHint
	TSkipRound
	TScoreboard
	TTeamStatus
)

// Navigate asks the root model to switch views.
type Navigate struct{ Target Target }

// StatusFlash shows a transient success/error notification.
type StatusFlash struct {
	Text    string
	Success bool
}

// ClearNotification expires the current flash message.
type ClearNotification struct{}

// ConfirmResult reports a yes/no dialog outcome.
type ConfirmResult struct {
	ID        int
	Yes       bool
	Cancelled bool
}

// ScoreboardTick triggers a periodic leaderboard refresh.
type ScoreboardTick struct{}

// Quit requests application exit.
type Quit struct{}
