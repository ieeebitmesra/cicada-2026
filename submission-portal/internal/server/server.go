package server

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	"golang.org/x/time/rate"

	"ieee-ctf/internal/auth"
	mw "ieee-ctf/internal/middleware"
	"ieee-ctf/internal/scoring"
	"ieee-ctf/internal/store"
	"ieee-ctf/internal/tui"
)

// New builds the Wish SSH server with the full security middleware chain.
func New(cfg *Config, db *store.DB, svc *scoring.Service) (*ssh.Server, error) {
	return wish.NewServer(
		wish.WithAddress(cfg.Server.Host+":"+cfg.Server.Port),
		wish.WithHostKeyPath(cfg.Server.HostKeyPath),

		// SSH authentication (bcrypt against the teams table).
		wish.WithPasswordAuth(func(ctx ssh.Context, password string) bool {
			return auth.VerifyTeamLoginWithIP(db, ctx.User(), password, ctx.RemoteAddr().String())
		}),

		// Middleware chain (outermost → innermost).
		wish.WithMiddleware(
			mw.RateLimitMiddleware( // Layer 2: per-IP token bucket
				rate.Limit(cfg.Security.RateLimit.RequestsPerSecond),
				cfg.Security.RateLimit.Burst,
			),
			mw.MaxSessionsMiddleware( // Layer 2: session caps
				int32(cfg.Security.Sessions.MaxGlobal),
				cfg.Security.Sessions.MaxPerIP,
			),
			mw.TimeoutMiddleware( // Layer 2: hard session timeout
				time.Duration(cfg.Security.Timeouts.MaxSessionMinutes)*time.Minute,
				time.Duration(cfg.Security.Timeouts.IdleTimeoutMinutes)*time.Minute,
			),
			bm.Middleware(func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
				team, err := db.GetTeamBySSHUser(s.User())
				if err != nil || team == nil {
					wish.Fatalln(s, "Unknown account.")
					return exitModel{}, []tea.ProgramOption{}
				}
				pty, _, _ := s.Pty()
				root := tui.NewRootModel(
					db,
					svc,
					team,
					pty.Window.Width,
					pty.Window.Height,
					time.Duration(cfg.Security.Timeouts.IdleTimeoutMinutes)*time.Minute,
				)
				return root, []tea.ProgramOption{tea.WithAltScreen()}
			}),
			activeterm.Middleware(),
			lm.Middleware(),
		),
	)
}

// exitModel is a fallback model that quits immediately; used only when an
// authenticated session cannot be mapped to a team row.
type exitModel struct{}

func (exitModel) Init() tea.Cmd { return tea.Quit }
func (exitModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return exitModel{}, tea.Quit
}
func (exitModel) View() string { return "Session unavailable.\n" }
