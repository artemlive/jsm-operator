# JSM Operator

A Kubernetes controller for managing **Jira Service Management (JSM)** services and teams declaratively via Custom Resources.  
This operator uses Atlassian's REST and GraphQL APIs to provision and sync JSM resources with your Kubernetes infrastructure using GitOps principles.

## ✨ Features

- Declarative management of:
  - **JSM Services**
  - **JSM Teams** 
- Automatic resolution of service-to-team relationships
- Status propagation and update handling
- Respects `generation` and optimizes for idempotency

# 📝 TODO – JSM Operator Roadmap

A growing list of improvements and features for the `jsm-operator`.

---

## ✅ MVP Scope (Completed or Near-Done)

- [x] Create and manage `JSMService` via GraphQL
- [x] Create and manage `JSMTeam` via REST/GraphQL
- [x] Resolve team relationships to JSM services (ownership)
- [x] Reconcile based on `generation` and `ObservedGeneration`
- [x] Status reporting (ID, revision, tier info, etc.)
- [x] Exponential backoff and rate limiting
- [x] Secure metrics and health endpoints
- [x] GraphQL client wrapper with reusable request structs

---

## 🚧 Near-Term Enhancements
- [ ] Kstatus propagation  
  Ensure `status` fields are updated correctly on resource changes.
  They are already there but need to be properly set on reconciliation.
- [ ] 🔁 Reconciliation Backoff Tuning  
  Make backoff configurable via flags (initial + max delay).

- [ ] 🧪 Unit Tests
  - [ ] JSM client coverage (mocked API)
  - [ ] Reconciliation logic (fake client, envtest or controller-runtime env)

- [ ] 🧪 Integration Tests
  - [ ] Validate JSM CRDs against mocked/stubbed API
  - [ ] Verify status propagation on changes

- [ ] 🔐 Webhook for validation
  - [ ] Enforce `name` immutability
  - [ ] Validate `tierLevel` range (1–4)
  - [ ] Optional validation for `serviceTypeKey`

- [ ] 🔄 Service renaming strategy  
  Decide if renames should be forbidden or handled via recreation logic.

- [ ] 📖 Better Documentation
  - [ ] Quickstart example with Secrets + ConfigMap
  - [ ] Architecture diagram (CRD → controller → JSM API)

---

## 🚀 Future Resource Support

- [ ] `JSMSchedule`  
  Manage Opsgenie schedules (REST-only)

- [ ] `JSMEscalation`  
  Support creation and management of escalation policies

- [ ] etc

The full API list is here: [Jira Service Management API](https://developer.atlassian.com/cloud/jira/service-desk-ops/rest/v2/intro/#authentication)
---

## 🧠 Other Ideas

- [ ] Finalizers for cleanup logic (e.g., remove team links on delete)

---

## 🤖 Nice-to-Have

- [ ] Helm chart
- [ ] GitHub Actions CI workflow
- [ ] Prometheus alert rules for controller health
- [ ] OperatorHub / Kubeapps support
---

## ⚙️ Installation

```bash
kubectl apply -k config/default
```

---

## 🚀 Usage

Apply CRs like `JSMService` and `JSMTeam` to declaratively manage Jira Ops resources:

```yaml
apiVersion: jsm.macpaw.dev/v1beta1
kind: JSMTeam
metadata:
  name: core-team
spec:
  name: "Core Team"
```

```yaml
apiVersion: jsm.macpaw.dev/v1beta1
kind: JSMService
metadata:
  name: app-service
spec:
  description: "app for internal workflows"
  tierLevel: 3
  serviceTypeKey: "APPLICATIONS"
  teamRef:
    name: core-team
```

---

## 🔐 Environment Configuration

You can configure the operator via environment variables or flags:

| Flag                  | Env Var             | Description                                                                 |
|-----------------------|---------------------|-----------------------------------------------------------------------------|
| `--jsm-api-token`     | `JSM_API_TOKEN`     | JSM API token                                                              |
| `--jsm-username`      | `JSM_USERNAME`      | JSM username (for basic auth)                                              |
| `--jsm-cloud-id`      | `JSM_CLOUD_ID`      | Atlassian Cloud ID (can be found in `_edge/tenant_info`)                   |
| `--jsm-graphql-url`   | `JSM_GRAPHQL_URL`   | GraphQL endpoint (`https://api.atlassian.com/graphql`)    |
| `--jsm-rest-url`      | `JSM_OPS_REST_URL`  | JSM REST base URL (e.g. `https://api.atlassian.com/jsm/ops/api`)           |

These can be passed as command-line flags or populated via a Kubernetes secret/config map.

---

## 🔄 Reconciliation Behavior

- Services and teams are reconciled based on the latest `generation`
- Status reflects external state (`id`, `revision`, `team relationship`)
- Renaming is **not supported** — names are treated as immutable in JSM

---

## 📈 Metrics and Health Probes

- Metrics endpoint: `:8443` (HTTPS) or `:8080` (HTTP, if disabled TLS)
- Health probes: exposed at `:8081` by default
- HTTP/2 is disabled by default due to security concerns (can be toggled with `--enable-http2`)

---

## 🛠 Dev Notes

- Uses controller-runtime and Kubebuilder

---

## 📄 License

Apache 2.0 — see `LICENSE` file.
