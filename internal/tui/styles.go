package tui

import "github.com/charmbracelet/lipgloss"

var (
	TitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Bold(true)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12"))

	LabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	ValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11"))

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))

	CriticalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true)

	TabActiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("4")).
			Padding(0, 1)

	TabInactiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	ProgressBarStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("4"))

	ProgressFillStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("10"))

	BorderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	DimStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))
)
