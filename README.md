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

Releases are built with [GoReleaser](https://goreleaser.com) (config:
[.goreleaser.yaml](.goreleaser.yaml)). Push a tag matching `v*.*.*` and
[.github/workflows/release.yml](.github/workflows/release.yml)
cross-compiles windows/amd64, darwin/arm64, and linux/arm64, then publishes
a GitHub Release with the archives, a `checksums.txt`, and an
auto-generated changelog attached:

```sh
git tag v0.1.0
git push origin v0.1.0
```

To try a config change locally first, without a tag or touching GitHub:

```sh
brew install goreleaser   # if you don't have it
make release-snapshot     # builds dist/ exactly as CI would, skips publishing
```

If a release run fails partway, delete the tag and its (partial) GitHub
release, then re-push the tag — GoReleaser refuses to publish over an
existing release for the same tag.

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
| `enter` | Describe the selected object (`kubectl describe`-equivalent); `d`/`y` inside toggle Describe/YAML |
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
- The detail overlay's Describe mode uses kubectl's own describers
  (`k8s.io/kubectl/pkg/describe`), so output matches `kubectl describe`
  exactly — same sections, same "N bytes" Secret redaction. Kinds kubectl
  itself has no typed describer for (Events, VPA, CRD) fall back to YAML,
  with a note. The YAML mode redacts Secret values (keys/lengths still
  visible), strips `managedFields`, and also blanks the
  `kubectl.kubernetes.io/last-applied-configuration` annotation on
  Secrets — created by `kubectl apply`, it otherwise stores a verbatim
  JSON copy of the object as last applied, including original plaintext
  `stringData` values, bypassing Data-only redaction entirely.
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
