package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/csivaprasad-c/k8s-dashboard/internal/k8sres"
	"github.com/csivaprasad-c/k8s-dashboard/internal/ui/styles"
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

	// SetColumns/SetRows each immediately re-render every current row
	// against whichever of (cols, rows) hasn't been updated yet — if the
	// new column count differs from the old (e.g. Pods' 5 columns ->
	// Config's 4), that intermediate render indexes past the shorter
	// side and panics. Clear rows first so both calls only ever render
	// against a matching pair.
	m.tbl.SetRows(nil)
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

const usageBarWidth = 24

func (m Model) renderOverview() string {
	if m.overviewErr != "" {
		return styles.ErrorText.Render("error: " + m.overviewErr)
	}
	o := m.overview
	line := func(label string, val string) string {
		return fmt.Sprintf("%-14s %s", label, val)
	}

	health := o.Health()
	healthLine := styles.StatusStyle(health.Class()).Bold(true).Render("● " + health.String())

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

	rsHealth := fmt.Sprintf("%d/%d healthy", o.ReplicaSetCount-o.ReplicaSetsBad, o.ReplicaSetCount)
	rsClass := "ok"
	if o.ReplicaSetsBad > 0 {
		rsClass = "warn"
	}
	rsHealth = styles.StatusStyle(rsClass).Render(rsHealth)

	rows := strings.Join([]string{
		line("Health:", healthLine),
		line("Nodes:", nodeHealth),
		line("Namespaces:", fmt.Sprintf("%d", o.NamespaceCount)),
		line("Pods:", fmt.Sprintf("%d total — %s", o.PodCount, podHealth)),
		line("Deployments:", fmt.Sprintf("%d total — %s", o.DeploymentCount, depHealth)),
		line("ReplicaSets:", fmt.Sprintf("%d active — %s", o.ReplicaSetCount, rsHealth)),
		"",
		line("CPU:", usageLine(o.CPUUsageMilli, o.CPUCapacityMilli, o.MetricsAvailable, o.MetricsErr, formatCores)),
		line("Memory:", usageLine(o.MemUsageBytes, o.MemCapacityBytes, o.MetricsAvailable, o.MetricsErr, formatBytes)),
	}, "\n")

	body := lipgloss.JoinVertical(lipgloss.Left, styles.Title.Render("Cluster Overview"), "", rows)
	if len(o.Issues) > 0 {
		body = lipgloss.JoinVertical(lipgloss.Left, body, "", renderIssuesBlock(o))
	}

	return styles.Border.Padding(1, 2).Render(body)
}

// renderIssuesBlock lists the same capped/prioritized issues shown in the
// always-visible summary strip (see renderSummaryBar), in full one-per-line
// form for the Overview tab's roomier body.
func renderIssuesBlock(o k8sres.Overview) string {
	title := styles.ErrorText.Render(fmt.Sprintf("Issues (%d):", o.IssuesTotal))
	lines := make([]string, 0, len(o.Issues)+2)
	lines = append(lines, title)
	for _, issue := range o.Issues {
		lines = append(lines, "  "+issue)
	}
	if o.IssuesTotal > len(o.Issues) {
		lines = append(lines, styles.Muted.Render(fmt.Sprintf("  … and %d more", o.IssuesTotal-len(o.Issues))))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// usageLine renders one CPU/memory row: a colored usage bar plus
// "used / capacity (pct%)", or a muted explanation when metrics-server
// (the metrics.k8s.io API) isn't available to report usage.
func usageLine(usage, capacity int64, available bool, unavailableReason string, format func(int64) string) string {
	if !available || capacity <= 0 {
		reason := unavailableReason
		if reason == "" {
			reason = "unavailable"
		}
		return styles.Muted.Render(fmt.Sprintf("capacity %s · usage %s", format(capacity), reason))
	}
	pct := float64(usage) / float64(capacity) * 100
	return fmt.Sprintf("%s  %s / %s (%.0f%%)", usageBar(pct), format(usage), format(capacity), pct)
}

func usageClass(pct float64) string {
	switch {
	case pct >= 90:
		return "bad"
	case pct >= 70:
		return "warn"
	default:
		return "ok"
	}
}

func usageBar(pct float64) string {
	filled := int(pct / 100 * usageBarWidth)
	filled = max(0, min(filled, usageBarWidth))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", usageBarWidth-filled)
	return styles.StatusStyle(usageClass(pct)).Render(bar)
}

func formatCores(milli int64) string {
	return fmt.Sprintf("%.2f cores", float64(milli)/1000)
}

// formatBytes renders a byte count using binary (Ki/Mi/Gi) units, matching
// kubectl's convention for resource quantities.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
