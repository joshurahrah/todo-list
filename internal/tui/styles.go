package tui

import "github.com/charmbracelet/lipgloss"

var (
	helpStyle         = lipgloss.NewStyle().Faint(true)
	errStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	doneStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selectedStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	selectedDoneStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	activeTabStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Underline(true)
	inactiveTabStyle  = lipgloss.NewStyle().Faint(true)
)
