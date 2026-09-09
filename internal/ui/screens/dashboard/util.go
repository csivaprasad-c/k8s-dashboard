package dashboard

import "github.com/charmbracelet/lipgloss"

func lipglossJoin(parts ...string) string {
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// overlayOn renders a modal box centered over the base screen. Bubble Tea
// has no real layering, so this just replaces the whole frame with a
// centered box on a blank canvas the size of the terminal — simple and
// flicker-free, at the cost of not showing the dimmed base underneath.
func overlayOn(_ string, box string, width, height int) string {
	return lipgloss.Place(max0(width), max0(height), lipgloss.Center, lipgloss.Center, box)
}
