package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// viewSplitPane renders the board (compressed) + log pane below.
func (m Model) viewSplitPane() string {
	var b strings.Builder

	// Board takes ~40% of height.
	boardHeight := m.height * 40 / 100
	if boardHeight < 5 {
		boardHeight = 5
	}

	board := m.renderBoard(boardHeight)
	b.WriteString(board)
	b.WriteString("\n")

	// Log pane header.
	header := m.logPaneHeader()
	b.WriteString(logHeaderStyle.Render(header))
	b.WriteString("\n")

	// Log content.
	logHeight := m.height - boardHeight - 4 // header + footer + margins
	if logHeight < 3 {
		logHeight = 3
	}
	b.WriteString(m.renderLogLines(logHeight, false))

	// Auto-scroll indicator.
	if m.logAutoScroll {
		b.WriteString(logAutoScrollStyle.Render("  ↓ auto"))
	}
	b.WriteString("\n")

	// Footer.
	b.WriteString(footerStyle.Render("Esc: close logs  J/K: scroll  f: fullscreen  j/k: navigate board"))

	return b.String()
}

// viewFullscreenLogs renders the log pane in fullscreen mode.
func (m Model) viewFullscreenLogs() string {
	var b strings.Builder

	// Header.
	header := m.logPaneHeader()
	b.WriteString(logHeaderStyle.Render(header))
	b.WriteString("\n")

	// Log content fills the screen.
	logHeight := m.height - 3 // header + footer
	if logHeight < 3 {
		logHeight = 3
	}
	b.WriteString(m.renderLogLines(logHeight, true))
	b.WriteString("\n")

	// Footer.
	b.WriteString(footerStyle.Render("Esc/f: back to split  J/K: scroll  G: go to bottom  q: quit"))

	return b.String()
}

// logPaneHeader builds the log pane header string.
func (m Model) logPaneHeader() string {
	issueID := m.logIssueID

	// Find the identifier for this issue.
	identifier := issueID
	for _, issue := range m.allIssues() {
		if issue.ID == issueID {
			identifier = issue.Identifier
			break
		}
	}

	// Check mode.
	if m.hasOrchestrator() {
		if run, ok := m.orchRunning[issueID]; ok {
			attempt := 1
			if run.Attempt != nil {
				attempt = *run.Attempt + 1
			}
			return fmt.Sprintf(" %s · Agent Logs (attempt %d) ", identifier, attempt)
		}
		if entry, ok := m.orchRetryAttempts[issueID]; ok {
			remaining := time.Until(time.UnixMilli(entry.DueAtMs))
			if remaining < 0 {
				remaining = 0
			}
			return fmt.Sprintf(" %s · Retrying in %ds (attempt %d) ", identifier, int(remaining.Seconds()), entry.Attempt+1)
		}
		// Check completed runs.
		for _, run := range m.orchCompletedRuns {
			if run.IssueID == issueID {
				status := "Completed"
				if run.ExitCode != nil {
					if *run.ExitCode == 0 {
						status = "Done"
					} else {
						status = fmt.Sprintf("Failed (exit %d)", *run.ExitCode)
					}
				}
				return fmt.Sprintf(" %s · %s ", identifier, status)
			}
		}
	}

	// IDE mode or no orchestrator.
	return fmt.Sprintf(" %s · Logs ", identifier)
}

// renderLogLines renders the visible portion of the log buffer.
func (m Model) renderLogLines(height int, showStreamTags bool) string {
	if len(m.logLines) == 0 {
		// Show contextual message.
		msg := m.logEmptyMessage()
		return emptyStyle.Render("  " + msg) + "\n"
	}

	var b strings.Builder

	// Calculate visible window.
	total := len(m.logLines)
	start := total - height - m.logScrollPos
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > total {
		end = total
	}

	for i := start; i < end; i++ {
		line := m.logLines[i]

		// Timestamp.
		ts := logTimestampStyle.Render(line.Time.Format("15:04:05"))

		// Stream tag (fullscreen mode).
		var tag string
		if showStreamTags {
			tag = logStreamTagStyle.Render(fmt.Sprintf("[%s]", line.Stream)) + "  "
		}

		// Content styled by stream.
		var content string
		switch line.Stream {
		case "stderr":
			content = logStderrStyle.Render(line.Text)
		case "tempad":
			content = logTempadStyle.Render(line.Text)
		default:
			content = logStdoutStyle.Render(line.Text)
		}

		b.WriteString(fmt.Sprintf("  %s  %s%s\n", ts, tag, content))
	}

	return b.String()
}

// logEmptyMessage returns a contextual empty message for the log pane.
func (m Model) logEmptyMessage() string {
	issueID := m.logIssueID
	if m.hasOrchestrator() {
		if _, ok := m.orchRunning[issueID]; ok {
			return "Waiting for agent output..."
		}
		if _, ok := m.orchRetryAttempts[issueID]; ok {
			return "Waiting for retry..."
		}
	}

	// Check if it's an active (IDE) task.
	for _, issue := range m.active {
		if issue.ID == issueID {
			return "Opened in IDE — No agent logs"
		}
	}

	return "No logs available"
}

// updateSplit handles key events in split view.
func (m Model) updateSplit(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	total := len(m.available) + len(m.active)

	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc":
		// Close log pane, return to board.
		m.view = viewBoard
		m.logIssueID = ""
		return m, nil

	case "f":
		// Toggle to fullscreen logs.
		m.view = viewLogs
		return m, nil

	case "j", "down":
		// Navigate board.
		if total > 0 && m.cursor < total-1 {
			m.cursor++
			m.selectedID = m.issueIDAt(m.cursor)
			// Switch log pane to new selection if it's an active task.
			if issue := m.selectedIssue(); issue != nil && issue.Assignee != "" {
				return m, m.openLogPane(issue.ID)
			}
		}
		return m, nil

	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.selectedID = m.issueIDAt(m.cursor)
			if issue := m.selectedIssue(); issue != nil && issue.Assignee != "" {
				return m, m.openLogPane(issue.ID)
			}
		}
		return m, nil

	case "J":
		// Scroll logs down.
		if m.logScrollPos > 0 {
			m.logScrollPos--
			m.logAutoScroll = m.logScrollPos == 0
		}
		return m, nil

	case "K":
		// Scroll logs up.
		m.logScrollPos++
		m.logAutoScroll = false
		return m, nil

	case "G":
		// Go to bottom.
		m.logScrollPos = 0
		m.logAutoScroll = true
		return m, nil

	case "d":
		if issue := m.selectedIssue(); issue != nil {
			m.detailIssue = issue
			m.view = viewDetail
		}
		return m, nil

	case "r":
		if !m.pollInFlight {
			m.pollInFlight = true
			m.status = "Refreshing..."
			return m, m.pollCmd()
		}
		return m, nil
	}

	return m, nil
}

// updateLogs handles key events in fullscreen log view.
func (m Model) updateLogs(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc", "f":
		// Back to split view.
		m.view = viewSplit
		return m, nil

	case "J":
		if m.logScrollPos > 0 {
			m.logScrollPos--
			m.logAutoScroll = m.logScrollPos == 0
		}
		return m, nil

	case "K":
		m.logScrollPos++
		m.logAutoScroll = false
		return m, nil

	case "G":
		m.logScrollPos = 0
		m.logAutoScroll = true
		return m, nil
	}

	return m, nil
}

// issueRunStatus returns a display status for use in the detail view.
func (m Model) issueRunStatus(issueID string) string {
	if !m.hasOrchestrator() {
		return ""
	}
	if run, ok := m.orchRunning[issueID]; ok {
		elapsed := time.Since(run.StartedAt).Truncate(time.Second)
		return fmt.Sprintf("Agent running (%s)", elapsed)
	}
	if entry, ok := m.orchRetryAttempts[issueID]; ok {
		remaining := time.Until(time.UnixMilli(entry.DueAtMs))
		return fmt.Sprintf("Retry in %ds", int(remaining.Seconds()))
	}
	for _, run := range m.orchCompletedRuns {
		if run.IssueID == issueID {
			if run.ExitCode != nil && *run.ExitCode == 0 {
				return "Completed (exit 0)"
			}
			if run.ExitCode != nil {
				return fmt.Sprintf("Failed (exit %d)", *run.ExitCode)
			}
		}
	}
	return ""
}
