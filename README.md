# Preview Environment PR Commenter (Go)

A small Kubernetes service that watches **Argo CD Applications** representing preview environments and posts a **GitHub Pull Request comment** with:

- the latest commit SHA deployed,
- an Argo CD link,
- preview links rendered as **QR codes** (minted via a protected QR signing service),
- Grafana metrics and Loki logs links.

It’s designed for GitOps/preview workflows where each PR spins up a temporary environment and you want an automatic “here’s where it is” comment.

---

## How it works

On a fixed interval, the service:

1. Lists Argo CD `Applications` (`argoproj.io/v1alpha1`) from Kubernetes using the dynamic client.
2. Filters to apps that:
    - have at least one `ownerReference`,
    - have annotation `preview_environment: "true"`,
    - have annotation `head_sha` and the SHA appears in `status.sync.revisions`,
    - have `status.health.status` of `Healthy` or `Degraded`,
    - have at least one `status.summary.externalURLs` entry,
    - contain required GitHub metadata annotations: `repository_organization`, `repository_name`, `pull_request_number`.
3. Builds links to:
    - Argo CD application page,
    - Grafana “workload metrics” dashboard,
    - Loki logs dashboard.
4. For each external URL, calls a QR mint endpoint to produce a signed QR image URL.
5. Posts a markdown table comment to the matching PR via the GitHub API.
6. De-duplicates per process lifetime using `head_sha` (in-memory).

---

## Inputs expected on the Argo CD Application

### Required annotations

| Annotation | Required | Meaning |
|---|:---:|---|
| `preview_environment` | ✅ | Must be `"true"` to be considered |
| `head_sha` | ✅ | Commit SHA expected to be deployed |
| `repository_organization` | ✅ | GitHub org/owner |
| `repository_name` | ✅ | GitHub repo name |
| `pull_request_number` | ✅ | PR number to comment on |

### Required status fields (set by Argo CD)

- `status.sync.revisions` must contain `head_sha`
- `status.health.status` must be `Healthy` or `Degraded`
- `status.summary.externalURLs` must contain at least one URL

---

## Configuration

Configuration is via environment variables:

| Env var | Default | Required | Description |
|---|---:|:---:|---|
| `NAMESPACE` | `glueops-core` |  | Namespace used to read ConfigMap/Secret and list Applications |
| `GITHUB_APP_SECRET_NAME` | `tenant-repo-creds` |  | K8s Secret with GitHub App credentials |
| `CAPTAIN_DOMAIN_K8S_CONFIGMAP_NAME` | `glueops-captain-domain` |  | K8s ConfigMap with `captain_domain` |
| `WATCH_FOR_APPS_DELAY_SECONDS` | `10` |  | Poll interval in seconds |
| `QR_MINT_TOKEN` |  | ✅ | Bearer token for QR signing endpoint |
| `QR_TTL_SECONDS` | `600` |  | Signed QR TTL in seconds |
| `KUBECONFIG` | `~/.kube/config` |  | Used only when not running in-cluster |

Hard-coded defaults:
- HTTP timeout: 15 seconds
- GitHub API version header: `2022-11-28`

---

## Kubernetes resources required

### ConfigMap: captain domain

A ConfigMap containing the base domain used for service links:

- Name: `CAPTAIN_DOMAIN_K8S_CONFIGMAP_NAME` (default `glueops-captain-domain`)
- Key: `captain_domain`

Example:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: glueops-captain-domain
  namespace: glueops-core
data:
  captain_domain: example.internal
