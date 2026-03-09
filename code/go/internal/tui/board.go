package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/oneneural/tempad/internal/domain"
)

// sortIssues sorts issues by: priority asc (null last) → created_at oldest → identifier.
func sortIssues(issues []domain.Issue) {
	sort.SliceStable(issues, func(i, j int) bool {
		a, b := issues[i], issues[j]

		// Priority ascending. Nil sorts last.
		ap := priorityVal(a.Priority)
		bp := priorityVal(b.Priority)
		if ap != bp {
			return ap < bp
		}

		// Oldest created_at first.
		if a.CreatedAt != nil && b.CreatedAt != nil {
			if !a.CreatedAt.Equal(*b.CreatedAt) {
				return a.CreatedAt.Before(*b.CreatedAt)
			}
		} else if a.CreatedAt != nil {
			return true
		} else if b.CreatedAt != nil {
			return false
		}

		// Identifier lexicographic tie-breaker.
		return a.Identifier < b.Identifier
	})
}

// priorityVal returns a sortable priority value. Nil → 999 (sorts last).
func priorityVal(p *int) int {
	if p == nil {
		return 999
	}
	return *p
}

// isBlocked returns true if the issue is in Todo state with non-terminal blockers.
func isBlocked(issue domain.Issue, terminalStates map[string]bool) bool {
	if domain.NormalizeState(issue.State) != "todo" {
		return false
	}
	return issue.HasNonTerminalBlockers(terminalStates)
}

// viewBoard renders the task board with Available, In Progress, and Completed sections.
func (m Model) viewBoard() string {
	return m.renderBoard(0)
}

// renderBoard renders the board, optionally limiting height for split view.
// maxHeight of 0 means no limit.
func (m Model) renderBoard(maxHeight int) string {
	var b strings.Builder

	// Title + summary bar.
	title := titleStyle.Render("TEMPAD")
	summary := m.renderSummaryBar()
	if summary != "" {
		// Pad between title and summary.
		padding := ""
		if m.width > len("TEMPAD")+len(summary)+4 {
			padding = strings.Repeat(" ", m.width-len("TEMPAD")-len(summary)-4)
		}
		b.WriteString(title + padding + summary)
	} else {
		b.WriteString(title)
	}
	b.WriteString("\n")

	terminalStates := domain.NormalizeStates(m.cfg.TerminalStates)

	// Sort both lists.
	available := make([]domain.Issue, len(m.available))
	copy(available, m.available)
	sortIssues(available)

	active := make([]domain.Issue, len(m.active))
	copy(active, m.active)
	sortIssues(active)

	// Track global cursor index.
	idx := 0

	// Available Tasks section.
	b.WriteString(sectionHeaderStyle.Render(" Available "))
	b.WriteString("\n")

	if len(available) == 0 {
		b.WriteString(emptyStyle.Render("  No available tasks"))
		b.WriteString("\n")
	} else {
		limit := len(available)
		if maxHeight > 0 && limit > 3 {
			limit = 3 // compress for split view
		}
		for i := 0; i < limit; i++ {
			issue := available[i]
			line := m.renderIssueRow(issue, idx, isBlocked(issue, terminalStates))
			b.WriteString(line)
			b.WriteString("\n")
			idx++
		}
		if limit < len(available) {
			b.WriteString(emptyStyle.Render(fmt.Sprintf("  ... and %d more", len(available)-limit)))
			b.WriteString("\n")
			idx += len(available) - limit
		}
	}

	b.WriteString("\n")

	// In Progress section.
	b.WriteString(activeSectionHeaderStyle.Render(" In Progress "))
	b.WriteString("\n")

	if len(active) == 0 {
		b.WriteString(emptyStyle.Render("  No active tasks"))
		b.WriteString("\n")
	} else {
		for _, issue := range active {
			line := m.renderIssueRow(issue, idx, false)
			b.WriteString(line)
			b.WriteString("\n")
			idx++
		}
	}

	// Completed section (only when orchestrator is present).
	if m.hasOrchestrator() && len(m.orchCompletedRuns) > 0 && maxHeight == 0 {
		b.WriteString("\n")
		b.WriteString(completedSectionHeaderStyle.Render(" Completed "))
		b.WriteString("\n")

		limit := len(m.orchCompletedRuns)
		if limit > 5 {
			limit = 5 // show last 5 in board view
		}
		for i := 0; i < limit; i++ {
			run := m.orchCompletedRuns[i]
			b.WriteString(m.renderCompletedRow(run))
			b.WriteString("\n")
		}
	}

	// Status / Error bar.
	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	} else if m.status != "" {
		b.WriteString("\n")
		b.WriteString(statusStyle.Render(m.status))
	}

	// Footer with keybindings.
	b.WriteString("\n")
	footer := "j/k: navigate  Enter: claim  d: details  r: refresh  u: release  o: open URL  q: quit"
	if m.hasOrchestrator() {
		footer = "j/k: navigate  Enter: claim  d: details  l: logs  r: refresh  u: release  q: quit"
	}
	b.WriteString(footerStyle.Render(footer))

	return b.String()
}

// renderSummaryBar creates the top-right status summary.
func (m Model) renderSummaryBar() string {
	if !m.hasOrchestrator() {
		return ""
	}

	var parts []string
	running := len(m.orchRunning)
	retrying := len(m.orchRetryAttempts)

	if running > 0 {
		parts = append(parts, agentRunningStyle.Render(fmt.Sprintf("● %d running", running)))
	}
	if retrying > 0 {
		parts = append(parts, retryPendingStyle.Render(fmt.Sprintf("↻ %d retry", retrying)))
	}

	if len(parts) == 0 {
		return ""
	}
	return summaryStyle.Render(strings.Join(parts, "  "))
}

// issueMode returns the mode indicator for an active issue based on orchestrator state.
func (m Model) issueMode(issueID string) (indicator string, detail string) {
	if !m.hasOrchestrator() {
		return "", ""
	}

	// Check if running as agent.
	if run, ok := m.orchRunning[issueID]; ok {
		attempt := 1
		if run.Attempt != nil {
			attempt = *run.Attempt + 1
		}
		if run.Status == "stalled" {
			return stalledStyle.Render("⧗ Stalled"), ""
		}
		return agentRunningStyle.Render("● Agent"), fmt.Sprintf("attempt %d", attempt)
	}

	// Check if in retry queue.
	if entry, ok := m.orchRetryAttempts[issueID]; ok {
		remaining := time.Until(time.UnixMilli(entry.DueAtMs))
		if remaining < 0 {
			remaining = 0
		}
		return retryPendingStyle.Render("↻ Retry"), fmt.Sprintf("in %ds", int(remaining.Seconds()))
	}

	// Active but not in orchestrator → IDE mode.
	return ideActiveStyle.Render("◐ IDE"), ""
}

// renderIssueRow renders a single issue row with priority, identifier, title, state, labels, blocked marker.
func (m Model) renderIssueRow(issue domain.Issue, idx int, blocked bool) string {
	selected := idx == m.cursor

	priority := priorityStyle(issue.Priority)

	id := identifierStyle.Render(issue.Identifier)

	// Truncate title to reasonable length.
	title := issue.Title
	maxTitle := 50
	if m.width > 100 {
		maxTitle = m.width - 50
	}
	if len(title) > maxTitle {
		title = title[:maxTitle-3] + "..."
	}

	// Determine status indicator.
	var statusText string
	if issue.Assignee != "" && m.hasOrchestrator() {
		indicator, detail := m.issueMode(issue.ID)
		if indicator != "" {
			statusText = indicator
			if detail != "" {
				statusText += "  " + stateStyle.Render(detail)
			}
		} else {
			statusText = stateStyle.Render(fmt.Sprintf("[%s]", issue.State))
		}
	} else {
		statusText = stateStyle.Render(fmt.Sprintf("[%s]", issue.State))
	}

	var parts []string
	parts = append(parts, priority, id, title, statusText)

	if len(issue.Labels) > 0 {
		parts = append(parts, labelStyle.Render(strings.Join(issue.Labels, ", ")))
	}
	if blocked {
		parts = append(parts, blockedTagStyle.Render("[BLOCKED]"))
	}

	line := "  " + strings.Join(parts, "  ")

	if selected {
		// Apply selection style.
		selParts := []string{priorityStyle(issue.Priority), issue.Identifier, issue.Title, statusText}
		return selectedStyle.Render(fmt.Sprintf("> %s", strings.Join(selParts, "  ")))
	}
	if blocked {
		return blockedStyle.Render(line)
	}

	return line
}

// renderCompletedRow renders a single completed run row.
func (m Model) renderCompletedRow(run *domain.RunAttempt) string {
	priority := stateStyle.Render("--")
	id := identifierStyle.Render(run.IssueIdentifier)

	var statusText string
	var duration string
	if run.ExitCode != nil && *run.ExitCode == 0 {
		statusText = doneStyle.Render("✓ Done")
	} else {
		statusText = failedStyle.Render("✗ Failed")
	}

	if run.ExitCode != nil {
		statusText += "  " + stateStyle.Render(fmt.Sprintf("exit %d", *run.ExitCode))
	}

	if run.FinishedAt != nil {
		d := run.FinishedAt.Sub(run.StartedAt).Truncate(time.Second)
		duration = stateStyle.Render(d.String())
	}

	parts := []string{priority, id, statusText}
	if duration != "" {
		parts = append(parts, duration)
	}

	return "  " + strings.Join(parts, "  ")
}
