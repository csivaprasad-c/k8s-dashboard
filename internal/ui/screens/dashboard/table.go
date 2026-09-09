package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/csivaprasad/k8s-dashboard/internal/k8sres"
	"github.com/csivaprasad/k8s-dashboard/internal/ui/styles"
)

// visibleRows applies the active filter (substring match on the row name)
// to the last-loaded rows.
func (m Model) visibleRows() []k8sres.Row {
	q := strings.ToLower(strings.TrimSpace(m.filterInput.Value()))
	if q == "" {
		return m.rows
	}
	out := make([]k8sres.Row, 0, len(m.rows))
	for _, r := range m.rows {
		if strings.Contains(strings.ToLower(r.Name), q) {
			out = append(out, r)
		}
	}
	return out
}

// rebuildTable recomputes column widths and rows for the bubbles/table
// widget from the current (filtered) rows.
func (m *Model) rebuildTable() {
	kind := m.currentKind()
	if kind == k8sres.KindOverview {
		return
	}
	headers := kind.Columns()
	visible := m.visibleRows()

	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, r := range visible {
		for i, c := range r.Cells {
			if i < len(widths) && len(c) > widths[i] {
				widths[i] = len(c)
			}
		}
	}
	// The last column (usually message/free text) absorbs remaining width.
	total := 0
	for _, w := range widths {
		total += w + 2
	}
	if slack := m.width - total; slack > 0 && len(widths) > 0 {
		widths[len(widths)-1] += slack
	}

	cols := make([]table.Column, len(headers))
	for i, h := range headers {
		cols[i] = table.Column{Title: h, Width: widths[i]}
	}

	rows := make([]table.Row, len(visible))
	for i, r := range visible {
		rows[i] = table.Row(r.Cells)
	}

	m.tbl.SetColumns(cols)
	m.tbl.SetRows(rows)
	// bubbles/table clamps the cursor down to -1 when a list goes empty
	// (e.g. switching to a namespace with no pods) but never restores it
	// to 0 once rows reappear — do that ourselves so selection/enter/x
	// keep working after a list empties and refills.
	if len(rows) > 0 && m.tbl.Cursor() < 0 {
		m.tbl.SetCursor(0)
	}
}

// selectedRow returns the row under the cursor, if any.
func (m Model) selectedRow() (k8sres.Row, bool) {
	visible := m.visibleRows()
	idx := m.tbl.Cursor()
	if idx < 0 || idx >= len(visible) {
		return k8sres.Row{}, false
	}
	return visible[idx], true
}

func (m Model) renderBody() string {
	if m.loading && len(m.rows) == 0 {
		return styles.Muted.Render("loading…")
	}
	if m.currentKind() == k8sres.KindOverview {
		return m.renderOverview()
	}
	return m.tbl.View()
}

func (m Model) renderOverview() string {
	if m.overviewErr != "" {
		return styles.ErrorText.Render("error: " + m.overviewErr)
	}
	o := m.overview
	line := func(label string, val string) string {
		return fmt.Sprintf("%-14s %s", label, val)
	}
	podHealth := fmt.Sprintf("%d running · %d pending · %d failed", o.PodsRunning, o.PodsPending, o.PodsFailed)
	if o.PodsFailed > 0 {
		podHealth = styles.StatusStyle("bad").Render(podHealth)
	} else if o.PodsPending > 0 {
		podHealth = styles.StatusStyle("warn").Render(podHealth)
	} else {
		podHealth = styles.StatusStyle("ok").Render(podHealth)
	}

	nodeHealth := fmt.Sprintf("%d/%d ready", o.NodesReady, o.NodeCount)
	nodeClass := "ok"
	if o.NodesReady < o.NodeCount {
		nodeClass = "bad"
	}
	nodeHealth = styles.StatusStyle(nodeClass).Render(nodeHealth)

	depHealth := fmt.Sprintf("%d/%d fully rolled out", o.DeploymentCount-o.DeploymentsBad, o.DeploymentCount)
	depClass := "ok"
	if o.DeploymentsBad > 0 {
		depClass = "warn"
	}
	depHealth = styles.StatusStyle(depClass).Render(depHealth)

	rows := strings.Join([]string{
		line("Nodes:", nodeHealth),
		line("Namespaces:", fmt.Sprintf("%d", o.NamespaceCount)),
		line("Pods:", fmt.Sprintf("%d total — %s", o.PodCount, podHealth)),
		line("Deployments:", fmt.Sprintf("%d total — %s", o.DeploymentCount, depHealth)),
	}, "\n")

	return styles.Border.Padding(1, 2).Render(
		lipgloss.JoinVertical(lipgloss.Left, styles.Title.Render("Cluster Overview"), "", rows))
}
