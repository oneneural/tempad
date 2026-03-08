package tui

import (
	"github.com/oneneural/tempad/internal/config"
	"github.com/oneneural/tempad/internal/domain"
)

// PollResultMsg carries the result of a tracker poll.
type PollResultMsg struct {
	Available []domain.Issue // unassigned candidates
	Active    []domain.Issue // assigned to current user
	Err       error
}

// ClaimResultMsg carries the result of a claim attempt.
type ClaimResultMsg struct {
	Issue domain.Issue
	Err   error
}

// WorkspaceReadyMsg indicates workspace preparation completed.
type WorkspaceReadyMsg struct {
	Issue     domain.Issue
	Workspace domain.Workspace
	Err       error
}

// IDEOpenedMsg indicates the IDE was launched.
type IDEOpenedMsg struct {
	Issue domain.Issue
	Err   error
}

// ReleaseResultMsg carries the result of releasing a claimed task.
type ReleaseResultMsg struct {
	IssueID string
	Err     error
}

// errMsg wraps errors from async commands for consistent handling.
type errMsg struct {
	err error
}

// ConfigReloadMsg carries a reloaded config from the file watcher.
type ConfigReloadMsg struct {
	Cfg *config.ServiceConfig
	Err error
}

// tickMsg triggers a poll cycle.
type tickMsg struct{}
