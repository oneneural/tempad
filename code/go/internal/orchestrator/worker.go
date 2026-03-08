package orchestrator

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/oneneural/tempad/internal/agent"
	"github.com/oneneural/tempad/internal/domain"
	"github.com/oneneural/tempad/internal/workspace"
)

// runWorker runs the full worker lifecycle for an issue.
// This runs in its own goroutine. It sends a WorkerResult when done.
// lastOutput is pre-allocated by the orchestrator goroutine for stall detection.
func (o *Orchestrator) runWorker(ctx context.Context, issue domain.Issue, attempt int, lastOutput *atomic.Int64) {
	log := o.logger.With("issue", issue.Identifier, "attempt", attempt)

	// Ensure we always send a result.
	var result WorkerResult
	defer func() {
		if r := recover(); r != nil {
			log.Error("worker panicked", "panic", r)
			result.Err = fmt.Errorf("worker panic: %v", r)
		}
		result.IssueID = issue.ID
		result.Identifier = issue.Identifier
		result.Attempt = attempt
		o.workerResults <- result
	}()

	// 1. Prepare workspace.
	log.Info("preparing workspace")
	hooks := workspace.HookConfig{
		AfterCreate: o.cfg.AfterCreateHook,
		BeforeRun:   o.cfg.BeforeRunHook,
		TimeoutMs:   o.cfg.HookTimeoutMs,
	}
	ws, err := o.ws.Prepare(ctx, issue, hooks)
	if err != nil {
		result.Err = fmt.Errorf("workspace prepare: %w", err)
		result.ExitCode = 1
		return
	}

	// 2. Render prompt.
	log.Info("rendering prompt")
	promptTemplate := "Work on issue {{ issue.identifier }}: {{ issue.title }}"
	// Use workflow prompt if available.
	rendered, err := o.builder.Render(promptTemplate, issue, &attempt)
	if err != nil {
		result.Err = fmt.Errorf("prompt render: %w", err)
		result.ExitCode = 1
		return
	}

	// 3. Build env vars.
	env := map[string]string{
		"TEMPAD_ISSUE_ID":         issue.ID,
		"TEMPAD_ISSUE_IDENTIFIER": issue.Identifier,
		"TEMPAD_ISSUE_TITLE":      issue.Title,
		"TEMPAD_ISSUE_URL":        issue.URL,
		"TEMPAD_WORKSPACE":        ws.Path,
		"TEMPAD_ATTEMPT":          fmt.Sprintf("%d", attempt),
	}

	// 4. Launch agent.
	log.Info("launching agent")
	handle, err := o.launcher.Launch(ctx, agent.LaunchOpts{
		Command:       o.cfg.AgentCommand,
		Args:          o.cfg.AgentArgs,
		WorkspacePath: ws.Path,
		Prompt:        rendered,
		PromptMethod:  o.cfg.PromptDelivery,
		Env:           env,
	})
	if err != nil {
		result.Err = fmt.Errorf("agent launch: %w", err)
		result.ExitCode = 1
		return
	}

	// Drain stdout/stderr in background, updating stall timestamp.
	go drainOutput(handle.Stdout, lastOutput)
	go drainOutput(handle.Stderr, lastOutput)

	// 5. Wait for exit.
	exitResult, err := handle.Wait()
	result.ExitCode = exitResult.ExitCode
	result.Duration = exitResult.Duration
	if err != nil {
		result.Err = err
	}

	// 6. Run after_run hook if configured.
	if o.cfg.AfterRunHook != "" {
		hookEnv := map[string]string{
			"TEMPAD_ISSUE_ID":    issue.ID,
			"TEMPAD_WORKSPACE":   ws.Path,
			"TEMPAD_EXIT_CODE":   fmt.Sprintf("%d", exitResult.ExitCode),
		}
		if _, hookErr := workspace.RunHook(ctx, "after_run", o.cfg.AfterRunHook, ws.Path, o.cfg.HookTimeoutMs, hookEnv); hookErr != nil {
			log.Warn("after_run hook failed", "error", hookErr)
		}
	}

	log.Info("agent finished",
		"exit_code", exitResult.ExitCode,
		"duration", exitResult.Duration,
	)
}

// drainOutput reads from r line by line, updating the lastOutput timestamp.
func drainOutput(r io.Reader, lastOutput *atomic.Int64) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lastOutput.Store(time.Now().UnixNano())
	}
}
