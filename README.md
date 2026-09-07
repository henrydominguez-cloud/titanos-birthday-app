# Titan OS — Infrastructure Take-Home

A small, production-shaped HTTP service that stores a user's date of birth and
returns a birthday-countdown message, packaged as a **Helm chart** and deployed
to a **local Kubernetes cluster (kind)** fronted by the **Kong Ingress
Controller**. Part 3 (AWS design) lives in [`docs/aws-architecture.md`](docs/aws-architecture.md).

```
        ┌─────────┐        ┌──────────────────────── kind cluster ─────────────────────────┐
 curl ─▶│  Kong    │─Ingress▶│  Service ─▶ Deployment (birthday-app, 2+ pods, HPA)  ─▶  Postgres │
        │ (proxy)  │        │            /hello/{username}                     (StatefulSet)   │
        └─────────┘        └────────────────────────────────────────────────────────────────┘
      localhost:8000
```

## The API

| Method | Path                 | Body                              | Success | Notes |
|--------|----------------------|-----------------------------------|---------|-------|
| `PUT`  | `/hello/{username}`  | `{"dateOfBirth":"YYYY-MM-DD"}`    | `204`   | username = letters only; date must be **before** today |
| `GET`  | `/hello/{username}`  | —                                 | `200`   | returns the birthday message |
| `GET`  | `/healthz`           | —                                 | `200`   | liveness |
| `GET`  | `/readyz`            | —                                 | `200`   | readiness (checks the DB) |

Message rules:

- Birthday today → `{"message":"Hello, <username>! Happy birthday!"}`
- Otherwise → `{"message":"Hello, <username>! Your birthday is in N day(s)"}`

> Kong only routes the public path `/hello`. `/healthz` and `/readyz` are internal
> (used by the kubelet probes), so they are reached directly on the pod, not through
> the gateway — hitting them via Kong returns a 404 by design.

## Quick start

Prereqs: `docker` (or colima), `kind`, `kubectl`, `helm`. No local Go needed —
it builds inside the container.

```bash
make test     # run Go unit tests (in a golang container)
make up       # kind cluster + Kong + build/load/deploy the app  (~5 min)
make smoke    # exercise the API through Kong on localhost:8000
make down     # tear everything down
```

After `make up`:

```bash
curl -i -X PUT http://localhost:8000/hello/jdoe \
  -H 'Host: birthday.local' -H 'Content-Type: application/json' \
  -d '{"dateOfBirth":"1990-09-10"}'          # -> HTTP/1.1 204 No Content

curl -s http://localhost:8000/hello/jdoe -H 'Host: birthday.local'
# {"message":"Hello, jdoe! Your birthday is in N day(s)"}
```

## Layout

```
app/                    Go service (net/http, stdlib router, pgx)
  cmd/server/           entrypoint (graceful shutdown, DB retry)
  internal/birthday/    pure date logic — fully unit-tested, no I/O
  internal/store/       Store interface + Postgres and in-memory impls
  internal/api/         HTTP handlers, validation, middleware
  Dockerfile           multi-stage -> distroless static, non-root
helm/birthday-app/      Helm chart (see docs/DESIGN.md for the component list)
kind/                   kind cluster config + one-command setup/teardown
scripts/smoke.sh        host-side smoke test through Kong
docs/
  DESIGN.md             decisions + "what a Helm chart needs to deploy an app"
  aws-architecture.md   Part 3: AWS diagram + how to build it
.github/workflows/      CI: Go tests + helm lint
```

## Scope & trade-offs

This is deliberately scoped to a small take-home rather than a full production
chart. Kept: two endpoints with validation and a real database, a container,
and the Helm pieces that matter most (Deployment, Service, Kong Ingress,
Config/Secret, probes, resource limits, HPA, a Helm test). Left out on purpose,
and easy to add: a `PodDisruptionBudget`, a `NetworkPolicy`, DB migrations
tooling, and a Prometheus `/metrics` endpoint. `docs/DESIGN.md` explains each
decision.

## Design choices (short version)

- **Go + stdlib** — tiny (~15 MB distroless) static image, fast start, ideal for
  Kubernetes. Business logic is isolated in `internal/birthday` so it is trivial
  to test without a database or HTTP server.
- **PostgreSQL** as the store — maps cleanly to **RDS/Aurora** in the AWS design.
  The chart ships a self-contained Postgres `StatefulSet` for local runs; in
  production you flip `postgresql.enabled=false` and point at a managed DB.
- **Kong Ingress Controller** as the edge — L7 routing plus a `rate-limiting`
  plugin, which is exactly the kind of resiliency layer a high-usage API needs.

Full rationale is in [`docs/DESIGN.md`](docs/DESIGN.md).

## On AI usage

The take-home allows it, and I used it transparently: **AI** for the standard,
repetitive parts (Helm template syntax, the multi-stage Dockerfile, test
scaffolding) and to move faster; **me** for the decisions and logic — the date
math, validation, choosing PostgreSQL and Kong, and what to scope out. For the
AWS design, where I work less than in GCP, I reasoned about the architecture in
cloud-neutral terms and used AI to map it to exact AWS services; the AWS↔GCP
equivalence in [`docs/aws-architecture.md`](docs/aws-architecture.md) makes clear
the design is my own.
