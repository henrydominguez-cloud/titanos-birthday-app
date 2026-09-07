#!/usr/bin/env bash
# End-to-end local bring-up:
#   kind cluster -> build image -> load into kind -> install Kong -> deploy chart
#
# Requirements: docker (or colima), kind, kubectl, helm.
set -euo pipefail

CLUSTER="${CLUSTER:-titanos}"
IMAGE="${IMAGE:-birthday-app:0.1.0}"
APP_NS="${APP_NS:-birthday}"
KONG_NS="${KONG_NS:-kong}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

log() { printf "\n\033[1;36m==> %s\033[0m\n" "$*"; }

# 1. Cluster --------------------------------------------------------------
if kind get clusters | grep -qx "$CLUSTER"; then
  log "kind cluster '$CLUSTER' already exists"
else
  log "Creating kind cluster '$CLUSTER'"
  kind create cluster --name "$CLUSTER" --config "$ROOT/kind/kind-config.yaml"
fi

# 2. Build & load image ---------------------------------------------------
log "Building image $IMAGE"
# --provenance=false forces a single-platform image. Without it, BuildKit emits
# an OCI manifest list with attestations that `kind load` can't serve, leaving
# pods in ImagePullBackOff.
docker build --provenance=false -t "$IMAGE" "$ROOT/app"

log "Loading image into kind"
kind load docker-image "$IMAGE" --name "$CLUSTER"

# 3. Kong Ingress Controller (DB-less) -----------------------------------
log "Installing Kong Ingress Controller"
helm repo add kong https://charts.konghq.com >/dev/null 2>&1 || true
helm repo update >/dev/null
helm upgrade --install kong kong/kong \
  --namespace "$KONG_NS" --create-namespace \
  --set proxy.type=NodePort \
  --set proxy.http.nodePort=30080 \
  --set proxy.tls.nodePort=30443 \
  --set admin.enabled=false \
  --wait --timeout 5m

# 4. Deploy the application chart ----------------------------------------
log "Deploying birthday-app Helm chart"
helm upgrade --install birthday-app "$ROOT/helm/birthday-app" \
  --namespace "$APP_NS" --create-namespace \
  --set image.repository="${IMAGE%%:*}" \
  --set image.tag="${IMAGE##*:}" \
  --wait --timeout 5m

log "Waiting for rollout"
kubectl -n "$APP_NS" rollout status deploy/birthday-app --timeout=120s

log "Done! Try it:"
cat <<EOF

  # Save a birthday
  curl -i -X PUT http://localhost:8000/hello/jdoe \\
    -H 'Host: birthday.local' -H 'Content-Type: application/json' \\
    -d '{"dateOfBirth":"1990-09-10"}'

  # Read the greeting
  curl -s http://localhost:8000/hello/jdoe -H 'Host: birthday.local'; echo

  # Helm smoke test
  helm test birthday-app -n $APP_NS
EOF
