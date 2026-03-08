package workspace

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

const (
	maxOutputBytes = 4096 // Truncate hook output in logs.
)

// HookResult contains the output from a hook execution.
type HookResult struct {
	Stdout string
	Stderr string
}

// RunHook executes a shell script in the given workspace directory with a timeout.
// It uses process groups so the entire process tree can be killed on timeout.
func RunHook(ctx context.Context, name, script, workspaceDir string, timeoutMs int, env map[string]string) (*HookResult, error) {
	if script == "" {
		return &HookResult{}, nil
	}

	timeout := time.Duration(timeoutMs) * time.Millisecond
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-lc", script)
	cmd.Dir = workspaceDir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Set environment variables.
	if len(env) > 0 {
		cmd.Env = append(cmd.Environ(), mapToEnv(env)...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &HookResult{
		Stdout: truncateOutput(stdout.String()),
		Stderr: truncateOutput(stderr.String()),
	}

	if err != nil {
		// Check if it was a timeout.
		if ctx.Err() == context.DeadlineExceeded {
			// Kill the entire process group.
			if cmd.Process != nil {
				syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			}
			return result, fmt.Errorf("hook %q timed out after %dms", name, timeoutMs)
		}
		return result, fmt.Errorf("hook %q failed: %w", name, err)
	}

	return result, nil
}

func mapToEnv(m map[string]string) []string {
	env := make([]string, 0, len(m))
	for k, v := range m {
		env = append(env, k+"="+v)
	}
	return env
}

func truncateOutput(s string) string {
	if len(s) <= maxOutputBytes {
		return s
	}
	return s[:maxOutputBytes] + "\n... (truncated)"
}
