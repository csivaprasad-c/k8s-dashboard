# k8sdash

A terminal dashboard for Kubernetes. Pick a cluster from your kubeconfig,
connect, and browse it live — nodes, workloads (pods, deployments,
daemonsets, statefulsets, jobs, cronjobs), networking (services, ingresses,
network policies), RBAC (service accounts, roles, role bindings), storage
(persistent volumes/claims, storage classes), governance (resource quotas,
limit ranges), autoscaling (HPAs, VPAs), CRDs, configmaps/secrets, and
events — without leaving the terminal.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[client-go](https://github.com/kubernetes/client-go).

## Build & run

```sh
go build -o bin/k8sdash ./cmd/k8sdash
./bin/k8sdash
```

It reads `$KUBECONFIG` (or `~/.kube/config` if unset), exactly like `kubectl`.

### Releasing

Push a tag matching `v*.*.*` and [.github/workflows/release.yml](.github/workflows/release.yml)
cross-compiles windows/amd64, darwin/arm64, and linux/arm64 (`make dist`),
then publishes them as a GitHub Release with a `checksums.txt`:

```sh
git tag v0.1.0
git push origin v0.1.0
```

To re-run a release for a tag that's already been pushed (e.g. after a
failed run), use the "Run workflow" button on the Release workflow in the
Actions tab, or `gh workflow run release.yml -f tag=v0.1.0` — it replaces
that release's assets instead of failing on "already exists".

## Screens

1. **Cluster select** — every context in your merged kubeconfig, with
   `/` to filter and `enter` to connect.
2. **Connecting** — verifies the cluster is actually reachable
   (`ServerVersion`) before handing off, so a stale or unreachable context
   fails fast with a readable error instead of hanging.
3. **Dashboard** — tabbed resource browser for the connected cluster. A
   one-line cluster-health strip (pods/deployments/replicasets ready, plus
   any prominent issues) sits above the tab bar on every tab, not just
   Overview, and keeps refreshing in the background regardless of which
   tab you're looking at.

## Keybindings (dashboard)

| Key | Action |
|---|---|
| `:` | Jump to a resource tab — type to search by name, `enter` to select (there are too many kinds for one key each) |
| `Tab` / `Shift+Tab` | Cycle to the next/previous resource tab |
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
- Nodes, PersistentVolumes, StorageClasses, and CustomResourceDefinitions
  are cluster-scoped, so they ignore the active namespace; every other tab
  is scoped to it.
- VerticalPodAutoscaler is a CRD from the separate
  [autoscaler](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler)
  project, not core Kubernetes, and often isn't installed. Its tab (and the
  CRD tab itself) is read generically via a dynamic client rather than
  pulling in a typed client as a dependency; a cluster without the CRD just
  shows the same "resource not found" error any missing API would.
- Deleting a CRD is far more destructive than deleting most other things
  here — it cascades to every custom resource of that type, cluster-wide —
  so its delete confirmation carries an extra warning line.
