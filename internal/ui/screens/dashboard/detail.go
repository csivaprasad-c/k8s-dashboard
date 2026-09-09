package dashboard

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/csivaprasad-c/k8s-dashboard/internal/k8sres"
	"github.com/csivaprasad-c/k8s-dashboard/internal/ui/styles"
)

// detailMode picks what the detail overlay shows: a kubectl-equivalent
// `describe` (the default — it's what you reach for most often) or raw
// YAML. Both are fetched on demand and toggled with d/y without leaving
// the overlay.
type detailMode int

const (
	detailModeDescribe detailMode = iota
	detailModeYAML
)

type detailState struct {
	kind      k8sres.Kind
	name      string
	namespace string
	mode      detailMode
	loading   bool
	err       string
	vp        viewport.Model
}

func (m Model) openDetail() (Model, tea.Cmd) {
	kind := m.currentKind()
	if kind == k8sres.KindOverview {
		return m, nil
	}
	row, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	m.active = overlayDetail
	m.dt = detailState{kind: kind, name: row.Name, namespace: row.Namespace, mode: detailModeDescribe, loading: true, vp: m.dt.vp}
	m.dt.vp.SetContent("")
	m.dt.vp.GotoTop()
	return m, m.fetchDetailCmd()
}

// fetchDetailCmd fetches content for the detail overlay's current
// kind/namespace/name/mode.
func (m Model) fetchDetailCmd() tea.Cmd {
	clients := m.clients()
	kind, ns, name, mode := m.dt.kind, m.dt.namespace, m.dt.name, m.dt.mode
	return func() tea.Msg {
		var content string
		var err error
		if mode == detailModeYAML {
			content, err = k8sres.GetYAML(clients, kind, ns, name)
		} else {
			content, err = k8sres.Describe(clients, kind, ns, name)
		}
		return yamlLoadedMsg{content: content, err: err, mode: mode}
	}
}

func (m Model) handleYAMLLoaded(msg yamlLoadedMsg) (Model, tea.Cmd) {
	if m.active != overlayDetail || msg.mode != m.dt.mode {
		return m, nil // overlay closed, or the user toggled modes again before this arrived
	}
	m.dt.loading = false
	if msg.err != nil {
		m.dt.err = msg.err.Error()
		return m, nil
	}
	m.dt.err = ""
	m.dt.vp.SetContent(msg.content)
	m.dt.vp.GotoTop()
	return m, nil
}

// switchDetailMode toggles the overlay to mode, re-fetching if it isn't
// already showing that mode.
func (m Model) switchDetailMode(mode detailMode) (Model, tea.Cmd) {
	if m.dt.mode == mode {
		return m, nil
	}
	m.dt.mode = mode
	m.dt.loading = true
	m.dt.err = ""
	m.dt.vp.SetContent("")
	return m, m.fetchDetailCmd()
}

func (m Model) updateDetailKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.active = overlayNone
		return m, nil
	case "d":
		return m.switchDetailMode(detailModeDescribe)
	case "y":
		return m.switchDetailMode(detailModeYAML)
	}
	var cmd tea.Cmd
	m.dt.vp, cmd = m.dt.vp.Update(msg)
	return m, cmd
}

func (m Model) renderDetailOverlay() string {
	modeLabel := "Describe"
	if m.dt.mode == detailModeYAML {
		modeLabel = "YAML"
	}
	title := styles.Title.Render(fmt.Sprintf("%s: %s", m.dt.kind.Title(), m.dt.name)) +
		styles.Muted.Render("  ["+modeLabel+"]")

	var body string
	switch {
	case m.dt.loading:
		body = styles.Muted.Render("loading…")
	case m.dt.err != "":
		body = styles.ErrorText.Render(m.dt.err)
	default:
		body = m.dt.vp.View()
	}
	footer := styles.StatusBar.Render("d:describe  y:yaml  ↑/↓/pgup/pgdn scroll · esc close")
	return styles.Border.Padding(1, 2).Render(title + "\n\n" + body + "\n" + footer)
}
