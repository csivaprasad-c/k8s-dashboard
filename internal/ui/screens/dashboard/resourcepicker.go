package dashboard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/csivaprasad-c/k8s-dashboard/internal/k8sres"
	"github.com/csivaprasad-c/k8s-dashboard/internal/ui/styles"
)

// resourceState backs the ":" "jump to resource" overlay — there are too
// many resource Kinds (see k8sres.Kinds) to give each its own number key,
// so this is a searchable list instead, the same idea as k9s's `:` command
// bar but scoped to just picking a tab.
type resourceState struct {
	filter string
	cursor int
}

func (m Model) openResourcePicker() (Model, tea.Cmd) {
	m.active = overlayResource
	m.rp = resourceState{}
	return m, nil
}

// filteredKinds returns the kinds whose title contains the current filter,
// case-insensitively, preserving k8sres.Kinds order.
func (m Model) filteredKinds() []k8sres.Kind {
	if m.rp.filter == "" {
		return k8sres.Kinds
	}
	q := strings.ToLower(m.rp.filter)
	out := make([]k8sres.Kind, 0, len(k8sres.Kinds))
	for _, k := range k8sres.Kinds {
		if strings.Contains(strings.ToLower(k.Title()), q) {
			out = append(out, k)
		}
	}
	return out
}

func (m Model) updateResourceKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+k":
		m.active = overlayNone
		return m, nil
	case "enter":
		list := m.filteredKinds()
		if m.rp.cursor < 0 || m.rp.cursor >= len(list) {
			return m, nil
		}
		chosen := list[m.rp.cursor]
		m.active = overlayNone
		for i, k := range k8sres.Kinds {
			if k == chosen {
				return m.switchKind(i)
			}
		}
		return m, nil
	case "up":
		if m.rp.cursor > 0 {
			m.rp.cursor--
		}
		return m, nil
	case "down":
		if m.rp.cursor < len(m.filteredKinds())-1 {
			m.rp.cursor++
		}
		return m, nil
	case "backspace":
		if len(m.rp.filter) > 0 {
			m.rp.filter = m.rp.filter[:len(m.rp.filter)-1]
			m.rp.cursor = 0
		}
		return m, nil
	default:
		// len > 1 happens too, not just fast typing: bubbletea can coalesce
		// several bytes arriving in one read into a single multi-rune
		// KeyMsg, so this must not require exactly one rune.
		if len(msg.Runes) > 0 {
			m.rp.filter += string(msg.Runes)
			m.rp.cursor = 0
		}
		return m, nil
	}
}

func (m Model) renderResourceOverlay() string {
	var b strings.Builder
	b.WriteString(styles.Title.Render("Jump to resource"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Filter: %s\n\n", m.rp.filter))

	list := m.filteredKinds()
	if len(list) == 0 {
		b.WriteString(styles.Muted.Render("no matches"))
	}
	for i, k := range list {
		cursor := "  "
		line := k.Title()
		if k == m.currentKind() {
			line += "  (current)"
		}
		if i == m.rp.cursor {
			cursor = "▸ "
			line = styles.SelectedRow.Render(line)
		}
		b.WriteString(cursor + line + "\n")
	}

	b.WriteString("\n" + styles.StatusBar.Render("↑/↓ move · type to filter · enter select · esc cancel"))
	return styles.Border.Padding(1, 2).Render(b.String())
}
