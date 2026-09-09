package dashboard

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/csivaprasad/k8s-dashboard/internal/k8sres"
	"github.com/csivaprasad/k8s-dashboard/internal/ui/styles"
)

const (
	logTailLines  = 200
	logBufferCap  = 2000
	logChanBuffer = 256
)

type logsState struct {
	podName   string
	namespace string

	picking      bool
	containers   []string
	pickerCursor int

	container string
	loading   bool
	streaming bool
	ended     bool
	err       string

	lines  []string
	lineCh chan logLineMsg
	cancel context.CancelFunc
	vp     viewport.Model
}

func (m Model) openLogs() (Model, tea.Cmd) {
	row, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	m.active = overlayLogs
	m.lg = logsState{podName: row.Name, namespace: row.Namespace, loading: true, vp: m.lg.vp}
	m.lg.vp.SetContent("")

	cs := m.session.Clientset
	ns, pod := row.Namespace, row.Name
	return m, func() tea.Msg {
		names, err := k8sres.Containers(cs, ns, pod)
		return containersLoadedMsg{names: names, err: err}
	}
}

func (m Model) handleContainersLoaded(msg containersLoadedMsg) (Model, tea.Cmd) {
	if m.active != overlayLogs {
		return m, nil
	}
	m.lg.loading = false
	if msg.err != nil {
		m.lg.err = msg.err.Error()
		return m, nil
	}
	switch len(msg.names) {
	case 0:
		m.lg.err = "pod has no containers"
		return m, nil
	case 1:
		m.lg.container = msg.names[0]
		return m.startLogStream()
	default:
		m.lg.picking = true
		m.lg.containers = msg.names
		return m, nil
	}
}

func (m Model) startLogStream() (Model, tea.Cmd) {
	m.lg.picking = false
	m.lg.loading = true
	ctx, cancel := context.WithCancel(context.Background())
	cs := m.session.Clientset
	ns, pod, container := m.lg.namespace, m.lg.podName, m.lg.container
	// Stash cancel immediately so closing the overlay before the stream
	// finishes opening still stops the in-flight dial.
	m.lg.cancel = cancel
	return m, func() tea.Msg {
		reader, err := k8sres.StreamLogs(ctx, cs, ns, pod, container, logTailLines)
		if err != nil {
			cancel()
			return logsStartedMsg{err: err}
		}
		ch := make(chan logLineMsg, logChanBuffer)
		go pumpLogs(ctx, reader, ch)
		return logsStartedMsg{lines: ch, cancel: cancel}
	}
}

func pumpLogs(ctx context.Context, reader io.ReadCloser, ch chan logLineMsg) {
	defer reader.Close()
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		select {
		case ch <- logLineMsg{line: scanner.Text()}:
		case <-ctx.Done():
			return
		}
	}
	final := logLineMsg{done: true}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		final = logLineMsg{err: err}
	}
	select {
	case ch <- final:
	case <-ctx.Done():
	}
}

func waitForLogLine(ch chan logLineMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return logLineMsg{done: true}
		}
		return msg
	}
}

func (m Model) handleLogsStarted(msg logsStartedMsg) (Model, tea.Cmd) {
	if m.active != overlayLogs {
		if msg.cancel != nil {
			msg.cancel()
		}
		return m, nil
	}
	m.lg.loading = false
	if msg.err != nil {
		m.lg.err = msg.err.Error()
		return m, nil
	}
	m.lg.streaming = true
	m.lg.lineCh = msg.lines
	m.lg.cancel = msg.cancel
	return m, waitForLogLine(msg.lines)
}

func (m Model) handleLogLine(msg logLineMsg) (Model, tea.Cmd) {
	if m.active != overlayLogs || !m.lg.streaming {
		return m, nil
	}
	if msg.err != nil {
		m.lg.err = msg.err.Error()
		m.lg.streaming = false
		return m, nil
	}
	if msg.done {
		m.lg.ended = true
		m.lg.streaming = false
		return m, nil
	}
	m.lg.lines = append(m.lg.lines, msg.line)
	if len(m.lg.lines) > logBufferCap {
		m.lg.lines = m.lg.lines[len(m.lg.lines)-logBufferCap:]
	}
	m.lg.vp.SetContent(strings.Join(m.lg.lines, "\n"))
	m.lg.vp.GotoBottom()
	return m, waitForLogLine(m.lg.lineCh)
}

func (m Model) closeLogs() Model {
	if m.lg.cancel != nil {
		m.lg.cancel()
	}
	m.active = overlayNone
	return m
}

func (m Model) updateLogsKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.lg.picking {
		switch msg.String() {
		case "esc":
			return m.closeLogs(), nil
		case "up":
			if m.lg.pickerCursor > 0 {
				m.lg.pickerCursor--
			}
			return m, nil
		case "down":
			if m.lg.pickerCursor < len(m.lg.containers)-1 {
				m.lg.pickerCursor++
			}
			return m, nil
		case "enter":
			m.lg.container = m.lg.containers[m.lg.pickerCursor]
			return m.startLogStream()
		}
		return m, nil
	}

	switch msg.String() {
	case "esc", "q":
		return m.closeLogs(), nil
	}
	var cmd tea.Cmd
	m.lg.vp, cmd = m.lg.vp.Update(msg)
	return m, cmd
}

func (m Model) renderLogsOverlay() string {
	title := styles.Title.Render(fmt.Sprintf("Logs: %s/%s", m.lg.namespace, m.lg.podName))
	if m.lg.container != "" {
		title += styles.Muted.Render(" [" + m.lg.container + "]")
	}

	var body string
	switch {
	case m.lg.picking:
		var b strings.Builder
		b.WriteString("Select a container:\n\n")
		for i, c := range m.lg.containers {
			cursor := "  "
			line := c
			if i == m.lg.pickerCursor {
				cursor = "▸ "
				line = styles.SelectedRow.Render(line)
			}
			b.WriteString(cursor + line + "\n")
		}
		body = b.String()
	case m.lg.loading:
		body = styles.Muted.Render("loading…")
	case m.lg.err != "":
		body = styles.ErrorText.Render(m.lg.err)
	default:
		body = m.lg.vp.View()
		if m.lg.ended {
			body += "\n" + styles.Muted.Render("[stream ended]")
		}
	}

	footer := styles.StatusBar.Render("↑/↓/pgup/pgdn scroll · esc close")
	return styles.Border.Padding(1, 2).Render(title + "\n\n" + body + "\n" + footer)
}
