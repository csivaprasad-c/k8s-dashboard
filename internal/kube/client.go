package kube

import (
	"context"
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

// Session is a live connection to one cluster context: the resolved REST
// config plus ready-to-use clientsets.
type Session struct {
	Context   ClusterContext
	Config    *rest.Config
	Clientset *kubernetes.Clientset
	// Metrics talks to the metrics.k8s.io API (metrics-server). Building
	// this client never fails on its own — only calls against it do, if
	// metrics-server isn't installed in the cluster — so callers should
	// treat those call failures as "usage data unavailable", not a fatal
	// connection error.
	Metrics *metricsclientset.Clientset
}

// Connect builds a clientset for the named kubeconfig context. It does not
// itself verify reachability; call Ping for that.
func Connect(contextName string) (*Session, error) {
	rules := loadingRules()
	overrides := &clientcmd.ConfigOverrides{CurrentContext: contextName}
	restConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("building client config for context %q: %w", contextName, err)
	}
	// Keep the dashboard responsive: don't let a single slow/hung request
	// block the UI goroutine forever.
	restConfig.Timeout = 10 * time.Second

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("building clientset for context %q: %w", contextName, err)
	}

	metricsClient, err := metricsclientset.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("building metrics client for context %q: %w", contextName, err)
	}

	contexts, err := LoadContexts()
	if err != nil {
		return nil, err
	}
	var cc ClusterContext
	for _, c := range contexts {
		if c.Name == contextName {
			cc = c
			break
		}
	}

	return &Session{Context: cc, Config: restConfig, Clientset: clientset, Metrics: metricsClient}, nil
}

// Ping verifies the cluster is actually reachable and credentials are valid
// by asking for the server version, bounded by the given timeout.
func (s *Session) Ping(timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	verCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		v, err := s.Clientset.Discovery().ServerVersion()
		if err != nil {
			errCh <- err
			return
		}
		verCh <- v.GitVersion
	}()

	select {
	case v := <-verCh:
		return v, nil
	case err := <-errCh:
		return "", fmt.Errorf("connecting to %s: %w", s.Context.Server, err)
	case <-ctx.Done():
		return "", fmt.Errorf("timed out connecting to %s", s.Context.Server)
	}
}
