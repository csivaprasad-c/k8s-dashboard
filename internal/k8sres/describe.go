package k8sres

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubectl/pkg/describe"
)

// describerGroupKind maps a dashboard Kind to the Kubernetes GroupKind
// kubectl's built-in describers are registered under (see
// k8s.io/kubectl/pkg/describe.DescriberFor). Kinds absent from this map —
// Events, VPA, CRD — have no typed describer in kubectl either; Describe
// falls back to YAML for those, same constraint real kubectl has.
func describerGroupKind(kind Kind) (schema.GroupKind, bool) {
	switch kind {
	case KindNodes:
		return schema.GroupKind{Kind: "Node"}, true
	case KindPods:
		return schema.GroupKind{Kind: "Pod"}, true
	case KindDeployments:
		return schema.GroupKind{Group: "apps", Kind: "Deployment"}, true
	case KindDaemonSets:
		return schema.GroupKind{Group: "apps", Kind: "DaemonSet"}, true
	case KindStatefulSets:
		return schema.GroupKind{Group: "apps", Kind: "StatefulSet"}, true
	case KindJobs:
		return schema.GroupKind{Group: "batch", Kind: "Job"}, true
	case KindCronJobs:
		return schema.GroupKind{Group: "batch", Kind: "CronJob"}, true
	case KindServices:
		return schema.GroupKind{Kind: "Service"}, true
	case KindIngress:
		return schema.GroupKind{Group: "networking.k8s.io", Kind: "Ingress"}, true
	case KindNetworkPolicies:
		return schema.GroupKind{Group: "networking.k8s.io", Kind: "NetworkPolicy"}, true
	case KindServiceAccounts:
		return schema.GroupKind{Kind: "ServiceAccount"}, true
	case KindRoles:
		return schema.GroupKind{Group: "rbac.authorization.k8s.io", Kind: "Role"}, true
	case KindRoleBindings:
		return schema.GroupKind{Group: "rbac.authorization.k8s.io", Kind: "RoleBinding"}, true
	case KindPV:
		return schema.GroupKind{Kind: "PersistentVolume"}, true
	case KindPVC:
		return schema.GroupKind{Kind: "PersistentVolumeClaim"}, true
	case KindStorageClasses:
		return schema.GroupKind{Group: "storage.k8s.io", Kind: "StorageClass"}, true
	case KindResourceQuotas:
		return schema.GroupKind{Kind: "ResourceQuota"}, true
	case KindLimitRanges:
		return schema.GroupKind{Kind: "LimitRange"}, true
	case KindHPA:
		return schema.GroupKind{Group: "autoscaling", Kind: "HorizontalPodAutoscaler"}, true
	default:
		return schema.GroupKind{}, false
	}
}

// Describe renders kubectl-equivalent `describe` output for a resource,
// using kubectl's own describer implementations so the content/formatting
// matches `kubectl describe` exactly (including its "N bytes" redaction of
// Secret values, and appended recent Events). Falls back to YAML — with a
// note — for the handful of kinds kubectl itself has no typed describer
// for (Events, VPA, CRD).
func Describe(clients Clients, kind Kind, namespace, name string) (string, error) {
	if kind == KindConfig {
		return describeConfig(clients, namespace, name)
	}

	gk, ok := describerGroupKind(kind)
	if !ok {
		return describeFallbackYAML(clients, kind, namespace, name)
	}

	d, ok := describe.DescriberFor(gk, clients.RestConfig)
	if !ok {
		return describeFallbackYAML(clients, kind, namespace, name)
	}
	return d.Describe(namespace, name, describe.DescriberSettings{ShowEvents: true})
}

// describeConfig tries ConfigMap first, then Secret — List/GetYAML/Delete
// all have this same ambiguity, since KindConfig lists both together.
func describeConfig(clients Clients, namespace, name string) (string, error) {
	if d, ok := describe.DescriberFor(schema.GroupKind{Kind: "ConfigMap"}, clients.RestConfig); ok {
		if out, err := d.Describe(namespace, name, describe.DescriberSettings{ShowEvents: true}); err == nil {
			return out, nil
		}
	}
	d, ok := describe.DescriberFor(schema.GroupKind{Kind: "Secret"}, clients.RestConfig)
	if !ok {
		return "", fmt.Errorf("no describer registered for ConfigMap/Secret")
	}
	return d.Describe(namespace, name, describe.DescriberSettings{ShowEvents: true})
}

func describeFallbackYAML(clients Clients, kind Kind, namespace, name string) (string, error) {
	yaml, err := GetYAML(clients, kind, namespace, name)
	if err != nil {
		return "", err
	}
	return "(kubectl has no typed describer for " + kind.Title() + "; showing YAML instead)\n\n" + yaml, nil
}
