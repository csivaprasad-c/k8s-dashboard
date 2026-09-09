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
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

const listTimeout = 10 * time.Second

// List fetches rows for a kind in the given namespace (ignored for
// cluster-scoped kinds). It is a synchronous, single-shot call intended to
// be run from a tea.Cmd goroutine, not the UI loop. metricsClient may be
// nil; it's only consulted for KindNodes, and only to add usage columns —
// its absence or failure never fails the list itself.
func List(clientset *kubernetes.Clientset, metricsClient *metricsclientset.Clientset, kind Kind, namespace string) ([]Row, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
	defer cancel()

	switch kind {
	case KindNodes:
		return listNodes(ctx, clientset, metricsClient)
	case KindPods:
		return listPods(ctx, clientset, namespace)
	case KindDeployments:
		return listDeployments(ctx, clientset, namespace)
	case KindServices:
		return listServices(ctx, clientset, namespace)
	case KindIngress:
		return listIngresses(ctx, clientset, namespace)
	case KindPV:
		return listPVs(ctx, clientset)
	case KindPVC:
		return listPVCs(ctx, clientset, namespace)
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

// nodeUsage is one node's usage against its own allocatable capacity, used
// to render the Nodes tab's CPU/MEMORY columns.
type nodeUsage struct {
	cpuMilli, cpuCapMilli int64
	memBytes, memCapBytes int64
}

func listNodes(ctx context.Context, cs *kubernetes.Clientset, metricsClient *metricsclientset.Clientset) ([]Row, error) {
	list, err := cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	usage := map[string]nodeUsage{}
	if metricsClient != nil {
		if metricsList, mErr := metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{}); mErr == nil {
			for _, nm := range metricsList.Items {
				usage[nm.Name] = nodeUsage{
					cpuMilli: nm.Usage.Cpu().MilliValue(),
					memBytes: nm.Usage.Memory().Value(),
				}
			}
		}
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

		u, hasUsage := usage[n.Name]
		u.cpuCapMilli = n.Status.Allocatable.Cpu().MilliValue()
		u.memCapBytes = n.Status.Allocatable.Memory().Value()
		cpuCell := "-"
		memCell := "-"
		if hasUsage {
			cpuCell = usageCell(u.cpuMilli, u.cpuCapMilli, formatMilliCores)
			memCell = usageCell(u.memBytes, u.memCapBytes, formatBytesCompact)
		}

		rows = append(rows, Row{
			Name: n.Name,
			Cells: []string{
				n.Name, status, roles, cpuCell, memCell, n.Status.NodeInfo.KubeletVersion, age(n.CreationTimestamp.Time),
			},
			StatusClass: class,
		})
	}
	sortRows(rows)
	return rows, nil
}

// usageCell renders "<used> (<pct>%)", e.g. "210m (1%)" or "1.3Gi (8%)".
func usageCell(used, capacity int64, format func(int64) string) string {
	if capacity <= 0 {
		return format(used)
	}
	pct := float64(used) / float64(capacity) * 100
	return fmt.Sprintf("%s (%.0f%%)", format(used), pct)
}

func formatMilliCores(m int64) string {
	return fmt.Sprintf("%dm", m)
}

// formatBytesCompact renders a byte count in compact binary units (e.g.
// "1.3Gi"), for narrow table cells; see also formatBytes in the dashboard
// package, which spells the unit out for the roomier Overview panel.
func formatBytesCompact(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ci", float64(b)/float64(div), "KMGTPE"[exp])
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
				s.Name, string(s.Spec.Type), s.Spec.ClusterIP, serviceExternalIP(s), strings.Join(ports, ","), age(s.CreationTimestamp.Time),
			},
		})
	}
	sortRows(rows)
	return rows, nil
}

// serviceExternalIP renders the EXTERNAL-IP column the way kubectl does:
// the ExternalName for that service type, the load balancer's assigned
// IP(s)/hostname(s) (or "<pending>" before one is assigned), or "<none>".
func serviceExternalIP(s corev1.Service) string {
	switch s.Spec.Type {
	case corev1.ServiceTypeExternalName:
		return s.Spec.ExternalName
	case corev1.ServiceTypeLoadBalancer:
		var addrs []string
		for _, ing := range s.Status.LoadBalancer.Ingress {
			if ing.IP != "" {
				addrs = append(addrs, ing.IP)
			} else if ing.Hostname != "" {
				addrs = append(addrs, ing.Hostname)
			}
		}
		if len(addrs) == 0 {
			return "<pending>"
		}
		return strings.Join(addrs, ",")
	default:
		return "<none>"
	}
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

func listIngresses(ctx context.Context, cs *kubernetes.Clientset, ns string) ([]Row, error) {
	list, err := cs.NetworkingV1().Ingresses(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	rows := make([]Row, 0, len(list.Items))
	for _, ing := range list.Items {
		class := "<none>"
		if ing.Spec.IngressClassName != nil {
			class = *ing.Spec.IngressClassName
		}

		hostSet := map[string]bool{}
		for _, r := range ing.Spec.Rules {
			if r.Host != "" {
				hostSet[r.Host] = true
			}
		}
		hosts := "*"
		if len(hostSet) > 0 {
			var hostList []string
			for h := range hostSet {
				hostList = append(hostList, h)
			}
			sort.Strings(hostList)
			hosts = strings.Join(hostList, ",")
		}

		var addrs []string
		for _, a := range ing.Status.LoadBalancer.Ingress {
			if a.IP != "" {
				addrs = append(addrs, a.IP)
			} else if a.Hostname != "" {
				addrs = append(addrs, a.Hostname)
			}
		}
		address := "-"
		if len(addrs) > 0 {
			address = strings.Join(addrs, ",")
		}

		ports := "80"
		if len(ing.Spec.TLS) > 0 {
			ports = "80, 443"
		}

		rows = append(rows, Row{
			Name:      ing.Name,
			Namespace: ing.Namespace,
			Cells:     []string{ing.Name, class, hosts, address, ports, age(ing.CreationTimestamp.Time)},
		})
	}
	sortRows(rows)
	return rows, nil
}

func listPVs(ctx context.Context, cs *kubernetes.Clientset) ([]Row, error) {
	list, err := cs.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	rows := make([]Row, 0, len(list.Items))
	for _, pv := range list.Items {
		class := pvPhaseClass(pv.Status.Phase)
		capacity := "-"
		if q, ok := pv.Spec.Capacity[corev1.ResourceStorage]; ok {
			capacity = formatBytesCompact(q.Value())
		}
		claim := "-"
		if pv.Spec.ClaimRef != nil {
			claim = pv.Spec.ClaimRef.Namespace + "/" + pv.Spec.ClaimRef.Name
		}
		storageClass := pv.Spec.StorageClassName
		if storageClass == "" {
			storageClass = "-"
		}
		rows = append(rows, Row{
			Name: pv.Name,
			Cells: []string{
				pv.Name, capacity, formatAccessModes(pv.Spec.AccessModes),
				string(pv.Status.Phase), claim, storageClass, age(pv.CreationTimestamp.Time),
			},
			StatusClass: class,
		})
	}
	sortRows(rows)
	return rows, nil
}

func listPVCs(ctx context.Context, cs *kubernetes.Clientset, ns string) ([]Row, error) {
	list, err := cs.CoreV1().PersistentVolumeClaims(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	rows := make([]Row, 0, len(list.Items))
	for _, pvc := range list.Items {
		class := pvcPhaseClass(pvc.Status.Phase)
		capacity := "-"
		if q, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
			capacity = formatBytesCompact(q.Value())
		}
		volume := pvc.Spec.VolumeName
		if volume == "" {
			volume = "-"
		}
		storageClass := "-"
		if pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName != "" {
			storageClass = *pvc.Spec.StorageClassName
		}
		accessModes := pvc.Status.AccessModes
		if len(accessModes) == 0 {
			accessModes = pvc.Spec.AccessModes
		}
		rows = append(rows, Row{
			Name:      pvc.Name,
			Namespace: pvc.Namespace,
			Cells: []string{
				pvc.Name, string(pvc.Status.Phase), volume, capacity, formatAccessModes(accessModes), storageClass, age(pvc.CreationTimestamp.Time),
			},
			StatusClass: class,
		})
	}
	sortRows(rows)
	return rows, nil
}

func pvPhaseClass(phase corev1.PersistentVolumePhase) string {
	switch phase {
	case corev1.VolumeBound, corev1.VolumeAvailable:
		return "ok"
	case corev1.VolumePending:
		return "warn"
	case corev1.VolumeReleased, corev1.VolumeFailed:
		return "bad"
	default:
		return ""
	}
}

func pvcPhaseClass(phase corev1.PersistentVolumeClaimPhase) string {
	switch phase {
	case corev1.ClaimBound:
		return "ok"
	case corev1.ClaimPending:
		return "warn"
	case corev1.ClaimLost:
		return "bad"
	default:
		return ""
	}
}

// formatAccessModes renders access modes using kubectl's short codes
// (RWO/ROX/RWX/RWOP).
func formatAccessModes(modes []corev1.PersistentVolumeAccessMode) string {
	if len(modes) == 0 {
		return "-"
	}
	codes := make([]string, 0, len(modes))
	for _, m := range modes {
		switch m {
		case corev1.ReadWriteOnce:
			codes = append(codes, "RWO")
		case corev1.ReadOnlyMany:
			codes = append(codes, "ROX")
		case corev1.ReadWriteMany:
			codes = append(codes, "RWX")
		case corev1.ReadWriteOncePod:
			codes = append(codes, "RWOP")
		default:
			codes = append(codes, string(m))
		}
	}
	return strings.Join(codes, ",")
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

// HealthStatus is a single-glance verdict on overall cluster health,
// derived from the same counters shown on the Overview tab.
type HealthStatus int

const (
	HealthUnknown HealthStatus = iota
	HealthHealthy
	HealthDegraded
	HealthUnhealthy
)

// String is the label shown next to the health indicator.
func (h HealthStatus) String() string {
	switch h {
	case HealthHealthy:
		return "Healthy"
	case HealthDegraded:
		return "Degraded"
	case HealthUnhealthy:
		return "Unhealthy"
	default:
		return "Unknown"
	}
}

// Class maps the status to a "ok"/"warn"/"bad" style class (see
// styles.StatusStyle).
func (h HealthStatus) Class() string {
	switch h {
	case HealthHealthy:
		return "ok"
	case HealthDegraded:
		return "warn"
	case HealthUnhealthy:
		return "bad"
	default:
		return ""
	}
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

	// CPU/memory are aggregated across all nodes. Capacity comes from
	// each node's allocatable resources; usage comes from metrics-server
	// (metrics.k8s.io) and is only populated when MetricsAvailable is
	// true — that API is an optional cluster add-on, so its absence is
	// expected and not an error.
	CPUUsageMilli    int64
	CPUCapacityMilli int64
	MemUsageBytes    int64
	MemCapacityBytes int64
	MetricsAvailable bool

	// MetricsErr holds why usage is unavailable (e.g. "metrics-server not
	// installed"), for display purposes only.
	MetricsErr string
}

// Health derives an overall verdict from the counters above: any node down
// is Unhealthy; failed pods or an incompletely rolled out deployment is
// Degraded; otherwise Healthy.
func (o Overview) Health() HealthStatus {
	if o.NodeCount == 0 {
		return HealthUnknown
	}
	if o.NodesReady < o.NodeCount {
		return HealthUnhealthy
	}
	if o.PodsFailed > 0 || o.DeploymentsBad > 0 || o.PodsPending > 0 {
		return HealthDegraded
	}
	return HealthHealthy
}

// GetOverview gathers the summary counters. It makes a handful of list
// calls scoped to the "" namespace (all namespaces) for pods/deployments so
// the Overview tab always reflects the whole cluster, not just the active
// namespace. metricsClient may be nil, in which case CPU/memory usage is
// simply left unavailable.
func GetOverview(clientset *kubernetes.Clientset, metricsClient *metricsclientset.Clientset) (Overview, error) {
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
		if cpu, ok := n.Status.Allocatable[corev1.ResourceCPU]; ok {
			o.CPUCapacityMilli += cpu.MilliValue()
		}
		if mem, ok := n.Status.Allocatable[corev1.ResourceMemory]; ok {
			o.MemCapacityBytes += mem.Value()
		}
	}

	if metricsClient == nil {
		o.MetricsErr = "metrics client not configured"
	} else {
		nodeMetrics, mErr := metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
		if mErr != nil {
			o.MetricsErr = "metrics-server not available"
		} else {
			o.MetricsAvailable = true
			for _, nm := range nodeMetrics.Items {
				if cpu, ok := nm.Usage[corev1.ResourceCPU]; ok {
					o.CPUUsageMilli += cpu.MilliValue()
				}
				if mem, ok := nm.Usage[corev1.ResourceMemory]; ok {
					o.MemUsageBytes += mem.Value()
				}
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
