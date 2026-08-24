// Package tui — root model: view state machine + router.
package tui

import (
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ieee-ctf/internal/models"
	"ieee-ctf/internal/scoring"
	"ieee-ctf/internal/store"
	"ieee-ctf/internal/tui/components"
	"ieee-ctf/internal/tui/msg"
	"ieee-ctf/internal/tui/views"
)

type viewState int

const (
	viewWelcome viewState = iota
	viewRegister
	viewDashboard
	viewSubmitFlag
	viewRequestHint
	viewSkipRound
	viewScoreboard
	viewTeamStatus
)

// RootModel is the top-level Bubble Tea model routing between views.
type RootModel struct {
	state         viewState
	db            *store.DB
	svc           *scoring.Service
	team          *models.Team
	idleTimeout   time.Duration
	lastActivity  time.Time
	width, height int

	welcome      views.WelcomeModel
	register     views.RegisterModel
	dashboard    views.DashboardModel
	submitFlag   views.SubmitFlagModel
	requestHint  views.RequestHintModel
	skipRound    views.SkipRoundModel
	scoreboard   views.ScoreboardModel
	teamStatus   views.TeamStatusModel
	notification components.Notification
}

// NewRootModel wires the full TUI.
func NewRootModel(db *store.DB, svc *scoring.Service, team *models.Team,
	width, height int, idleTimeout time.Duration) RootModel {
	m := RootModel{
		db:          db,
		svc:         svc,
		team:        team,
		idleTimeout: idleTimeout,
		width:       int(width),
		height:      int(height),

		welcome:      views.NewWelcome(),
		register:     views.NewRegister(db, team),
		dashboard:    views.NewDashboard(svc, team),
		submitFlag:   views.NewSubmitFlag(svc, team),
		requestHint:  views.NewRequestHint(svc, team),
		skipRound:    views.NewSkipRound(svc, team),
		scoreboard:   views.NewScoreboard(db, team),
		teamStatus:   views.NewTeamStatus(svc, team),
	}
	if !team.Registered {
		m.state = viewRegister
	} else {
		m.state = viewWelcome
	}
	m.lastActivity = time.Now()
	m.resizeAll()
	return m
}

func (m *RootModel) resizeAll() {
	w := m.width
	h := m.height
	m.welcome.SetSize(w, h)
	m.register.SetSize(w, h)
	m.dashboard.SetSize(w, h)
	m.submitFlag.SetSize(w, h)
	m.requestHint.SetSize(w, h)
	m.skipRound.SetSize(w, h)
	m.scoreboard.SetSize(w, h)
	m.teamStatus.SetSize(w, h)
}

func (m *RootModel) initCurrentView() tea.Cmd {
	switch m.state {
	case viewDashboard:
		return m.dashboard.Refresh()
	case viewSubmitFlag:
		return m.submitFlag.Refresh()
	case viewRequestHint:
		return m.requestHint.Refresh()
	case viewSkipRound:
		return m.skipRound.Refresh()
	case viewScoreboard:
		return m.scoreboard.Refresh()
	case viewTeamStatus:
		return m.teamStatus.Refresh()
	}
	return nil
}

// Init starts the idle-session watchdog.
func (m RootModel) Init() tea.Cmd { return idleTickCmd() }

type idleTickMsg struct{}

func idleTickCmd() tea.Cmd {
	return tea.Tick(30*time.Second, func(time.Time) tea.Msg { return idleTickMsg{} })
}

// Update routes messages to the active view.
func (m RootModel) Update(m_ tea.Msg) (tea.Model, tea.Cmd) {
	switch t := m_.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = t.Width, t.Height
		m.resizeAll()
		return m, nil

	case tea.KeyMsg:
		m.lastActivity = time.Now()
		switch t.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.state == viewScoreboard || m.state == viewTeamStatus {
				m.state = viewDashboard
				return m, m.initCurrentView()
			}
			if m.state == viewWelcome || m.state == viewRegister || m.state == viewDashboard {
				return m, nil // views own esc elsewhere; these states ignore it
			}
		}

	case msg.Navigate:
		m.state = viewState(t.Target)
		if t.Target == msg.TRegister {
			m.register = views.NewRegister(m.db, m.team)
			m.resizeAll()
		}
		return m, m.initCurrentView()

	case idleTickMsg:
		if m.idleTimeout > 0 && time.Since(m.lastActivity) > m.idleTimeout {
			return m, tea.Quit
		}
		return m, idleTickCmd()

	default:
		if m.notification.Handle(m_) {
			return m, nil
		}
	}

	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch m.state {
	case viewWelcome:
		m.welcome, cmd = m.welcome.Update(m_)
	case viewRegister:
		m.register, cmd = m.register.Update(m_)
	case viewDashboard:
		m.dashboard, cmd = m.dashboard.Update(m_)
	case viewSubmitFlag:
		m.submitFlag, cmd = m.submitFlag.Update(m_)
	case viewRequestHint:
		m.requestHint, cmd = m.requestHint.Update(m_)
	case viewSkipRound:
		m.skipRound, cmd = m.skipRound.Update(m_)
	case viewScoreboard:
		m.scoreboard, cmd = m.scoreboard.Update(m_)
	case viewTeamStatus:
		m.teamStatus, cmd = m.teamStatus.Update(m_)
	}
	cmds = append(cmds, cmd)

	if flash, ok := m_.(msg.StatusFlash); ok {
		cmds = append(cmds, m.notification.Set(flash.Text, flash.Success))
	}
	return m, tea.Batch(cmds...)
}

func (m RootModel) currentScore() string {
	b, err := m.svc.Breakdown(m.team.ID)
	if err != nil {
		return "?"
	}
	return strconv.FormatFloat(b.Total, 'f', -1, 64)
}

func (m RootModel) footerBindings() []string {
	switch m.state {
	case viewWelcome:
		return []string{"[Enter] continue", "[Ctrl+C] quit"}
	case viewRegister:
		return []string{"[Tab] next field", "[Ctrl+S] register", "[Ctrl+C] quit"}
	case viewDashboard:
		return []string{"[↑/↓] navigate", "[Enter] select", "[r] refresh", "[Ctrl+C] quit"}
	case viewSubmitFlag, viewRequestHint:
		return []string{"[Esc] back", "[Ctrl+S] confirm", "[Ctrl+C] quit"}
	case viewSkipRound:
		return []string{"[↑/↓] choose round", "[Enter] preview skip", "[Esc] back", "[Ctrl+C] quit"}
	case viewScoreboard:
		return []string{"[r] refresh", "[Esc] back", "[Ctrl+C] quit"}
	case viewTeamStatus:
		return []string{"[↑/↓] scroll", "[r] refresh", "[Esc] back", "[Ctrl+C] quit"}
	}
	return nil
}

// View renders header + active view + notification + footer.
func (m RootModel) View() string {
	header := components.Header(m.team, m.currentScore(), m.width)
	footer := components.Footer(m.footerBindings(), m.width)

	var body string
	switch m.state {
	case viewWelcome:
		body = m.welcome.View()
	case viewRegister:
		body = m.register.View()
	case viewDashboard:
		body = m.dashboard.View()
	case viewSubmitFlag:
		body = m.submitFlag.View()
	case viewRequestHint:
		body = m.requestHint.View()
	case viewSkipRound:
		body = m.skipRound.View()
	case viewScoreboard:
		body = m.scoreboard.View()
	case viewTeamStatus:
		body = m.teamStatus.View()
	}

	// Compose vertical layout with a fixed-height body region so the footer
	// stays pinned at the bottom of the terminal.
	headerH := lipgloss.Height(header)
	footerH := lipgloss.Height(footer)
	bodyH := m.height - headerH - footerH - 1 // reserve one line for notifications
	if bodyH < 1 {
		bodyH = 1
	}
	bodyRegion := lipgloss.NewStyle().Height(bodyH).MaxHeight(bodyH).Render(body)

	out := lipgloss.JoinVertical(lipgloss.Left, header, bodyRegion)

	if notif := m.notification.View(m.width); notif != "" {
		out = strings.TrimRight(out, " \n") + "\n" + notif
	} else {
		out += "\n" + strings.Repeat(" ", m.width)
	}
	out = lipgloss.JoinVertical(lipgloss.Left, out, footer)
	return out
}
