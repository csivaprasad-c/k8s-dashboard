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

// scrollWindow returns the [start, end) bounds for showing at most
// maxVisible of n items so the cursor always stays inside the visible
// window, scrolling to follow it — used by the namespace and resource
// picker overlays, whose list can run well past what fits on screen (e.g.
// a cluster with dozens of namespaces).
func scrollWindow(n, cursor, maxVisible int) (start, end int) {
	if maxVisible <= 0 || n <= maxVisible {
		return 0, n
	}
	start = cursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	end = start + maxVisible
	if end > n {
		end = n
		start = end - maxVisible
	}
	return start, end
}

// pickerMaxVisible is how many list rows a picker overlay (namespace,
// resource) can show without its box overflowing the terminal: title(1) +
// blank(1) + filter(1) + blank(1) + blank(1) + footer(1) + up to 2
// scroll-indicator lines + Border(2) + Padding(1,2)(2) = 12 non-list lines.
func pickerMaxVisible(height int) int {
	v := height - 12
	if v < 3 {
		v = 3
	}
	return v
}
