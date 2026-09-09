// Package clusterselect implements the first screen: pick a kubeconfig
// context to connect to.
package clusterselect

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/csivaprasad/k8s-dashboard/internal/kube"
	"github.com/csivaprasad/k8s-dashboard/internal/ui/styles"
)

// ContextChosenMsg is emitted when the user picks a context to connect to.
type ContextChosenMsg struct {
	Context kube.ClusterContext
}

// QuitMsg is emitted when the user quits from this screen.
type QuitMsg struct{}

type Model struct {
	all      []kube.ClusterContext
	filtered []kube.ClusterContext
	cursor   int
	filter   textinput.Model
	loadErr  error
	width    int
	height   int
}

func New() Model {
	ti := textinput.New()
	ti.Placeholder = "type to filter…"
	ti.Prompt = "Filter: "
	ti.CharLimit = 64

	m := Model{filter: ti}
	contexts, err := kube.LoadContexts()
	if err != nil {
		m.loadErr = err
		return m
	}
	m.all = contexts
	m.filtered = contexts
	return m
}

func (m Model) Init() tea.Cmd { return nil }

func (m *Model) SetSize(w, h int) {
	m.width, m.height = w, h
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			if !m.filter.Focused() {
				return m, func() tea.Msg { return QuitMsg{} }
			}
		case "esc":
			if m.filter.Focused() {
				m.filter.Blur()
				return m, nil
			}
			return m, func() tea.Msg { return QuitMsg{} }
		case "/":
			if !m.filter.Focused() {
				m.filter.Focus()
				return m, textinput.Blink
			}
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		case "enter":
			if len(m.filtered) == 0 {
				return m, nil
			}
			chosen := m.filtered[m.cursor]
			return m, func() tea.Msg { return ContextChosenMsg{Context: chosen} }
		}
	}

	if m.filter.Focused() {
		var cmd tea.Cmd
		prev := m.filter.Value()
		m.filter, cmd = m.filter.Update(msg)
		if m.filter.Value() != prev {
			m.applyFilter()
		}
		return m, cmd
	}
	return m, nil
}

func (m *Model) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	if q == "" {
		m.filtered = m.all
	} else {
		matches := make([]kube.ClusterContext, 0, len(m.all))
		for _, c := range m.all {
			if strings.Contains(strings.ToLower(c.Name), q) || strings.Contains(strings.ToLower(c.Cluster), q) {
				matches = append(matches, c)
			}
		}
		m.filtered = matches
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m Model) View() string {
	if m.loadErr != nil {
		return styles.ErrorText.Render(fmt.Sprintf("Could not load kubeconfig: %v", m.loadErr)) +
			"\n\n" + styles.Muted.Render("press q to quit")
	}
	if len(m.all) == 0 {
		return styles.Muted.Render("No clusters found in kubeconfig.") + "\n\n" + styles.Muted.Render("press q to quit")
	}

	var b strings.Builder
	b.WriteString(styles.Title.Render("Select a cluster"))
	b.WriteString("\n\n")
	b.WriteString(m.filter.View())
	b.WriteString("\n\n")

	nameW, serverW := 0, 0
	for _, c := range m.filtered {
		if len(c.Name) > nameW {
			nameW = len(c.Name)
		}
		if len(c.Server) > serverW {
			serverW = len(c.Server)
		}
	}

	if len(m.filtered) == 0 {
		b.WriteString(styles.Muted.Render("  no matches"))
	}
	for i, c := range m.filtered {
		cursor := "  "
		lineStyle := lipgloss.NewStyle()
		if i == m.cursor {
			cursor = "▸ "
			lineStyle = styles.SelectedRow
		}
		marker := " "
		if c.Current {
			marker = styles.StatusStyle("ok").Render("●")
		}
		line := fmt.Sprintf("%-*s  %-*s  ns:%-12s %s", nameW, c.Name, serverW, c.Server, c.Namespace, marker)
		b.WriteString(cursor + lineStyle.Render(line) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.StatusBar.Render("↑/↓ move · / filter · enter connect · q quit"))
	return b.String()
}
