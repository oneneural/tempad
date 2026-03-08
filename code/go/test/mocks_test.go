package test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/oneneural/tempad/internal/agent"
	"github.com/oneneural/tempad/internal/domain"
)

// e2eTracker is a mock tracker for e2e tests.
type e2eTracker struct {
	mu               sync.Mutex
	issues           []domain.Issue
	states           map[string]string
	assigned         map[string]string
	fetchCalls       atomic.Int32
	assignCalls      atomic.Int32
	unassignCalls    atomic.Int32
	statesFetchCalls atomic.Int32
}

func newE2ETracker(issues []domain.Issue) *e2eTracker {
	states := make(map[string]string)
	for _, iss := range issues {
		states[iss.ID] = iss.State
	}
	return &e2eTracker{
		issues:   issues,
		states:   states,
		assigned: make(map[string]string),
	}
}

func (m *e2eTracker) FetchCandidateIssues(ctx context.Context) ([]domain.Issue, error) {
	m.fetchCalls.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.Issue
	for _, iss := range m.issues {
		cp := iss
		cp.Assignee = m.assigned[iss.ID]
		if st, ok := m.states[iss.ID]; ok {
			cp.State = st
		}
		result = append(result, cp)
	}
	return result, nil
}

func (m *e2eTracker) FetchIssueStatesByIDs(ctx context.Context, ids []string) (map[string]string, error) {
	m.statesFetchCalls.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string]string)
	for _, id := range ids {
		if st, ok := m.states[id]; ok {
			result[id] = st
		}
	}
	return result, nil
}

func (m *e2eTracker) FetchIssuesByStates(ctx context.Context, states []string) ([]domain.Issue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	stateSet := make(map[string]bool)
	for _, s := range states {
		stateSet[domain.NormalizeState(s)] = true
	}
	var result []domain.Issue
	for _, iss := range m.issues {
		st := iss.State
		if s, ok := m.states[iss.ID]; ok {
			st = s
		}
		if stateSet[domain.NormalizeState(st)] {
			result = append(result, iss)
		}
	}
	return result, nil
}

func (m *e2eTracker) FetchIssue(ctx context.Context, id string) (*domain.Issue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, iss := range m.issues {
		if iss.ID == id {
			cp := iss
			cp.Assignee = m.assigned[id]
			if st, ok := m.states[id]; ok {
				cp.State = st
			}
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("issue %q not found", id)
}

func (m *e2eTracker) AssignIssue(ctx context.Context, issueID string, identity string) error {
	m.assignCalls.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.assigned[issueID] = identity
	return nil
}

func (m *e2eTracker) UnassignIssue(ctx context.Context, issueID string) error {
	m.unassignCalls.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.assigned, issueID)
	return nil
}

func (m *e2eTracker) setIssueState(id, state string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[id] = state
}

// e2eLauncher is a mock agent launcher for e2e tests.
type e2eLauncher struct {
	mu       sync.Mutex
	exitCode int
	delay    time.Duration
	launches atomic.Int32
}

func newE2ELauncher(exitCode int, delay time.Duration) *e2eLauncher {
	return &e2eLauncher{exitCode: exitCode, delay: delay}
}

func (m *e2eLauncher) Launch(ctx context.Context, opts agent.LaunchOpts) (*agent.RunHandle, error) {
	m.launches.Add(1)
	m.mu.Lock()
	ec := m.exitCode
	d := m.delay
	m.mu.Unlock()

	start := time.Now()
	stdoutR, stdoutW := io.Pipe()
	stderrR := strings.NewReader("")
	done := make(chan struct{})
	var once sync.Once

	go func() {
		defer once.Do(func() { close(done) })
		defer stdoutW.Close()
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		deadline := time.After(d)
		for {
			select {
			case <-ctx.Done():
				return
			case <-deadline:
				return
			case <-ticker.C:
				stdoutW.Write([]byte("working...\n"))
			}
		}
	}()

	return &agent.RunHandle{
		Stdout: stdoutR,
		Stderr: stderrR,
		Wait: func() (agent.ExitResult, error) {
			select {
			case <-done:
			case <-ctx.Done():
			}
			duration := time.Since(start)
			if ctx.Err() != nil {
				return agent.ExitResult{ExitCode: 1, Duration: duration}, ctx.Err()
			}
			return agent.ExitResult{ExitCode: ec, Duration: duration}, nil
		},
		Cancel: func() {
			once.Do(func() { close(done) })
		},
	}, nil
}
