# k8sdash

A terminal dashboard for Kubernetes. Pick a cluster from your kubeconfig,
connect, and browse it live — nodes, pods, deployments, services, ingresses,
persistent volumes/claims, configmaps/secrets, and events — without leaving
the terminal.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[client-go](https://github.com/kubernetes/client-go).

## Build & run

```sh
go build -o bin/k8sdash ./cmd/k8sdash
./bin/k8sdash
```

It reads `$KUBECONFIG` (or `~/.kube/config` if unset), exactly like `kubectl`.

## Screens

1. **Cluster select** — every context in your merged kubeconfig, with
   `/` to filter and `enter` to connect.
2. **Connecting** — verifies the cluster is actually reachable
   (`ServerVersion`) before handing off, so a stale or unreachable context
   fails fast with a readable error instead of hanging.
3. **Dashboard** — tabbed resource browser for the connected cluster.

## Keybindings (dashboard)

| Key | Action |
|---|---|
| `1`-`9`, `0` / `Tab` / `Shift+Tab` | Switch resource tab (Overview, Nodes, Pods, Deploy, Svc, Ingress, PV, PVC, Cfg/Secrets, Events — `0` selects the 10th) |
| `↑`/`↓` | Move selection |
| `/` | Filter the current table by name |
| `n` | Switch namespace |
| `enter` | View the selected object as YAML |
| `l` | Tail logs (pods only; prompts for a container if there's more than one) |
| `x` | Delete the selected object (asks for `y`/`n` confirmation first) |
| `r` | Force refresh (the dashboard also auto-refreshes every 5s) |
| `ctrl+k` | Back to the cluster picker, without quitting |
| `?` | Toggle this help |
| `q` / `ctrl+c` | Quit (or close the current overlay) |

## Layout

```
cmd/k8sdash/          entrypoint
internal/
  kube/               kubeconfig loading + cluster connections
  k8sres/              client-go wrappers: list/describe/delete/logs per resource kind
  ui/
    styles/            shared lipgloss theme
    screens/
      clusterselect/   screen 1
      connecting/      screen 2
      dashboard/       screen 3 (tabs, table, namespace/detail/logs/delete/help overlays)
```

## Notes

- Resource lists refresh on a 5s poll (plus manual `r`) rather than a
  live watch/informer — simple and robust; a `SharedInformerFactory` would
  be the natural next step for push-based updates.
- The YAML detail view redacts Secret values (keys/lengths are still
  visible) and strips `managedFields` for readability.
- CPU/memory usage (Overview tab and the Nodes tab's CPU/MEMORY columns)
  comes from `metrics.k8s.io` (metrics-server). If that add-on isn't
  installed in the cluster, usage falls back to a muted "not available"
  note rather than erroring — capacity/health still show either way.
- Nodes and PersistentVolumes are cluster-scoped, so they ignore the
  active namespace; every other tab is scoped to it.
