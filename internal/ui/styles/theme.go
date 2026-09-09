// Package styles centralizes the dashboard's lipgloss color palette and
// reusable style fragments so every screen looks consistent.
package styles

import "github.com/charmbracelet/lipgloss"

var (
	ColorPrimary = lipgloss.Color("39")  // cyan-blue accent
	ColorMuted   = lipgloss.Color("245") // gray secondary text
	ColorOK      = lipgloss.Color("42")  // green
	ColorWarn    = lipgloss.Color("214") // amber
	ColorBad     = lipgloss.Color("203") // red
	ColorBorder  = lipgloss.Color("237")

	Title = lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)

	Muted = lipgloss.NewStyle().Foreground(ColorMuted)

	StatusBar = lipgloss.NewStyle().Foreground(ColorMuted)

	HeaderBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	TabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(ColorPrimary).
			Padding(0, 1)

	TabInactive = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 1)

	Border = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder)

	ErrorText = lipgloss.NewStyle().Foreground(ColorBad).Bold(true)

	SelectedRow = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(ColorPrimary)
)

// StatusStyle maps a Row.StatusClass to a color style for the first
// meaningfully-colored cell in a table row.
func StatusStyle(class string) lipgloss.Style {
	switch class {
	case "ok":
		return lipgloss.NewStyle().Foreground(ColorOK)
	case "warn":
		return lipgloss.NewStyle().Foreground(ColorWarn)
	case "bad":
		return lipgloss.NewStyle().Foreground(ColorBad).Bold(true)
	default:
		return lipgloss.NewStyle()
	}
}
