package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/oneneural/tempad/internal/config"
	"github.com/oneneural/tempad/internal/domain"
	"github.com/oneneural/tempad/internal/orchestrator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func testOrchestrator() *orchestrator.Orchestrator {
	cfg := &config.ServiceConfig{
		PollIntervalMs: 60000,
		MaxConcurrent:  3,
	}
	return orchestrator.New(cfg, nil, nil, testLogger(), nil)
}

func startTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	orch := testOrchestrator()
	srv, err := New(0, orch, testLogger()) // port 0 = ephemeral
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go srv.Serve(ctx)
	time.Sleep(50 * time.Millisecond) // Let server start.

	return srv, "http://" + srv.Addr()
}

func TestServer_Healthz(t *testing.T) {
	_, baseURL := startTestServer(t)

	resp, err := http.Get(baseURL + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "ok", result["status"])
}

func TestServer_GetState(t *testing.T) {
	_, baseURL := startTestServer(t)

	resp, err := http.Get(baseURL + "/api/v1/state")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "poll_interval_ms")
}

func TestServer_GetIssue_NotFound(t *testing.T) {
	_, baseURL := startTestServer(t)

	resp, err := http.Get(baseURL + "/api/v1/UNKNOWN-1")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestServer_GetIssue_Found(t *testing.T) {
	orch := testOrchestrator()
	attempt := 0
	orch.State().Running["id-1"] = &domain.RunAttempt{
		IssueID:         "id-1",
		IssueIdentifier: "PROJ-1",
		Attempt:         &attempt,
		StartedAt:       time.Now(),
		Status:          "running",
	}

	srv, err := New(0, orch, testLogger())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go srv.Serve(ctx)
	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get("http://" + srv.Addr() + "/api/v1/PROJ-1")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "PROJ-1")
}

func TestServer_TriggerRefresh(t *testing.T) {
	_, baseURL := startTestServer(t)

	resp, err := http.Post(baseURL+"/api/v1/refresh", "", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
}

func TestServer_Dashboard(t *testing.T) {
	_, baseURL := startTestServer(t)

	resp, err := http.Get(baseURL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "TEMPAD Dashboard")
}

func TestServer_GracefulShutdown(t *testing.T) {
	orch := testOrchestrator()
	srv, err := New(0, orch, testLogger())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- srv.Serve(ctx)
	}()
	time.Sleep(50 * time.Millisecond)

	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err) // Should be nil (ErrServerClosed handled).
	case <-time.After(3 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}
