#!/usr/bin/env bash
# Exercises the full request path through Kong (localhost:8000 -> Kong -> app).
set -euo pipefail
BASE="${BASE:-http://localhost:8000}"
HOST="${HOST:-birthday.local}"
USER="${USER_NAME:-jdoe}"

# Pick a birthday 5 days from today so the countdown is deterministic-ish.
DOB="$(date -v-30y -v+5d +%Y-%m-%d 2>/dev/null || date -d '+5 days -30 years' +%Y-%m-%d)"

echo "PUT /hello/$USER  (dateOfBirth=$DOB)"
curl -sS -i -X PUT "$BASE/hello/$USER" \
  -H "Host: $HOST" -H 'Content-Type: application/json' \
  -d "{\"dateOfBirth\":\"$DOB\"}" | head -n1

echo "GET /hello/$USER"
curl -sS "$BASE/hello/$USER" -H "Host: $HOST"; echo

echo "Validation: non-letter username should be 400"
curl -sS -o /dev/null -w "  status=%{http_code}\n" -X PUT "$BASE/hello/bad123" \
  -H "Host: $HOST" -H 'Content-Type: application/json' -d '{"dateOfBirth":"1990-01-01"}'
