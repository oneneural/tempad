package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/oneneural/tempad/internal/orchestrator"
)

type handlers struct {
	orch *orchestrator.Orchestrator
}

// healthz returns a simple health check response.
func (h *handlers) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// getState returns the full orchestrator state as JSON.
func (h *handlers) getState(w http.ResponseWriter, r *http.Request) {
	state, unlock := h.orch.State().Snapshot()
	defer unlock()

	writeJSON(w, http.StatusOK, state)
}

// getIssue returns details for a specific issue by identifier.
func (h *handlers) getIssue(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "identifier")
	if identifier == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "identifier required"})
		return
	}

	state, unlock := h.orch.State().Snapshot()
	defer unlock()

	// Search running issues by identifier.
	for _, run := range state.Running {
		if run.IssueIdentifier == identifier {
			writeJSON(w, http.StatusOK, run)
			return
		}
	}

	// Search retry queue.
	for _, entry := range state.RetryAttempts {
		if entry.Identifier == identifier {
			writeJSON(w, http.StatusOK, entry)
			return
		}
	}

	writeJSON(w, http.StatusNotFound, map[string]string{
		"error": fmt.Sprintf("issue %q not found", identifier),
	})
}

// triggerRefresh triggers an immediate poll cycle.
func (h *handlers) triggerRefresh(w http.ResponseWriter, r *http.Request) {
	h.orch.TriggerPoll()
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "refresh triggered"})
}

// dashboard returns a simple HTML dashboard.
func (h *handlers) dashboard(w http.ResponseWriter, r *http.Request) {
	state, unlock := h.orch.State().Snapshot()
	defer unlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>TEMPAD Dashboard</title>
<style>
body { font-family: monospace; max-width: 800px; margin: 40px auto; padding: 0 20px; }
h1 { border-bottom: 2px solid #333; }
table { border-collapse: collapse; width: 100%%; }
td, th { padding: 8px; text-align: left; border-bottom: 1px solid #ddd; }
th { background: #f0f0f0; }
.running { color: green; } .retry { color: orange; }
</style>
</head>
<body>
<h1>TEMPAD Dashboard</h1>
<h2>Status</h2>
<p>Running: %d | Claimed: %d | Completed: %d</p>
<p>Total Runtime: %.1fs</p>
`, len(state.Running), len(state.Claimed), len(state.Completed),
		state.AgentTotals.TotalRuntimeSeconds)

	if len(state.Running) > 0 {
		fmt.Fprintf(w, `<h2>Running Agents</h2>
<table><tr><th>Issue</th><th>Attempt</th><th>Status</th><th>Started</th></tr>`)
		for _, run := range state.Running {
			attempt := 0
			if run.Attempt != nil {
				attempt = *run.Attempt
			}
			fmt.Fprintf(w, `<tr><td>%s</td><td>%d</td><td class="running">%s</td><td>%s</td></tr>`,
				run.IssueIdentifier, attempt, run.Status, run.StartedAt.Format("15:04:05"))
		}
		fmt.Fprintf(w, "</table>")
	}

	if len(state.RetryAttempts) > 0 {
		fmt.Fprintf(w, `<h2>Retry Queue</h2>
<table><tr><th>Issue</th><th>Attempt</th><th>Error</th></tr>`)
		for _, entry := range state.RetryAttempts {
			fmt.Fprintf(w, `<tr><td>%s</td><td>%d</td><td class="retry">%s</td></tr>`,
				entry.Identifier, entry.Attempt, entry.Error)
		}
		fmt.Fprintf(w, "</table>")
	}

	fmt.Fprintf(w, `</body></html>`)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
