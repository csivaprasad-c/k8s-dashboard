package dashboard

import (
	"context"

	"github.com/csivaprasad-c/k8s-dashboard/internal/k8sres"
)

// BackToClusterSelectMsg is emitted on Ctrl+K to hop back to the cluster
// picker without quitting the app.
type BackToClusterSelectMsg struct{}

type rowsLoadedMsg struct {
	kind k8sres.Kind
	ns   string
	rows []k8sres.Row
	err  error
}

type overviewLoadedMsg struct {
	overview k8sres.Overview
	err      error
}

type namespacesLoadedMsg struct {
	names []string
	err   error
}

type yamlLoadedMsg struct {
	content string
	err     error
}

type deleteResultMsg struct {
	err error
}

type refreshTickMsg struct{}

type containersLoadedMsg struct {
	names []string
	err   error
}

type logLineMsg struct {
	line string
	done bool
	err  error
}

type logsStartedMsg struct {
	lines  chan logLineMsg
	cancel context.CancelFunc
	err    error
}
