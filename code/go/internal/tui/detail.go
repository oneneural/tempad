package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorAccent).
				MarginBottom(1)

	detailLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#AAAAAA"))

	detailValueStyle = lipgloss.NewStyle()

	detailDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCCCCC"))

	detailFooterStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				MarginTop(1)
)

// viewDetail renders the full detail view for the selected issue.
func (m Model) viewDetail() string {
	if m.detailIssue == nil {
		return "No issue selected"
	}

	issue := m.detailIssue
	var b strings.Builder

	// Header
	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("%s  %s", issue.Identifier, issue.Title)))
	b.WriteString("\n\n")

	// Fields
	writeField(&b, "State", issue.State)
	writeField(&b, "Priority", formatPriority(issue.Priority))

	if issue.Assignee != "" {
		writeField(&b, "Assignee", issue.Assignee)
	}

	if len(issue.Labels) > 0 {
		writeField(&b, "Labels", strings.Join(issue.Labels, ", "))
	}

	if issue.URL != "" {
		writeField(&b, "URL", issue.URL)
	}

	if issue.BranchName != "" {
		writeField(&b, "Branch", issue.BranchName)
	}

	if issue.CreatedAt != nil {
		writeField(&b, "Created", issue.CreatedAt.Format("2006-01-02 15:04"))
	}

	if issue.UpdatedAt != nil {
		writeField(&b, "Updated", issue.UpdatedAt.Format("2006-01-02 15:04"))
	}

	// Blockers
	if len(issue.BlockedBy) > 0 {
		b.WriteString("\n")
		b.WriteString(detailLabelStyle.Render("Blocked By:"))
		b.WriteString("\n")
		for _, blocker := range issue.BlockedBy {
			b.WriteString(fmt.Sprintf("  - %s [%s]\n", blocker.Identifier, blocker.State))
		}
	}

	// Description
	if issue.Description != "" {
		b.WriteString("\n")
		b.WriteString(detailLabelStyle.Render("Description:"))
		b.WriteString("\n")

		// Word-wrap description to terminal width.
		maxWidth := 80
		if m.width > 10 {
			maxWidth = m.width - 4
		}
		wrapped := wordWrap(issue.Description, maxWidth)
		b.WriteString(detailDescStyle.Render(wrapped))
		b.WriteString("\n")
	}

	// Footer
	b.WriteString(detailFooterStyle.Render("Esc/Backspace: back to board"))

	return b.String()
}

// updateDetail handles key events in the detail view.
func (m Model) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc", "backspace":
			m.view = viewBoard
			m.detailIssue = nil
			return m, nil
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func writeField(b *strings.Builder, label, value string) {
	b.WriteString(fmt.Sprintf("%s %s\n",
		detailLabelStyle.Render(label+":"),
		detailValueStyle.Render(value),
	))
}

func formatPriority(p *int) string {
	if p == nil {
		return "None"
	}
	switch *p {
	case 1:
		return "Urgent (P1)"
	case 2:
		return "High (P2)"
	case 3:
		return "Medium (P3)"
	case 4:
		return "Low (P4)"
	default:
		return fmt.Sprintf("P%d", *p)
	}
}

// wordWrap wraps text at the given width, breaking on spaces.
func wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if len(line) <= width {
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}

		words := strings.Fields(line)
		lineLen := 0
		for i, word := range words {
			if i > 0 && lineLen+1+len(word) > width {
				result.WriteString("\n")
				lineLen = 0
			} else if i > 0 {
				result.WriteString(" ")
				lineLen++
			}
			result.WriteString(word)
			lineLen += len(word)
		}
		result.WriteString("\n")
	}

	return strings.TrimRight(result.String(), "\n")
}
