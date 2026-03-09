package domain

import "time"

// RunAttempt represents one execution attempt for one issue in daemon mode.
// See Spec Section 4.1.6.
type RunAttempt struct {
	// IssueID is the tracker-internal ID of the issue being worked.
	IssueID string `json:"issue_id"`

	// IssueIdentifier is the human-readable key (e.g. "ABC-123") for logs.
	IssueIdentifier string `json:"issue_identifier"`

	// Attempt is nil for the first run, >=1 for retries/continuations.
	Attempt *int `json:"attempt"`

	// WorkspacePath is the absolute path to the workspace directory.
	WorkspacePath string `json:"workspace_path"`

	// StartedAt is when this attempt was launched.
	StartedAt time.Time `json:"started_at"`

	// FinishedAt is when the attempt completed (nil if still running).
	FinishedAt *time.Time `json:"finished_at,omitempty"`

	// Status is the current status of the attempt.
	// Values: "running", "succeeded", "failed", "timed_out", "stalled", "canceled"
	Status string `json:"status"`

	// Mode indicates how the issue is being worked: "agent" or "ide".
	Mode string `json:"mode,omitempty"`

	// ExitCode is the agent process exit code (nil if still running or IDE mode).
	ExitCode *int `json:"exit_code,omitempty"`

	// Error is an optional error message if the attempt failed.
	Error string `json:"error,omitempty"`
}

// RetryEntry is the scheduled retry state for an issue in daemon mode.
// See Spec Section 4.1.7.
type RetryEntry struct {
	// IssueID is the tracker-internal ID.
	IssueID string `json:"issue_id"`

	// Identifier is the human-readable key for log context.
	Identifier string `json:"identifier"`

	// Attempt is 1-based for the retry queue.
	Attempt int `json:"attempt"`

	// DueAtMs is the monotonic clock timestamp when this retry should fire.
	DueAtMs int64 `json:"due_at_ms"`

	// TimerHandle is a runtime-specific timer reference.
	// In Go, this will be a *time.Timer stored separately; this field
	// is kept for serialization/observability.
	TimerHandle string `json:"timer_handle,omitempty"`

	// Error is the error from the previous attempt (for logging context).
	Error string `json:"error,omitempty"`

	// IsContinuation is true if this is a continuation retry (exit code 0)
	// rather than a failure retry. Continuation retries don't count toward
	// max_retries.
	IsContinuation bool `json:"is_continuation"`
}

// AgentTotals tracks aggregate resource consumption across all agent runs.
type AgentTotals struct {
	// TotalTokens is the sum of tokens reported by agents (if available).
	TotalTokens int64 `json:"total_tokens"`

	// TotalRuntimeSeconds is the sum of agent wall-clock time.
	TotalRuntimeSeconds float64 `json:"total_runtime_seconds"`
}
