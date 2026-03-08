package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	colorUrgent  = lipgloss.Color("#FF4444") // P1
	colorHigh    = lipgloss.Color("#FF8C00") // P2
	colorMedium  = lipgloss.Color("#FFD700") // P3
	colorLow     = lipgloss.Color("#6699CC") // P4
	colorBlocked = lipgloss.Color("#666666")
	colorMuted   = lipgloss.Color("#888888")
	colorAccent  = lipgloss.Color("#7C3AED")
	colorSuccess = lipgloss.Color("#10B981")

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			MarginBottom(1)

	sectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(colorAccent).
				Padding(0, 1)

	activeSectionHeaderStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("#FFFFFF")).
					Background(colorSuccess).
					Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#3B3B5C"))

	normalStyle = lipgloss.NewStyle()

	blockedStyle = lipgloss.NewStyle().
			Foreground(colorBlocked)

	blockedTagStyle = lipgloss.NewStyle().
			Foreground(colorBlocked).
			Bold(true)

	identifierStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	stateStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	labelStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1)

	statusStyle = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444")).
			Bold(true)

	emptyStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)
)

// priorityStyle returns a styled priority indicator.
func priorityStyle(p *int) string {
	if p == nil {
		return lipgloss.NewStyle().Foreground(colorMuted).Render("--")
	}
	switch *p {
	case 1:
		return lipgloss.NewStyle().Foreground(colorUrgent).Bold(true).Render("P1")
	case 2:
		return lipgloss.NewStyle().Foreground(colorHigh).Bold(true).Render("P2")
	case 3:
		return lipgloss.NewStyle().Foreground(colorMedium).Render("P3")
	case 4:
		return lipgloss.NewStyle().Foreground(colorLow).Render("P4")
	default:
		return lipgloss.NewStyle().Foreground(colorMuted).Render("--")
	}
}
