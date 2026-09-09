package dashboard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/csivaprasad-c/k8s-dashboard/internal/k8sres"
	"github.com/csivaprasad-c/k8s-dashboard/internal/ui/styles"
)

type namespaceState struct {
	loading bool
	err     string
	all     []string
	filter  string
	cursor  int
}

func (m Model) openNamespacePicker() (Model, tea.Cmd) {
	m.active = overlayNamespace
	m.ns = namespaceState{loading: true}
	cs := m.session.Clientset
	return m, func() tea.Msg {
		names, err := k8sres.ListNamespaces(cs)
		return namespacesLoadedMsg{names: names, err: err}
	}
}

func (m Model) handleNamespacesLoaded(msg namespacesLoadedMsg) (Model, tea.Cmd) {
	m.ns.loading = false
	if msg.err != nil {
		m.ns.err = msg.err.Error()
		return m, nil
	}
	m.ns.all = msg.names
	return m, nil
}

func (m Model) filteredNamespaces() []string {
	if m.ns.filter == "" {
		return m.ns.all
	}
	q := strings.ToLower(m.ns.filter)
	var out []string
	for _, n := range m.ns.all {
		if strings.Contains(strings.ToLower(n), q) {
			out = append(out, n)
		}
	}
	return out
}

func (m Model) updateNamespaceKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+k":
		m.active = overlayNone
		return m, nil
	case "enter":
		list := m.filteredNamespaces()
		if m.ns.cursor >= 0 && m.ns.cursor < len(list) {
			m.namespace = list[m.ns.cursor]
			m.active = overlayNone
			m.loading = true
			m.loadErr = ""
			return m, m.loadCmd()
		}
		return m, nil
	case "up":
		if m.ns.cursor > 0 {
			m.ns.cursor--
		}
		return m, nil
	case "down":
		if m.ns.cursor < len(m.filteredNamespaces())-1 {
			m.ns.cursor++
		}
		return m, nil
	case "backspace":
		if len(m.ns.filter) > 0 {
			m.ns.filter = m.ns.filter[:len(m.ns.filter)-1]
			m.ns.cursor = 0
		}
		return m, nil
	default:
		// len > 1 happens too, not just fast typing: bubbletea can coalesce
		// several bytes arriving in one read into a single multi-rune
		// KeyMsg, so this must not require exactly one rune.
		if len(msg.Runes) > 0 {
			m.ns.filter += string(msg.Runes)
			m.ns.cursor = 0
		}
		return m, nil
	}
}

func (m Model) renderNamespaceOverlay() string {
	var b strings.Builder
	b.WriteString(styles.Title.Render("Switch namespace"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Filter: %s\n\n", m.ns.filter))
	if m.ns.loading {
		b.WriteString(styles.Muted.Render("loading namespaces…"))
	} else if m.ns.err != "" {
		b.WriteString(styles.ErrorText.Render(m.ns.err))
	} else {
		for i, n := range m.filteredNamespaces() {
			cursor := "  "
			line := n
			if n == m.namespace {
				line += "  (current)"
			}
			if i == m.ns.cursor {
				cursor = "▸ "
				line = styles.SelectedRow.Render(line)
			}
			b.WriteString(cursor + line + "\n")
		}
	}
	b.WriteString("\n" + styles.StatusBar.Render("↑/↓ move · type to filter · enter select · esc cancel"))
	return styles.Border.Padding(1, 2).Render(b.String())
}
