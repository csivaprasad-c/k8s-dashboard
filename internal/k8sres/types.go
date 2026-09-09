// Package k8sres wraps client-go calls needed to populate the dashboard:
// listing resources into table-ready rows, fetching a single object's YAML,
// and tailing pod logs.
package k8sres

// Kind identifies one of the resource views/tabs in the dashboard.
type Kind int

const (
	KindOverview Kind = iota
	KindNodes
	KindPods
	KindDeployments
	KindServices
	KindConfig // ConfigMaps + Secrets, listed together
	KindEvents
)

// Kinds is the fixed tab order shown in the dashboard header.
var Kinds = []Kind{KindOverview, KindNodes, KindPods, KindDeployments, KindServices, KindConfig, KindEvents}

// Title is the tab label for a kind.
func (k Kind) Title() string {
	switch k {
	case KindOverview:
		return "Overview"
	case KindNodes:
		return "Nodes"
	case KindPods:
		return "Pods"
	case KindDeployments:
		return "Deploy"
	case KindServices:
		return "Svc"
	case KindConfig:
		return "Cfg/Secrets"
	case KindEvents:
		return "Events"
	default:
		return "?"
	}
}

// Namespaced reports whether the kind is scoped to a namespace (true) or
// cluster-wide (false).
func (k Kind) Namespaced() bool {
	return k != KindNodes && k != KindOverview
}

// Columns returns the table header for a kind's row data.
func (k Kind) Columns() []string {
	switch k {
	case KindNodes:
		return []string{"NAME", "STATUS", "ROLES", "VERSION", "AGE"}
	case KindPods:
		return []string{"NAME", "READY", "STATUS", "RESTARTS", "AGE"}
	case KindDeployments:
		return []string{"NAME", "READY", "UP-TO-DATE", "AVAILABLE", "AGE"}
	case KindServices:
		return []string{"NAME", "TYPE", "CLUSTER-IP", "PORT(S)", "AGE"}
	case KindConfig:
		return []string{"NAME", "KIND", "DATA", "AGE"}
	case KindEvents:
		return []string{"LAST SEEN", "TYPE", "REASON", "OBJECT", "MESSAGE"}
	default:
		return nil
	}
}

// Row is one rendered table row plus the underlying object name/namespace,
// kept so detail/logs/delete actions know what they're acting on.
type Row struct {
	Cells     []string
	Name      string
	Namespace string
	// StatusClass hints at coloring the STATUS/READY-ish cell: "ok", "warn", "bad", or "" for none.
	StatusClass string
}
