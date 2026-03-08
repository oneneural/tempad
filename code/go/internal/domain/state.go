package domain

import "sync"

// OrchestratorState is the single authoritative in-memory state owned by
// the daemon-mode orchestrator. Only the orchestrator goroutine may mutate
// this struct. See Spec Section 4.1.8.
type OrchestratorState struct {
	// PollIntervalMs is the current effective poll interval.
	PollIntervalMs int `json:"poll_interval_ms"`

	// MaxConcurrentAgents is the current effective global concurrency limit.
	MaxConcurrentAgents int `json:"max_concurrent_agents"`

	// Running maps issue_id to the currently active RunAttempt.
	Running map[string]*RunAttempt `json:"running"`

	// Claimed is the set of issue IDs that are reserved/running/retrying.
	// An issue ID is in this set from claim until release.
	Claimed map[string]bool `json:"claimed"`

	// RetryAttempts maps issue_id to its pending RetryEntry.
	RetryAttempts map[string]*RetryEntry `json:"retry_attempts"`

	// Completed is the set of issue IDs that have completed at least once.
	// Bookkeeping only — not used for dispatch gating.
	Completed map[string]bool `json:"completed"`

	// AgentTotals holds aggregate resource consumption.
	AgentTotals AgentTotals `json:"agent_totals"`

	// mu protects the state for snapshot reads (e.g. HTTP API).
	// The orchestrator goroutine itself doesn't need to lock — it's the
	// sole writer. The lock is only for concurrent readers (HTTP handlers).
	mu sync.RWMutex
}

// NewOrchestratorState creates a new state with all maps initialized.
func NewOrchestratorState(pollIntervalMs, maxConcurrent int) *OrchestratorState {
	return &OrchestratorState{
		PollIntervalMs:      pollIntervalMs,
		MaxConcurrentAgents: maxConcurrent,
		Running:             make(map[string]*RunAttempt),
		Claimed:             make(map[string]bool),
		RetryAttempts:       make(map[string]*RetryEntry),
		Completed:           make(map[string]bool),
	}
}

// Snapshot returns a read-locked copy-safe view of the state for external
// consumers (HTTP API, TUI dashboard). The caller must call the returned
// unlock function when done reading.
func (s *OrchestratorState) Snapshot() (state *OrchestratorState, unlock func()) {
	s.mu.RLock()
	return s, s.mu.RUnlock
}

// RunningCount returns the number of currently running agents.
func (s *OrchestratorState) RunningCount() int {
	return len(s.Running)
}

// IsClaimedOrRunning returns true if the issue is currently claimed,
// running, or in the retry queue.
func (s *OrchestratorState) IsClaimedOrRunning(issueID string) bool {
	return s.Claimed[issueID]
}

// AvailableSlots returns how many more agents can be started.
func (s *OrchestratorState) AvailableSlots() int {
	slots := s.MaxConcurrentAgents - len(s.Running)
	if slots < 0 {
		return 0
	}
	return slots
}
