// Package agent handles subprocess management for coding agents.
package agent

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// LaunchOpts configures agent subprocess launch.
type LaunchOpts struct {
	Command       string            // agent command (e.g., "claude-code")
	Args          string            // additional arguments
	WorkspacePath string            // working directory
	Prompt        string            // rendered prompt text
	PromptMethod  string            // "file", "stdin", "arg", "env"
	Env           map[string]string // TEMPAD_ISSUE_ID, etc.
}

// RunHandle provides control over a running agent subprocess.
type RunHandle struct {
	Wait   func() (ExitResult, error) // blocks until agent exits
	Cancel func()                     // kills the agent subprocess
	Stdout io.Reader                  // live stdout stream
	Stderr io.Reader                  // live stderr stream
}

// ExitResult contains the outcome of an agent run.
type ExitResult struct {
	ExitCode int
	Duration time.Duration
}

// Launcher is the interface for starting agent subprocesses.
type Launcher interface {
	Launch(ctx context.Context, opts LaunchOpts) (*RunHandle, error)
}

// SubprocessLauncher launches agents as shell subprocesses.
type SubprocessLauncher struct{}

// NewSubprocessLauncher creates a new SubprocessLauncher.
func NewSubprocessLauncher() *SubprocessLauncher {
	return &SubprocessLauncher{}
}

// Launch starts an agent subprocess with process group isolation.
func (l *SubprocessLauncher) Launch(ctx context.Context, opts LaunchOpts) (*RunHandle, error) {
	// Prepare prompt delivery.
	delivery, err := DeliverPrompt(opts.PromptMethod, opts.Prompt, opts.WorkspacePath)
	if err != nil {
		return nil, fmt.Errorf("prompt delivery: %w", err)
	}

	// Build command string.
	cmdStr := opts.Command
	if opts.Args != "" {
		cmdStr += " " + opts.Args
	}
	if delivery != nil && len(delivery.ExtraArgs) > 0 {
		for _, arg := range delivery.ExtraArgs {
			cmdStr += " " + shellQuote(arg)
		}
	}

	cmd := exec.CommandContext(ctx, "bash", "-lc", cmdStr)
	cmd.Dir = opts.WorkspacePath

	// Process group isolation — kill entire group on cancel.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Environment: inherit parent + TEMPAD vars + delivery vars.
	env := os.Environ()
	for k, v := range opts.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	if delivery != nil {
		for k, v := range delivery.ExtraEnv {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	cmd.Env = env

	// Stdin from delivery.
	if delivery != nil && delivery.StdinPipe != nil {
		cmd.Stdin = delivery.StdinPipe
	}

	// Capture stdout/stderr via pipes.
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start agent: %w", err)
	}

	handle := &RunHandle{
		Stdout: stdout,
		Stderr: stderr,
		Wait: func() (ExitResult, error) {
			err := cmd.Wait()
			duration := time.Since(startTime)
			if delivery != nil && delivery.Cleanup != nil {
				delivery.Cleanup()
			}

			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
					err = nil // Non-zero exit is not an error — it's a result.
				}
			}
			return ExitResult{ExitCode: exitCode, Duration: duration}, err
		},
		Cancel: func() {
			killProcessGroup(cmd)
		},
	}

	return handle, nil
}

// shellQuote wraps a string in single quotes for safe shell interpolation.
func shellQuote(s string) string {
	// Replace single quotes with '\'' (end quote, escaped quote, start quote)
	escaped := strings.ReplaceAll(s, "'", `'"'"'`)
	return "'" + escaped + "'"
}

// killProcessGroup sends SIGTERM, waits 5s, then SIGKILL to the process group.
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid

	// SIGTERM to process group.
	_ = syscall.Kill(-pid, syscall.SIGTERM)

	// Wait 5s for graceful exit, then SIGKILL.
	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(5 * time.Second):
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	}
}
