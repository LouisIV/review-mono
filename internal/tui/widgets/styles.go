package widgets

import "github.com/charmbracelet/lipgloss"

var (
	borderStyle  = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	activeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("57"))
	addStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	removeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	hunkStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("105"))
	commentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	searchStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("220"))
)
