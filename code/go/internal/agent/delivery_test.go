package agent

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeliverPrompt_File(t *testing.T) {
	dir := t.TempDir()
	prompt := "Work on issue PROJ-42"

	result, err := DeliverPrompt("file", prompt, dir)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify file was written.
	promptPath := filepath.Join(dir, "PROMPT.md")
	content, err := os.ReadFile(promptPath)
	require.NoError(t, err)
	assert.Equal(t, prompt, string(content))

	// Verify env var.
	assert.Equal(t, promptPath, result.ExtraEnv["TEMPAD_PROMPT_FILE"])

	// Cleanup should remove the file.
	result.Cleanup()
	_, err = os.Stat(promptPath)
	assert.True(t, os.IsNotExist(err))
}

func TestDeliverPrompt_Stdin(t *testing.T) {
	prompt := "Work on issue PROJ-42"

	result, err := DeliverPrompt("stdin", prompt, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.StdinPipe)

	data, err := io.ReadAll(result.StdinPipe)
	require.NoError(t, err)
	assert.Equal(t, prompt, string(data))
}

func TestDeliverPrompt_Arg(t *testing.T) {
	prompt := "Work on issue PROJ-42"

	result, err := DeliverPrompt("arg", prompt, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, []string{prompt}, result.ExtraArgs)
}

func TestDeliverPrompt_Env(t *testing.T) {
	prompt := "Work on issue PROJ-42"

	result, err := DeliverPrompt("env", prompt, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, prompt, result.ExtraEnv["TEMPAD_PROMPT"])
}

func TestDeliverPrompt_Unknown(t *testing.T) {
	_, err := DeliverPrompt("unknown", "prompt", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}
