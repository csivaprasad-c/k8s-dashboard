// Package connecting shows a spinner while dialing the chosen cluster and
// verifying it's actually reachable before handing off to the dashboard.
package connecting

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/csivaprasad/k8s-dashboard/internal/kube"
	"github.com/csivaprasad/k8s-dashboard/internal/ui/styles"
)

const pingTimeout = 5 * time.Second

// ConnectedMsg is emitted once the connection is verified live.
type ConnectedMsg struct {
	Session *kube.Session
	Version string
}

// FailedMsg is emitted when connecting or the reachability check fails.
type FailedMsg struct {
	ContextName string
	Err         error
}

// BackMsg is emitted when the user dismisses a failed-connection screen.
type BackMsg struct{}

type Model struct {
	target  kube.ClusterContext
	sp      spinner.Model
	failed  error
	failing bool
}

func New(target kube.ClusterContext) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return Model{target: target, sp: sp}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.sp.Tick, connectCmd(m.target))
}

func connectCmd(target kube.ClusterContext) tea.Cmd {
	return func() tea.Msg {
		session, err := kube.Connect(target.Name)
		if err != nil {
			return FailedMsg{ContextName: target.Name, Err: err}
		}
		version, err := session.Ping(pingTimeout)
		if err != nil {
			return FailedMsg{ContextName: target.Name, Err: err}
		}
		return ConnectedMsg{Session: session, Version: version}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case FailedMsg:
		m.failing = true
		m.failed = msg.Err
		return m, nil
	case tea.KeyMsg:
		if m.failing {
			return m, func() tea.Msg { return BackMsg{} }
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.sp, cmd = m.sp.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) View() string {
	if m.failing {
		return styles.ErrorText.Render(fmt.Sprintf("Failed to connect to %q", m.target.Name)) +
			"\n\n" + m.failed.Error() +
			"\n\n" + styles.Muted.Render("press any key to go back")
	}
	return fmt.Sprintf("%s Connecting to %s (%s)…", m.sp.View(), m.target.Name, m.target.Server)
}
