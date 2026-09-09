package k8sres

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

// GetYAML fetches a single object and renders it as YAML for the detail
// overlay. It clears ManagedFields, which are voluminous and rarely useful
// in a quick-look pane.
func GetYAML(clientset *kubernetes.Clientset, kind Kind, namespace, name string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
	defer cancel()

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
	case KindServices:
		full, e := clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
		if e != nil {
			return "", e
		}
		full.ManagedFields = nil
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
func Delete(clientset *kubernetes.Clientset, kind Kind, namespace, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
	defer cancel()

	switch kind {
	case KindPods:
		return clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case KindDeployments:
		return clientset.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case KindServices:
		return clientset.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case KindConfig:
		if err := clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err == nil {
			return nil
		}
		return clientset.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	default:
		return fmt.Errorf("delete not supported for kind %v", kind)
	}
}
