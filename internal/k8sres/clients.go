package k8sres

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

// Clients bundles every Kubernetes API client List/GetYAML/Delete/Describe
// might need for a given Kind. Metrics and Dynamic back optional/add-on
// APIs (metrics-server, and CRDs like VerticalPodAutoscaler) — callers pass
// nil if unavailable, and every use here treats that (or a call failing) as
// "feature unavailable", never a fatal error.
type Clients struct {
	Core    *kubernetes.Clientset
	Metrics *metricsclientset.Clientset
	Dynamic dynamic.Interface
	// RestConfig backs Describe, which needs to build kubectl's own
	// describer types (k8s.io/kubectl/pkg/describe) directly from a REST
	// config rather than an existing clientset.
	RestConfig *rest.Config
}
