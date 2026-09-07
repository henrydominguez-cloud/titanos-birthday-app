#!/usr/bin/env bash
set -euo pipefail
CLUSTER="${CLUSTER:-titanos}"
echo "Deleting kind cluster '$CLUSTER'"
kind delete cluster --name "$CLUSTER"
