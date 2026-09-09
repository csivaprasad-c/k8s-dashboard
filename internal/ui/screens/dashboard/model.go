// Package dashboard is the main cluster view: resource tabs, a live(ish)
// table, and overlays for namespace switching, YAML detail, logs, delete
// confirmation, and a help cheat-sheet.
package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/csivaprasad/k8s-dashboard/internal/k8sres"
	"github.com/csivaprasad/k8s-dashboard/internal/kube"
	"github.com/csivaprasad/k8s-dashboard/internal/ui/styles"
)

const refreshInterval = 5 * time.Second

type overlay int

const (
	overlayNone overlay = iota
	overlayNamespace
	overlayDetail
	overlayLogs
	overlayHelp
	overlayDeleteConfirm
)

type Model struct {
	session *kube.Session
	version string

	width, height int

	kindIdx   int // index into k8sres.Kinds
	namespace string

	rows       []k8sres.Row // last loaded, unfiltered
	tbl        table.Model
	loading    bool
	loadErr    string
	lastLoaded time.Time

	filtering   bool
	filterInput textinput.Model

	overview    k8sres.Overview
	overviewErr string

	active overlay

	ns namespaceState
	dt detailState
	lg logsState
	dl deleteState
}

func New(session *kube.Session, version string) Model {
	fi := textinput.New()
	fi.Prompt = "/"
	fi.CharLimit = 64

	m := Model{
		session:     session,
		version:     version,
		namespace:   session.Context.Namespace,
		filterInput: fi,
		tbl:         table.New(table.WithFocused(true)),
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadCmd(), tea.Tick(refreshInterval, func(time.Time) tea.Msg { return refreshTickMsg{} }))
}

func (m *Model) SetSize(w, h int) {
	m.width, m.height = w, h
	bodyHeight := h - 6
	if bodyHeight < 3 {
		bodyHeight = 3
	}
	m.tbl.SetHeight(bodyHeight)
	m.tbl.SetWidth(w)
	m.dt.vp.Width = w - 4
	m.dt.vp.Height = h - 4
	m.lg.vp.Width = w - 4
	m.lg.vp.Height = h - 6
}

func (m Model) currentKind() k8sres.Kind { return k8sres.Kinds[m.kindIdx] }

// loadCmd fetches whatever the active tab needs (overview summary or a
// resource table) for the current namespace.
func (m Model) loadCmd() tea.Cmd {
	kind := m.currentKind()
	session := m.session
	ns := m.namespace
	if kind == k8sres.KindOverview {
		return func() tea.Msg {
			ov, err := k8sres.GetOverview(session.Clientset, session.Metrics)
			return overviewLoadedMsg{overview: ov, err: err}
		}
	}
	return func() tea.Msg {
		rows, err := k8sres.List(session.Clientset, session.Metrics, kind, ns)
		return rowsLoadedMsg{kind: kind, ns: ns, rows: rows, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case refreshTickMsg:
		cmds := []tea.Cmd{tea.Tick(refreshInterval, func(time.Time) tea.Msg { return refreshTickMsg{} })}
		if m.active == overlayNone {
			cmds = append(cmds, m.loadCmd())
		}
		return m, tea.Batch(cmds...)

	case rowsLoadedMsg:
		if msg.kind != m.currentKind() || msg.ns != m.namespace {
			return m, nil // stale response from a since-changed tab/namespace
		}
		m.loading = false
		if msg.err != nil {
			m.loadErr = msg.err.Error()
			return m, nil
		}
		m.loadErr = ""
		m.rows = msg.rows
		m.lastLoaded = time.Now()
		m.rebuildTable()
		return m, nil

	case overviewLoadedMsg:
		if msg.err != nil {
			m.overviewErr = msg.err.Error()
			return m, nil
		}
		m.overviewErr = ""
		m.overview = msg.overview
		m.lastLoaded = time.Now()
		return m, nil

	case namespacesLoadedMsg:
		return m.handleNamespacesLoaded(msg)

	case yamlLoadedMsg:
		return m.handleYAMLLoaded(msg)

	case deleteResultMsg:
		return m.handleDeleteResult(msg)

	case containersLoadedMsg:
		return m.handleContainersLoaded(msg)

	case logsStartedMsg:
		return m.handleLogsStarted(msg)

	case logLineMsg:
		return m.handleLogLine(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch m.active {
	case overlayNamespace:
		return m.updateNamespaceKey(msg)
	case overlayDetail:
		return m.updateDetailKey(msg)
	case overlayLogs:
		return m.updateLogsKey(msg)
	case overlayDeleteConfirm:
		return m.updateDeleteKey(msg)
	case overlayHelp:
		m.active = overlayNone
		return m, nil
	}

	if m.filtering {
		switch msg.String() {
		case "esc":
			m.filtering = false
			m.filterInput.Blur()
			m.filterInput.SetValue("")
			m.rebuildTable()
			return m, nil
		case "enter":
			m.filtering = false
			m.filterInput.Blur()
			return m, nil
		}
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		m.rebuildTable()
		return m, cmd
	}

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		return m, tea.Quit
	case "ctrl+k":
		return m, func() tea.Msg { return BackToClusterSelectMsg{} }
	case "?":
		m.active = overlayHelp
		return m, nil
	case "r":
		m.loading = true
		return m, m.loadCmd()
	case "n":
		return m.openNamespacePicker()
	case "/":
		if m.currentKind() != k8sres.KindOverview {
			m.filtering = true
			m.filterInput.Focus()
			return m, textinput.Blink
		}
	case "1", "2", "3", "4", "5", "6", "7", "8", "9", "0":
		// Keys 1-9 select tabs 1-9; 0 selects the 10th tab, matching the
		// numbering shown in the tab bar (see tabKeyLabel).
		idx := 9
		if msg.String() != "0" {
			idx = int(msg.String()[0] - '1')
		}
		if idx < len(k8sres.Kinds) {
			m.kindIdx = idx
			m.filterInput.SetValue("")
			m.loading = true
			m.loadErr = ""
			return m, m.loadCmd()
		}
	case "tab":
		m.kindIdx = (m.kindIdx + 1) % len(k8sres.Kinds)
		m.loading = true
		m.loadErr = ""
		return m, m.loadCmd()
	case "shift+tab":
		m.kindIdx = (m.kindIdx - 1 + len(k8sres.Kinds)) % len(k8sres.Kinds)
		m.loading = true
		m.loadErr = ""
		return m, m.loadCmd()
	case "up", "down":
		var cmd tea.Cmd
		m.tbl, cmd = m.tbl.Update(msg)
		return m, cmd
	case "enter":
		return m.openDetail()
	case "l":
		if m.currentKind() == k8sres.KindPods {
			return m.openLogs()
		}
	case "x":
		return m.openDeleteConfirm()
	}
	return m, nil
}

func (m Model) View() string {
	header := m.renderHeader()
	tabs := m.renderTabs()
	body := m.renderBody()
	footer := m.renderFooter()

	base := lipglossJoin(header, tabs, body, footer)
	switch m.active {
	case overlayNamespace:
		return overlayOn(base, m.renderNamespaceOverlay(), m.width, m.height)
	case overlayDetail:
		return overlayOn(base, m.renderDetailOverlay(), m.width, m.height)
	case overlayLogs:
		return overlayOn(base, m.renderLogsOverlay(), m.width, m.height)
	case overlayDeleteConfirm:
		return overlayOn(base, m.renderDeleteOverlay(), m.width, m.height)
	case overlayHelp:
		return overlayOn(base, renderHelp(), m.width, m.height)
	}
	return base
}

func (m Model) renderHeader() string {
	dot := styles.StatusStyle("ok").Render("●")
	txt := fmt.Sprintf("%s │ ctx: %s │ ns: %s │ %s │ %s connected",
		m.session.Context.Cluster, m.session.Context.Name, m.namespace, m.version, dot)
	return styles.HeaderBar.Width(max0(m.width)).Render(txt)
}

// tabKeyLabel is the key that selects the tab at index i: "1"-"9" for the
// first nine tabs, then "0" for a tenth (see the "1".."9","0" key handling
// in handleKey).
func tabKeyLabel(i int) string {
	if i < 9 {
		return fmt.Sprintf("%d", i+1)
	}
	return "0"
}

func (m Model) renderTabs() string {
	var b strings.Builder
	for i, k := range k8sres.Kinds {
		label := tabKeyLabel(i) + ":" + k.Title()
		if i == m.kindIdx {
			b.WriteString(styles.TabActive.Render(label))
		} else {
			b.WriteString(styles.TabInactive.Render(label))
		}
	}
	return b.String()
}

func (m Model) renderFooter() string {
	if m.loadErr != "" {
		return styles.ErrorText.Render("error: " + m.loadErr)
	}
	if m.filtering {
		return m.filterInput.View()
	}
	status := styles.Muted.Render(fmt.Sprintf("updated %s ago", roundSeconds(time.Since(m.lastLoaded))))
	keys := "  /:filter  n:namespace  enter:yaml  l:logs(pods)  x:delete  r:refresh  ctrl+k:cluster  ?:help  q:quit"
	return status + styles.StatusBar.Render(keys)
}

func roundSeconds(d time.Duration) string {
	if d < time.Second {
		return "0s"
	}
	return d.Round(time.Second).String()
}

func max0(w int) int {
	if w < 0 {
		return 0
	}
	return w
}
