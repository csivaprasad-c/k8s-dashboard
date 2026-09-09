package k8sres

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// GetYAML fetches a single object and renders it as YAML for the detail
// overlay. It clears ManagedFields, which are voluminous and rarely useful
// in a quick-look pane.
func GetYAML(clients Clients, kind Kind, namespace, name string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
	defer cancel()
	clientset := clients.Core

	var obj interface{}
	var err error

	switch kind {
	case KindNodes:
		full, e := clientset.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return "", e
		}
		full.ManagedFields = nil
		obj = full
	case KindPods:
		full, e := clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return "", e
		}
		full.ManagedFields = nil
		obj = full
	case KindDeployments:
		full, e := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return "", e
		}
		full.ManagedFields = nil
		obj = full
	case KindDaemonSets:
		full, e := clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return "", e
		}
		full.ManagedFields = nil
		obj = full
	case KindStatefulSets:
		full, e := clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return "", e
		}
		full.ManagedFields = nil
		obj = full
	case KindJobs:
		full, e := clientset.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return "", e
		}
		full.ManagedFields = nil
		obj = full
	case KindCronJobs:
		full, e := clientset.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return "", e
		}
		full.ManagedFields = nil
		obj = full
	case KindServices:
		full, e := clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return "", e
		}
		full.ManagedFields = nil
		obj = full
	case KindIngress:
		full, e := clientset.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return "", e
		}
		full.ManagedFields = nil
		obj = full
	case KindNetworkPolicies:
		full, e := clientset.NetworkingV1().NetworkPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return "", e
		}
		full.ManagedFields = nil
		obj = full
	case KindPV:
		full, e := clientset.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return "", e
		}
		full.ManagedFields = nil
		obj = full
	case KindPVC:
		full, e := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return "", e
		}
		full.ManagedFields = nil
		obj = full
	case KindStorageClasses:
		full, e := clientset.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return "", e
		}
		full.ManagedFields = nil
		obj = full
	case KindHPA:
		full, e := clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return "", e
		}
		full.ManagedFields = nil
		obj = full
	case KindVPA:
		full, e := getVPAYAML(ctx, clients.Dynamic, namespace, name)
		if e != nil {
			return "", e
		}
		obj = full
	case KindConfig:
		// Try ConfigMap first, then Secret.
		if cm, e := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{}); e == nil {
			cm.ManagedFields = nil
			obj = cm
		} else if s, e2 := clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{}); e2 == nil {
			s.ManagedFields = nil
			// Redact secret values; presence/length is still visible via Data keys.
			for k := range s.Data {
				s.Data[k] = []byte("<redacted>")
			}
			obj = s
		} else {
			return "", e2
		}
	case KindEvents:
		full, e := clientset.CoreV1().Events(namespace).Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return "", e
		}
		full.ManagedFields = nil
		obj = full
	default:
		return "", fmt.Errorf("no detail view for kind %v", kind)
	}

	b, err := yaml.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Delete removes a single object. Callers are expected to confirm with the
// user before invoking this — it does not ask.
func Delete(clients Clients, kind Kind, namespace, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
	defer cancel()
	clientset := clients.Core

	switch kind {
	case KindPods:
		return clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case KindDeployments:
		return clientset.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case KindDaemonSets:
		return clientset.AppsV1().DaemonSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case KindStatefulSets:
		return clientset.AppsV1().StatefulSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case KindJobs:
		return clientset.BatchV1().Jobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case KindCronJobs:
		return clientset.BatchV1().CronJobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case KindServices:
		return clientset.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case KindIngress:
		return clientset.NetworkingV1().Ingresses(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case KindNetworkPolicies:
		return clientset.NetworkingV1().NetworkPolicies(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case KindPV:
		return clientset.CoreV1().PersistentVolumes().Delete(ctx, name, metav1.DeleteOptions{})
	case KindPVC:
		return clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case KindStorageClasses:
		return clientset.StorageV1().StorageClasses().Delete(ctx, name, metav1.DeleteOptions{})
	case KindHPA:
		return clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case KindVPA:
		return deleteVPA(ctx, clients.Dynamic, namespace, name)
	case KindConfig:
		if err := clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err == nil {
			return nil
		}
		return clientset.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	default:
		return fmt.Errorf("delete not supported for kind %v", kind)
	}
}
