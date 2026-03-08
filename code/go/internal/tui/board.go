package tui

import (
	"fmt"
	"sort"
	"strings"

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

// viewBoard renders the task board with Available and Active sections.
func (m Model) viewBoard() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("TEMPAD"))
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
	b.WriteString(sectionHeaderStyle.Render(" Available Tasks "))
	b.WriteString("\n")

	if len(available) == 0 {
		b.WriteString(emptyStyle.Render("  No available tasks"))
		b.WriteString("\n")
	} else {
		for _, issue := range available {
			line := m.renderIssueRow(issue, idx, isBlocked(issue, terminalStates))
			b.WriteString(line)
			b.WriteString("\n")
			idx++
		}
	}

	b.WriteString("\n")

	// My Active Tasks section.
	b.WriteString(activeSectionHeaderStyle.Render(" My Active Tasks "))
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
	b.WriteString(footerStyle.Render("j/k: navigate  Enter: select  d: details  r: refresh  u: release  o: open URL  q: quit"))

	return b.String()
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

	state := stateStyle.Render(fmt.Sprintf("[%s]", issue.State))

	var labels string
	if len(issue.Labels) > 0 {
		labels = labelStyle.Render(strings.Join(issue.Labels, ", "))
	}

	var parts []string
	parts = append(parts, priority, id, title, state)
	if labels != "" {
		parts = append(parts, labels)
	}
	if blocked {
		parts = append(parts, blockedTagStyle.Render("[BLOCKED]"))
	}

	line := "  " + strings.Join(parts, "  ")

	if selected {
		// Apply selection style — strip existing styles for clean highlight.
		return selectedStyle.Render(fmt.Sprintf("> %s  %s  %s  %s", priorityStyle(issue.Priority), issue.Identifier, issue.Title, issue.State))
	}
	if blocked {
		return blockedStyle.Render(line)
	}

	return line
}
