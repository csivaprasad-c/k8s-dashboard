package dashboard

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/csivaprasad/k8s-dashboard/internal/k8sres"
	"github.com/csivaprasad/k8s-dashboard/internal/ui/styles"
)

type detailState struct {
	kind      k8sres.Kind
	name      string
	namespace string
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
	m.dt = detailState{kind: kind, name: row.Name, namespace: row.Namespace, loading: true, vp: m.dt.vp}
	m.dt.vp.SetContent("")
	m.dt.vp.GotoTop()

	clients := m.clients()
	return m, func() tea.Msg {
		content, err := k8sres.GetYAML(clients, kind, row.Namespace, row.Name)
		return yamlLoadedMsg{content: content, err: err}
	}
}

func (m Model) handleYAMLLoaded(msg yamlLoadedMsg) (Model, tea.Cmd) {
	if m.active != overlayDetail {
		return m, nil
	}
	m.dt.loading = false
	if msg.err != nil {
		m.dt.err = msg.err.Error()
		return m, nil
	}
	m.dt.err = ""
	m.dt.vp.SetContent(msg.content)
	return m, nil
}

func (m Model) updateDetailKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.active = overlayNone
		return m, nil
	}
	var cmd tea.Cmd
	m.dt.vp, cmd = m.dt.vp.Update(msg)
	return m, cmd
}

func (m Model) renderDetailOverlay() string {
	title := styles.Title.Render(fmt.Sprintf("%s: %s", m.dt.kind.Title(), m.dt.name))
	var body string
	switch {
	case m.dt.loading:
		body = styles.Muted.Render("loading…")
	case m.dt.err != "":
		body = styles.ErrorText.Render(m.dt.err)
	default:
		body = m.dt.vp.View()
	}
	footer := styles.StatusBar.Render("↑/↓/pgup/pgdn scroll · esc close")
	return styles.Border.Padding(1, 2).Render(title + "\n\n" + body + "\n" + footer)
}
