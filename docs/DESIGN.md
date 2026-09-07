# Design & Decisions

This document explains **what was built and why**, and directly answers the
take-home's prompt:

> *"What are all the components required to have a helm chart deploy an
> application? Try to add as much as you can, but focus on main functionalities."*

---

## 1. The application

### Language & framework

Go with the **standard library** (`net/http`, the Go 1.22+ method+wildcard
router). No web framework. Reasons:

- A statically-linked Go binary produces a **~15 MB distroless image** that
  starts in milliseconds — a good citizen in Kubernetes (fast rollouts, cheap
  autoscaling, small attack surface).
- The standard library is more than enough for two endpoints; fewer
  dependencies means less to audit and patch.

### Structure — separation of concerns

```
internal/birthday/   Pure business logic. No HTTP, no DB, no clock reads.
                     `Message(username, dob, now)` -> string. 100% unit-tested,
                     including leap-year (Feb 29) edge cases.
internal/store/      Persistence behind a small `Store` interface with two
                     implementations: Postgres (prod) and in-memory (tests/demo).
internal/api/        HTTP handlers, input validation, JSON, logging & recovery
                     middleware, health/readiness probes.
cmd/server/          Wiring: config from env, DB connect-with-retry, graceful
                     shutdown on SIGTERM.
```

The clock (`now`) and the store are injected, so every rule is testable without
spinning up infrastructure — see `internal/birthday/birthday_test.go` and
`internal/api/handlers_test.go`.

### Validation (from the spec)

- `username` must match `^[A-Za-z]+$` → otherwise `400`.
- `dateOfBirth` must parse as `YYYY-MM-DD` **and** be strictly before today (UTC)
  → otherwise `400`.
- Unknown JSON fields and oversized bodies are rejected.
- The spec's literal typo (`dateOfBrith`) is accepted as a fallback so either
  payload works.

### Storage choice: PostgreSQL

"Storage of your choice." PostgreSQL because it maps 1:1 onto **RDS/Aurora** in
the AWS design, giving a coherent story from laptop to production. The schema is
a single table:

```sql
CREATE TABLE users (username TEXT PRIMARY KEY, date_of_birth DATE NOT NULL);
```

Writes are an idempotent `INSERT ... ON CONFLICT DO UPDATE` (upsert), matching
the "save/update" requirement.

### Container

Multi-stage `Dockerfile`:

1. `golang:1.23-alpine` builds a static, stripped binary (`CGO_ENABLED=0`).
2. `gcr.io/distroless/static-debian12:nonroot` runs it as UID 65532 — no shell,
   no package manager, read-only root filesystem enforced by the chart.

---

## 2. What a Helm chart needs to deploy an application

Here is the checklist the prompt asks for, mapped to the files in
`helm/birthday-app/templates/`. Everything marked ✅ is implemented.

| # | Component | Purpose | In this chart |
|---|-----------|---------|---------------|
| 1 | **Chart.yaml** | Chart identity, version, appVersion | ✅ |
| 2 | **values.yaml** | Single source of tunables; per-env overrides | ✅ |
| 3 | **Deployment** | Runs N replicas of the app, rolling updates | ✅ (`maxUnavailable: 0` = zero-downtime) |
| 4 | **Service** | Stable in-cluster address / load balancing | ✅ ClusterIP |
| 5 | **Ingress** | L7 entrypoint from outside the cluster | ✅ Kong Ingress |
| 6 | **ConfigMap** | Non-secret config (port, log level, backend) | ✅ |
| 7 | **Secret** | DB credentials / `DATABASE_URL` | ✅ (or `existingSecret` in prod) |
| 8 | **ServiceAccount** | Pod identity (→ IRSA in AWS) | ✅ token automount disabled |
| 9 | **Liveness/Readiness probes** | Self-healing + safe rollouts | ✅ `/healthz`, `/readyz` |
| 10 | **Resource requests/limits** | Scheduling & noisy-neighbor protection | ✅ |
| 11 | **HorizontalPodAutoscaler** | Scale with load (high usage) | ✅ CPU-based |
| 12 | **SecurityContext** | Runs as non-root, no privilege escalation | ✅ |
| 13 | **Helm test hook** | Post-install smoke test | ✅ curl PUT+GET |
| 14 | **NOTES.txt** | Post-install usage instructions | ✅ |
| 15 | **_helpers.tpl** | DRY names/labels | ✅ |
| 16 | **Database** | The app's state | ✅ in-chart StatefulSet (local) / managed (prod) |

Deliberately **out of the chart** (and why):

- **Kong itself** — installed once per cluster as platform infrastructure, not
  per-app. The chart only creates the `Ingress` + `KongPlugin` that plug into it.
- **NetworkPolicy and PodDisruptionBudget** — real production concerns, but I
  left them out to keep the take-home focused on the main path. Both are a few
  lines to add and I can explain exactly what each would do. (A `PodDisruptionBudget`
  keeps a minimum number of replicas during node drains; a `NetworkPolicy` limits
  which pods can reach the app.) Scoping down what I can fully defend beats
  padding the chart with knobs I'd struggle to justify.
- **cert-manager / TLS, ExternalDNS, Prometheus metrics** — platform-layer
  concerns; in AWS they map to ACM, Route 53 and managed monitoring (see the AWS
  doc). I did *not* add a Prometheus `ServiceMonitor` because the app doesn't
  expose `/metrics` yet — that's more honest than shipping a scrape that 404s.

### Resiliency features baked in

- **Zero-downtime rolling updates** (`maxUnavailable: 0`, `maxSurge: 1`).
- **Self-healing** via liveness/readiness probes.
- **Autoscaling** (HPA) for the "high usage" requirement.
- **DB connect-with-retry** so `helm install` works even if the database pod
  becomes ready a few seconds after the app.

---

## 3. Local platform: kind + Kong

The spec suggested minikube or k3s; I used **kind**, which is an equivalent
lightweight local cluster, for three concrete reasons:

- **Multi-node in seconds.** kind runs each node as a Docker container, so a
  1 control-plane + 2 workers cluster comes up in one command — enough to
  actually demonstrate replica spreading and rolling updates. minikube is
  single-node by default; a multi-node k3s means wiring up several hosts/VMs.
- **Ephemeral and scriptable.** kind is what the Kubernetes project uses for its
  own CI: a whole cluster spins up and tears down from a single script, which
  keeps the local bring-up reproducible.
- **No hypervisor.** It only needs Docker (or colima), which the dev already has.
  minikube typically spins up a VM driver.

Honest trade-off: kind is built for *testing/CI*, not production. If the goal
were a production-like local cluster I'd reach for **k3s** (a real, CNCF-grade
lightweight distro used at the edge); if it were the friendliest onboarding UX,
**minikube** (one-command addons). For this take-home — reproducible, multi-node
and script-driven — kind is the best fit. Kong's proxy is exposed as a `NodePort`
mapped to `localhost:8000` via kind `extraPortMappings`.
- **Kong Ingress Controller** (DB-less): the cluster edge. Beyond routing, the
  chart attaches a `rate-limiting` `KongPlugin` — a concrete example of the
  protection a high-criticality API needs. Kong is the natural on-cluster analog
  of an API Gateway / managed edge in the AWS design.

`kind/setup.sh` performs the whole flow: create cluster → build image → load it
into kind → `helm install` Kong → `helm install` the app → wait for rollout.

---

## Trade-offs & "what I'd do next"

- **Migrations**: a single idempotent `CREATE TABLE IF NOT EXISTS` runs at
  startup. For more tables I'd use `golang-migrate` run as an init container or
  Helm pre-install hook.
- **Observability**: structured JSON logs are in; next steps are a Prometheus
  `/metrics` endpoint (+ ServiceMonitor) and OpenTelemetry tracing through Kong.
- **Secrets**: local dev renders a Secret from values; production uses
  `existingSecret`, populated by the External Secrets Operator from AWS Secrets
  Manager so no credential ever touches Git.
- **Auth**: the endpoints are open — the spec doesn't mention auth, so I left it
  out. Kong makes it a one-plugin change to add API keys / JWT / OIDC at the edge.
