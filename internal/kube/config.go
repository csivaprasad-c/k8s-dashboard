// Package kube handles loading the developer's kubeconfig(s) and turning
// them into usable Kubernetes clients.
package kube

import (
	"fmt"
	"sort"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// ClusterContext describes one selectable entry from the merged kubeconfig.
// A "context" (not a bare cluster) is what's selectable, since that's what
// pairs a cluster with credentials and a default namespace.
type ClusterContext struct {
	Name      string // context name, e.g. "orbstack"
	Cluster   string // cluster name referenced by the context
	AuthInfo  string // user/credentials name referenced by the context
	Namespace string // default namespace for this context (falls back to "default")
	Server    string // API server URL, resolved from the cluster entry
	Current   bool   // true if this is kubeconfig's current-context
}

// loadingRules returns the standard kubeconfig discovery rules: it honors
// $KUBECONFIG (colon-separated list of files, merged) and falls back to
// ~/.kube/config.
func loadingRules() *clientcmd.ClientConfigLoadingRules {
	return clientcmd.NewDefaultClientConfigLoadingRules()
}

// LoadContexts reads and merges all configured kubeconfig files and returns
// every context defined in them, sorted by name with the current context
// first.
func LoadContexts() ([]ClusterContext, error) {
	rules := loadingRules()
	raw, err := rules.Load()
	if err != nil {
		return nil, fmt.Errorf("loading kubeconfig: %w", err)
	}
	if len(raw.Contexts) == 0 {
		return nil, fmt.Errorf("no contexts found in kubeconfig (checked %s)", rules.GetLoadingPrecedence())
	}

	out := make([]ClusterContext, 0, len(raw.Contexts))
	for name, ctx := range raw.Contexts {
		out = append(out, toClusterContext(raw, name, ctx))
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Current != out[j].Current {
			return out[i].Current // current context sorts first
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func toClusterContext(raw *clientcmdapi.Config, name string, ctx *clientcmdapi.Context) ClusterContext {
	ns := ctx.Namespace
	if ns == "" {
		ns = "default"
	}
	server := ""
	if cluster, ok := raw.Clusters[ctx.Cluster]; ok {
		server = cluster.Server
	}
	return ClusterContext{
		Name:      name,
		Cluster:   ctx.Cluster,
		AuthInfo:  ctx.AuthInfo,
		Namespace: ns,
		Server:    server,
		Current:   name == raw.CurrentContext,
	}
}
