// Package tui implements the interactive TUI mode using Bubble Tea.
package tui

import (
	"context"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/oneneural/tempad/internal/claim"
	"github.com/oneneural/tempad/internal/config"
	"github.com/oneneural/tempad/internal/domain"
	"github.com/oneneural/tempad/internal/tracker"
	"github.com/oneneural/tempad/internal/workspace"
)

// viewState tracks which view is currently displayed.
type viewState int

const (
	viewBoard  viewState = iota // task board (default)
	viewDetail                  // single issue detail
)

// Model is the root Bubble Tea model for TUI mode.
type Model struct {
	// Dependencies
	cfg     *config.ServiceConfig
	tracker tracker.Client
	ws      *workspace.Manager

	// View state
	view viewState

	// Board data
	available []domain.Issue // unassigned candidates
	active    []domain.Issue // assigned to current user
	cursor    int            // selected index in the combined list
	selectedID string        // preserved across poll refreshes

	// Detail view
	detailIssue *domain.Issue

	// Poll state
	pollInFlight bool

	// Selection flow state
	claiming bool // true while claim→workspace→IDE is in progress

	// UI state
	width  int
	height int
	err    error
	status string // transient status message

	// Context for cancellation
	ctx context.Context
}

// NewModel creates a new TUI model with the given dependencies.
func NewModel(ctx context.Context, cfg *config.ServiceConfig, client tracker.Client, ws *workspace.Manager) Model {
	return Model{
		cfg:     cfg,
		tracker: client,
		ws:      ws,
		view:    viewBoard,
		ctx:     ctx,
	}
}

// Init implements tea.Model. Fires an initial poll.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.pollCmd(),
		m.tickCmd(),
	)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		if m.pollInFlight {
			return m, m.tickCmd()
		}
		m.pollInFlight = true
		return m, tea.Batch(m.pollCmd(), m.tickCmd())

	case PollResultMsg:
		m.pollInFlight = false
		m.status = ""
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.err = nil
		m.available = msg.Available
		m.active = msg.Active
		m.restoreCursor()
		return m, nil

	case ClaimResultMsg:
		if msg.Err != nil {
			m.claiming = false
			m.err = msg.Err
			m.status = "Claim failed"
			return m, nil
		}
		m.status = "Preparing workspace..."
		return m, m.prepareWorkspaceCmd(msg.Issue)

	case WorkspaceReadyMsg:
		if msg.Err != nil {
			m.claiming = false
			m.err = msg.Err
			m.status = "Workspace error"
			return m, nil
		}
		m.status = "Opening IDE..."
		return m, m.openIDECmd(msg.Issue, msg.Workspace)

	case IDEOpenedMsg:
		m.claiming = false
		if msg.Err != nil {
			m.err = msg.Err
			m.status = "IDE launch failed"
			return m, nil
		}
		m.err = nil
		m.status = "Opened in IDE"
		// Trigger a refresh to update the board (issue is now assigned).
		if !m.pollInFlight {
			m.pollInFlight = true
			return m, m.pollCmd()
		}
		return m, nil

	case ReleaseResultMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.status = "Release failed"
			return m, nil
		}
		m.status = "Task released"
		// Refresh to show updated board.
		if !m.pollInFlight {
			m.pollInFlight = true
			return m, m.pollCmd()
		}
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	default:
		// Delegate to view-specific update.
		switch m.view {
		case viewBoard:
			return m.updateBoard(msg)
		case viewDetail:
			return m.updateDetail(msg)
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	switch m.view {
	case viewDetail:
		return m.viewDetail()
	default:
		return m.viewBoard()
	}
}

// allIssues returns the combined list: available then active.
func (m Model) allIssues() []domain.Issue {
	all := make([]domain.Issue, 0, len(m.available)+len(m.active))
	all = append(all, m.available...)
	all = append(all, m.active...)
	return all
}

// selectedIssue returns the issue at the current cursor position, or nil.
func (m Model) selectedIssue() *domain.Issue {
	all := m.allIssues()
	if m.cursor >= 0 && m.cursor < len(all) {
		issue := all[m.cursor]
		return &issue
	}
	return nil
}

// restoreCursor preserves selection across poll refreshes by finding the
// previously selected issue ID in the new list.
func (m *Model) restoreCursor() {
	if m.selectedID == "" {
		m.cursor = 0
		return
	}
	all := m.allIssues()
	for i, issue := range all {
		if issue.ID == m.selectedID {
			m.cursor = i
			return
		}
	}
	// Issue no longer in list — clamp cursor.
	if m.cursor >= len(all) {
		m.cursor = max(0, len(all)-1)
	}
}

// pollCmd creates a tea.Cmd that fetches issues from the tracker.
func (m Model) pollCmd() tea.Cmd {
	client := m.tracker
	ctx := m.ctx
	return func() tea.Msg {
		candidates, err := client.FetchCandidateIssues(ctx)
		if err != nil {
			return PollResultMsg{Err: err}
		}
		// Split into available (unassigned) and active (assigned to me).
		var available, active []domain.Issue
		for _, issue := range candidates {
			if issue.Assignee != "" {
				active = append(active, issue)
			} else {
				available = append(available, issue)
			}
		}
		return PollResultMsg{Available: available, Active: active}
	}
}

// tickCmd schedules the next poll tick.
func (m Model) tickCmd() tea.Cmd {
	interval := time.Duration(m.cfg.PollIntervalMs) * time.Millisecond
	return tea.Tick(interval, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

// claimCmd creates a tea.Cmd that claims an issue.
func (m Model) claimCmd(issue domain.Issue) tea.Cmd {
	client := m.tracker
	ctx := m.ctx
	identity := m.cfg.TrackerIdentity
	return func() tea.Msg {
		err := claim.Claim(ctx, client, issue.ID, identity)
		return ClaimResultMsg{Issue: issue, Err: err}
	}
}

// prepareWorkspaceCmd creates a tea.Cmd that prepares a workspace for an issue.
func (m Model) prepareWorkspaceCmd(issue domain.Issue) tea.Cmd {
	ws := m.ws
	ctx := m.ctx
	hooks := workspace.HookConfig{
		AfterCreate: m.cfg.AfterCreateHook,
		BeforeRun:   m.cfg.BeforeRunHook,
		TimeoutMs:   m.cfg.HookTimeoutMs,
	}
	return func() tea.Msg {
		w, err := ws.Prepare(ctx, issue, hooks)
		if err != nil {
			return WorkspaceReadyMsg{Issue: issue, Err: err}
		}
		return WorkspaceReadyMsg{Issue: issue, Workspace: *w}
	}
}

// openIDECmd creates a tea.Cmd that launches the IDE for the workspace.
func (m Model) openIDECmd(issue domain.Issue, ws domain.Workspace) tea.Cmd {
	ideCmd := m.cfg.IDECommand
	ideArgs := m.cfg.IDEArgs
	path := ws.Path
	return func() tea.Msg {
		cmdStr := ideCmd
		if ideArgs != "" {
			cmdStr += " " + ideArgs
		}
		cmdStr += " " + path

		cmd := exec.Command("bash", "-lc", cmdStr)
		if err := cmd.Start(); err != nil {
			return IDEOpenedMsg{Issue: issue, Err: err}
		}
		return IDEOpenedMsg{Issue: issue}
	}
}

// releaseCmd creates a tea.Cmd that releases a claimed issue.
func (m Model) releaseCmd(issueID string) tea.Cmd {
	client := m.tracker
	ctx := m.ctx
	return func() tea.Msg {
		err := claim.Release(ctx, client, issueID)
		return ReleaseResultMsg{IssueID: issueID, Err: err}
	}
}

// updateBoard is defined in keys.go.
// updateDetail and viewDetail are defined in detail.go.
