package k8sres

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const listTimeout = 10 * time.Second

// List fetches rows for a kind in the given namespace (ignored for
// cluster-scoped kinds). It is a synchronous, single-shot call intended to
// be run from a tea.Cmd goroutine, not the UI loop.
func List(clientset *kubernetes.Clientset, kind Kind, namespace string) ([]Row, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
	defer cancel()

	switch kind {
	case KindNodes:
		return listNodes(ctx, clientset)
	case KindPods:
		return listPods(ctx, clientset, namespace)
	case KindDeployments:
		return listDeployments(ctx, clientset, namespace)
	case KindServices:
		return listServices(ctx, clientset, namespace)
	case KindConfig:
		return listConfig(ctx, clientset, namespace)
	case KindEvents:
		return listEvents(ctx, clientset, namespace)
	default:
		return nil, fmt.Errorf("no lister for kind %v", kind)
	}
}

func age(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func listNodes(ctx context.Context, cs *kubernetes.Clientset) ([]Row, error) {
	list, err := cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	rows := make([]Row, 0, len(list.Items))
	for _, n := range list.Items {
		status := "NotReady"
		class := "bad"
		for _, c := range n.Status.Conditions {
			if c.Type == corev1.NodeReady {
				if c.Status == corev1.ConditionTrue {
					status = "Ready"
					class = "ok"
				}
				break
			}
		}
		roles := nodeRoles(n.Labels)
		rows = append(rows, Row{
			Name: n.Name,
			Cells: []string{
				n.Name, status, roles, n.Status.NodeInfo.KubeletVersion, age(n.CreationTimestamp.Time),
			},
			StatusClass: class,
		})
	}
	sortRows(rows)
	return rows, nil
}

func nodeRoles(labels map[string]string) string {
	var roles []string
	for k := range labels {
		if strings.HasPrefix(k, "node-role.kubernetes.io/") {
			roles = append(roles, strings.TrimPrefix(k, "node-role.kubernetes.io/"))
		}
	}
	if len(roles) == 0 {
		return "<none>"
	}
	sort.Strings(roles)
	return strings.Join(roles, ",")
}

func listPods(ctx context.Context, cs *kubernetes.Clientset, ns string) ([]Row, error) {
	list, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	rows := make([]Row, 0, len(list.Items))
	for _, p := range list.Items {
		ready, total := 0, len(p.Spec.Containers)
		restarts := int32(0)
		for _, cs := range p.Status.ContainerStatuses {
			if cs.Ready {
				ready++
			}
			restarts += cs.RestartCount
		}
		status, class := podStatus(&p)
		rows = append(rows, Row{
			Name:      p.Name,
			Namespace: p.Namespace,
			Cells: []string{
				p.Name,
				fmt.Sprintf("%d/%d", ready, total),
				status,
				fmt.Sprintf("%d", restarts),
				age(p.CreationTimestamp.Time),
			},
			StatusClass: class,
		})
	}
	sortRows(rows)
	return rows, nil
}

// podStatus mirrors kubectl's phase/reason precedence closely enough for a
// dashboard: prefer a container waiting reason (e.g. CrashLoopBackOff) over
// the coarse pod phase.
func podStatus(p *corev1.Pod) (string, string) {
	for _, cs := range p.Status.ContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			return cs.State.Waiting.Reason, "bad"
		}
		if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" && cs.State.Terminated.ExitCode != 0 {
			return cs.State.Terminated.Reason, "bad"
		}
	}
	phase := string(p.Status.Phase)
	switch p.Status.Phase {
	case corev1.PodRunning:
		return phase, "ok"
	case corev1.PodSucceeded:
		return phase, "ok"
	case corev1.PodPending:
		return phase, "warn"
	default:
		return phase, "bad"
	}
}

func listDeployments(ctx context.Context, cs *kubernetes.Clientset, ns string) ([]Row, error) {
	list, err := cs.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	rows := make([]Row, 0, len(list.Items))
	for _, d := range list.Items {
		class := "ok"
		if d.Status.ReadyReplicas < d.Status.Replicas {
			class = "warn"
		}
		rows = append(rows, Row{
			Name:      d.Name,
			Namespace: d.Namespace,
			Cells: []string{
				d.Name,
				fmt.Sprintf("%d/%d", d.Status.ReadyReplicas, d.Status.Replicas),
				fmt.Sprintf("%d", d.Status.UpdatedReplicas),
				fmt.Sprintf("%d", d.Status.AvailableReplicas),
				age(d.CreationTimestamp.Time),
			},
			StatusClass: class,
		})
	}
	sortRows(rows)
	return rows, nil
}

func listServices(ctx context.Context, cs *kubernetes.Clientset, ns string) ([]Row, error) {
	list, err := cs.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	rows := make([]Row, 0, len(list.Items))
	for _, s := range list.Items {
		var ports []string
		for _, p := range s.Spec.Ports {
			ports = append(ports, fmt.Sprintf("%d/%s", p.Port, p.Protocol))
		}
		rows = append(rows, Row{
			Name:      s.Name,
			Namespace: s.Namespace,
			Cells: []string{
				s.Name, string(s.Spec.Type), s.Spec.ClusterIP, strings.Join(ports, ","), age(s.CreationTimestamp.Time),
			},
		})
	}
	sortRows(rows)
	return rows, nil
}

func listConfig(ctx context.Context, cs *kubernetes.Clientset, ns string) ([]Row, error) {
	var rows []Row
	cms, err := cs.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, c := range cms.Items {
		rows = append(rows, Row{
			Name: c.Name, Namespace: c.Namespace,
			Cells: []string{c.Name, "ConfigMap", fmt.Sprintf("%d", len(c.Data)), age(c.CreationTimestamp.Time)},
		})
	}
	secrets, err := cs.CoreV1().Secrets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, s := range secrets.Items {
		rows = append(rows, Row{
			Name: s.Name, Namespace: s.Namespace,
			Cells: []string{s.Name, "Secret", fmt.Sprintf("%d", len(s.Data)), age(s.CreationTimestamp.Time)},
		})
	}
	sortRows(rows)
	return rows, nil
}

func listEvents(ctx context.Context, cs *kubernetes.Clientset, ns string) ([]Row, error) {
	list, err := cs.CoreV1().Events(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	rows := make([]Row, 0, len(list.Items))
	for _, e := range list.Items {
		last := e.LastTimestamp.Time
		if last.IsZero() {
			last = e.EventTime.Time
		}
		class := "ok"
		if e.Type != corev1.EventTypeNormal {
			class = "warn"
		}
		rows = append(rows, Row{
			Name:      e.Name,
			Namespace: e.Namespace,
			Cells: []string{
				age(last), e.Type, e.Reason, fmt.Sprintf("%s/%s", e.InvolvedObject.Kind, e.InvolvedObject.Name), e.Message,
			},
			StatusClass: class,
		})
	}
	// newest first
	sort.Slice(rows, func(i, j int) bool { return rows[i].Cells[0] < rows[j].Cells[0] })
	return rows, nil
}

func sortRows(rows []Row) {
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
}

// ListNamespaces returns every namespace name in the cluster, used by the
// namespace-picker overlay.
func ListNamespaces(clientset *kubernetes.Clientset) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
	defer cancel()
	list, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(list.Items))
	for _, n := range list.Items {
		names = append(names, n.Name)
	}
	sort.Strings(names)
	return names, nil
}

// Overview summarizes cluster health for the Overview tab.
type Overview struct {
	NodeCount       int
	NodesReady      int
	NamespaceCount  int
	PodCount        int
	PodsRunning     int
	PodsPending     int
	PodsFailed      int
	DeploymentCount int
	DeploymentsBad  int
}

// GetOverview gathers the summary counters. It makes a handful of list
// calls scoped to the "" namespace (all namespaces) for pods/deployments so
// the Overview tab always reflects the whole cluster, not just the active
// namespace.
func GetOverview(clientset *kubernetes.Clientset) (Overview, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
	defer cancel()
	var o Overview

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return o, err
	}
	o.NodeCount = len(nodes.Items)
	for _, n := range nodes.Items {
		for _, c := range n.Status.Conditions {
			if c.Type == corev1.NodeReady && c.Status == corev1.ConditionTrue {
				o.NodesReady++
			}
		}
	}

	nss, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return o, err
	}
	o.NamespaceCount = len(nss.Items)

	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return o, err
	}
	o.PodCount = len(pods.Items)
	for _, p := range pods.Items {
		switch p.Status.Phase {
		case corev1.PodRunning, corev1.PodSucceeded:
			o.PodsRunning++
		case corev1.PodPending:
			o.PodsPending++
		default:
			o.PodsFailed++
		}
	}

	deps, err := clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return o, err
	}
	o.DeploymentCount = len(deps.Items)
	for _, d := range deps.Items {
		if d.Status.ReadyReplicas < d.Status.Replicas {
			o.DeploymentsBad++
		}
	}

	return o, nil
}
