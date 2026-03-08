package tui

import (
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
)

// updateBoard handles key events on the task board.
func (m Model) updateBoard(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	total := len(m.available) + len(m.active)

	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		if total > 0 && m.cursor < total-1 {
			m.cursor++
			m.selectedID = m.issueIDAt(m.cursor)
		}
		return m, nil

	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.selectedID = m.issueIDAt(m.cursor)
		}
		return m, nil

	case "enter":
		// Selection flow — claim → workspace → IDE (T-P406).
		// For now, store selected ID for later wiring.
		if issue := m.selectedIssue(); issue != nil {
			m.selectedID = issue.ID
		}
		return m, nil

	case "d":
		// Open detail view.
		if issue := m.selectedIssue(); issue != nil {
			m.detailIssue = issue
			m.view = viewDetail
		}
		return m, nil

	case "r":
		// Manual refresh.
		if !m.pollInFlight {
			m.pollInFlight = true
			m.status = "Refreshing..."
			return m, m.pollCmd()
		}
		return m, nil

	case "o":
		// Open issue URL in browser.
		if issue := m.selectedIssue(); issue != nil && issue.URL != "" {
			return m, openURL(issue.URL)
		}
		return m, nil

	case "u":
		// Release task — placeholder for T-P407.
		return m, nil
	}

	return m, nil
}

// issueIDAt returns the issue ID at the given cursor index.
func (m Model) issueIDAt(idx int) string {
	all := m.allIssues()
	if idx >= 0 && idx < len(all) {
		return all[idx].ID
	}
	return ""
}

// openURL opens a URL in the default browser.
func openURL(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}
		if err := cmd.Start(); err != nil {
			return errMsg{err: err}
		}
		return nil
	}
}
