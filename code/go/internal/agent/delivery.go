package agent

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// DeliveryResult holds the outputs of prompt delivery configuration.
type DeliveryResult struct {
	StdinPipe io.Reader          // non-nil for "stdin" delivery
	ExtraArgs []string           // additional CLI args for "arg" delivery
	ExtraEnv  map[string]string  // additional env vars for "env" delivery
	Cleanup   func()             // cleanup function (e.g., remove temp files)
}

// DeliverPrompt prepares the prompt for the agent based on the delivery method.
// Methods: "file", "stdin", "arg", "env".
func DeliverPrompt(method, prompt, workspacePath string) (*DeliveryResult, error) {
	switch method {
	case "file":
		return deliverFile(prompt, workspacePath)
	case "stdin":
		return deliverStdin(prompt)
	case "arg":
		return deliverArg(prompt)
	case "env":
		return deliverEnv(prompt)
	default:
		return nil, fmt.Errorf("unsupported prompt delivery method: %q", method)
	}
}

// deliverFile writes the prompt to PROMPT.md in the workspace directory.
func deliverFile(prompt, workspacePath string) (*DeliveryResult, error) {
	promptPath := filepath.Join(workspacePath, "PROMPT.md")
	if err := os.WriteFile(promptPath, []byte(prompt), 0644); err != nil {
		return nil, fmt.Errorf("write prompt file: %w", err)
	}
	return &DeliveryResult{
		ExtraEnv: map[string]string{
			"TEMPAD_PROMPT_FILE": promptPath,
		},
		Cleanup: func() {
			os.Remove(promptPath)
		},
	}, nil
}

// deliverStdin provides the prompt as an io.Reader for stdin piping.
func deliverStdin(prompt string) (*DeliveryResult, error) {
	return &DeliveryResult{
		StdinPipe: stringReader(prompt),
	}, nil
}

// deliverArg passes the prompt as a CLI argument.
func deliverArg(prompt string) (*DeliveryResult, error) {
	return &DeliveryResult{
		ExtraArgs: []string{prompt},
	}, nil
}

// deliverEnv sets the TEMPAD_PROMPT environment variable.
func deliverEnv(prompt string) (*DeliveryResult, error) {
	return &DeliveryResult{
		ExtraEnv: map[string]string{
			"TEMPAD_PROMPT": prompt,
		},
	}, nil
}

// stringReader wraps a string as an io.Reader.
type stringReaderImpl struct {
	data []byte
	pos  int
}

func stringReader(s string) io.Reader {
	return &stringReaderImpl{data: []byte(s)}
}

func (r *stringReaderImpl) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
