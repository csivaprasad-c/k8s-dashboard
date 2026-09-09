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
	KindDaemonSets
	KindStatefulSets
	KindJobs
	KindCronJobs
	KindServices
	KindIngress
	KindNetworkPolicies
	KindServiceAccounts
	KindRoles
	KindRoleBindings
	KindPV
	KindPVC
	KindStorageClasses
	KindResourceQuotas
	KindLimitRanges
	KindHPA
	KindVPA
	KindCRDs
	KindConfig // ConfigMaps + Secrets, listed together
	KindEvents
)

// Kinds is the fixed tab order shown in the dashboard header. There are too
// many to bind one key per tab (see the ":" jump-to-resource overlay and
// Tab/Shift+Tab cycling in the dashboard).
var Kinds = []Kind{
	KindOverview, KindNodes, KindPods, KindDeployments, KindDaemonSets, KindStatefulSets,
	KindJobs, KindCronJobs, KindServices, KindIngress, KindNetworkPolicies,
	KindServiceAccounts, KindRoles, KindRoleBindings,
	KindPV, KindPVC, KindStorageClasses, KindResourceQuotas, KindLimitRanges,
	KindHPA, KindVPA, KindCRDs, KindConfig, KindEvents,
}

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
	case KindDaemonSets:
		return "DaemonSet"
	case KindStatefulSets:
		return "StatefulSet"
	case KindJobs:
		return "Job"
	case KindCronJobs:
		return "CronJob"
	case KindServices:
		return "Svc"
	case KindIngress:
		return "Ingress"
	case KindNetworkPolicies:
		return "NetPol"
	case KindServiceAccounts:
		return "SvcAccount"
	case KindRoles:
		return "Role"
	case KindRoleBindings:
		return "RoleBinding"
	case KindPV:
		return "PV"
	case KindPVC:
		return "PVC"
	case KindStorageClasses:
		return "StorageClass"
	case KindResourceQuotas:
		return "ResourceQuota"
	case KindLimitRanges:
		return "LimitRange"
	case KindHPA:
		return "HPA"
	case KindVPA:
		return "VPA"
	case KindCRDs:
		return "CRD"
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
	switch k {
	case KindOverview, KindNodes, KindPV, KindStorageClasses, KindCRDs:
		return false
	default:
		return true
	}
}

// Columns returns the table header for a kind's row data.
func (k Kind) Columns() []string {
	switch k {
	case KindNodes:
		return []string{"NAME", "STATUS", "ROLES", "CPU", "MEMORY", "VERSION", "AGE"}
	case KindPods:
		return []string{"NAME", "READY", "STATUS", "RESTARTS", "AGE"}
	case KindDeployments:
		return []string{"NAME", "READY", "UP-TO-DATE", "AVAILABLE", "AGE"}
	case KindDaemonSets:
		return []string{"NAME", "DESIRED", "CURRENT", "READY", "UP-TO-DATE", "AVAILABLE", "AGE"}
	case KindStatefulSets:
		return []string{"NAME", "READY", "AGE"}
	case KindJobs:
		return []string{"NAME", "COMPLETIONS", "DURATION", "AGE"}
	case KindCronJobs:
		return []string{"NAME", "SCHEDULE", "SUSPEND", "ACTIVE", "LAST SCHEDULE", "AGE"}
	case KindServices:
		return []string{"NAME", "TYPE", "CLUSTER-IP", "EXTERNAL-IP", "PORT(S)", "AGE"}
	case KindIngress:
		return []string{"NAME", "CLASS", "HOSTS", "ADDRESS", "PORTS", "AGE"}
	case KindNetworkPolicies:
		return []string{"NAME", "POD-SELECTOR", "AGE"}
	case KindServiceAccounts:
		return []string{"NAME", "SECRETS", "AGE"}
	case KindRoles:
		return []string{"NAME", "RULES", "AGE"}
	case KindRoleBindings:
		return []string{"NAME", "ROLE", "SUBJECTS", "AGE"}
	case KindPV:
		return []string{"NAME", "CAPACITY", "ACCESS MODES", "STATUS", "CLAIM", "STORAGECLASS", "AGE"}
	case KindPVC:
		return []string{"NAME", "STATUS", "VOLUME", "CAPACITY", "ACCESS MODES", "STORAGECLASS", "AGE"}
	case KindStorageClasses:
		return []string{"NAME", "PROVISIONER", "RECLAIMPOLICY", "VOLUMEBINDINGMODE", "ALLOWEXPANSION", "AGE"}
	case KindResourceQuotas:
		return []string{"NAME", "RESOURCES (USED/HARD)", "AGE"}
	case KindLimitRanges:
		return []string{"NAME", "LIMITS", "AGE"}
	case KindHPA:
		return []string{"NAME", "REFERENCE", "TARGETS", "MINPODS", "MAXPODS", "REPLICAS", "AGE"}
	case KindVPA:
		return []string{"NAME", "MODE", "TARGET", "CPU", "MEMORY", "AGE"}
	case KindCRDs:
		return []string{"NAME", "GROUP", "VERSIONS", "KIND", "SCOPE", "AGE"}
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
