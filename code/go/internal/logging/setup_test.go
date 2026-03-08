package logging

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLevel(t *testing.T) {
	assert.Equal(t, slog.LevelDebug, ParseLevel("debug"))
	assert.Equal(t, slog.LevelInfo, ParseLevel("info"))
	assert.Equal(t, slog.LevelWarn, ParseLevel("warn"))
	assert.Equal(t, slog.LevelError, ParseLevel("error"))
	assert.Equal(t, slog.LevelInfo, ParseLevel("unknown"))
	assert.Equal(t, slog.LevelInfo, ParseLevel(""))
}

func TestSetup_TUI(t *testing.T) {
	logger := Setup(Config{Level: "debug", Mode: "tui"})
	require.NotNil(t, logger)
	// Should not panic when logging.
	logger.Info("test message", "key", "value")
}

func TestSetup_Daemon(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	logger := Setup(Config{Level: "info", File: logFile, Mode: "daemon"})
	require.NotNil(t, logger)

	logger.Info("daemon test", "key", "value")

	// Verify log file was created.
	_, err := os.Stat(logFile)
	assert.NoError(t, err)
}

func TestIssueLogger(t *testing.T) {
	tmpDir := t.TempDir()

	logger := IssueLogger(tmpDir, "PROJ-123")
	require.NotNil(t, logger)

	logger.Info("issue log test")

	// Verify directory and file created.
	logPath := filepath.Join(tmpDir, "PROJ-123", "agent.log")
	_, err := os.Stat(logPath)
	assert.NoError(t, err)
}
