package dashboard

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/csivaprasad/k8s-dashboard/internal/k8sres"
	"github.com/csivaprasad/k8s-dashboard/internal/ui/styles"
)

type deleteState struct {
	kind      k8sres.Kind
	name      string
	namespace string
	deleting  bool
	err       string
}

var deletableKinds = map[k8sres.Kind]bool{
	k8sres.KindPods:            true,
	k8sres.KindDeployments:     true,
	k8sres.KindDaemonSets:      true,
	k8sres.KindStatefulSets:    true,
	k8sres.KindJobs:            true,
	k8sres.KindCronJobs:        true,
	k8sres.KindServices:        true,
	k8sres.KindIngress:         true,
	k8sres.KindNetworkPolicies: true,
	k8sres.KindServiceAccounts: true,
	k8sres.KindRoles:           true,
	k8sres.KindRoleBindings:    true,
	k8sres.KindPV:              true,
	k8sres.KindPVC:             true,
	k8sres.KindStorageClasses:  true,
	k8sres.KindResourceQuotas:  true,
	k8sres.KindLimitRanges:     true,
	k8sres.KindHPA:             true,
	k8sres.KindVPA:             true,
	k8sres.KindCRDs:            true,
	k8sres.KindConfig:          true,
}

// deleteWarnings adds an extra line to the confirm prompt for kinds whose
// blast radius isn't obvious from the name alone.
var deleteWarnings = map[k8sres.Kind]string{
	k8sres.KindCRDs: "This deletes every custom resource of this type, cluster-wide.",
}

func (m Model) openDeleteConfirm() (Model, tea.Cmd) {
	kind := m.currentKind()
	if !deletableKinds[kind] {
		return m, nil
	}
	row, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	m.active = overlayDeleteConfirm
	m.dl = deleteState{kind: kind, name: row.Name, namespace: row.Namespace}
	return m, nil
}

func (m Model) updateDeleteKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.dl.deleting {
		return m, nil
	}
	switch msg.String() {
	case "y":
		m.dl.deleting = true
		kind, ns, name := m.dl.kind, m.dl.namespace, m.dl.name
		clients := m.clients()
		return m, func() tea.Msg {
			err := k8sres.Delete(clients, kind, ns, name)
			return deleteResultMsg{err: err}
		}
	case "n", "esc":
		m.active = overlayNone
		return m, nil
	}
	return m, nil
}

func (m Model) handleDeleteResult(msg deleteResultMsg) (Model, tea.Cmd) {
	if msg.err != nil {
		m.dl.deleting = false
		m.dl.err = msg.err.Error()
		return m, nil
	}
	m.active = overlayNone
	m.loading = true
	return m, m.loadCmd()
}

func (m Model) renderDeleteOverlay() string {
	title := styles.ErrorText.Render("Delete " + m.dl.kind.Title() + "?")
	body := fmt.Sprintf("%s/%s", m.dl.namespace, m.dl.name)
	if m.dl.namespace == "" {
		body = m.dl.name
	}
	warning := ""
	if w, ok := deleteWarnings[m.dl.kind]; ok {
		warning = "\n\n" + styles.ErrorText.Render(w)
	}
	status := ""
	switch {
	case m.dl.deleting:
		status = "\n\n" + styles.Muted.Render("deleting…")
	case m.dl.err != "":
		status = "\n\n" + styles.ErrorText.Render(m.dl.err)
	}
	footer := "\n\n" + styles.StatusBar.Render("y confirm · n/esc cancel")
	return styles.Border.Padding(1, 2).Render(title + "\n\n" + body + warning + status + footer)
}
