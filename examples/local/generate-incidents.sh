#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
API_BASE="${1:-http://localhost:8080}"

echo "[demo] posting demo alerts to ${API_BASE}/api/alertmanager"

payloads=(
  "examples/payloads/demo-api-5xx.json"
  "examples/payloads/demo-latency-p95.json"
  "examples/payloads/demo-db-pool-exhausted.json"
)

for rel in "${payloads[@]}"; do
  file="${ROOT_DIR}/${rel}"
  if [[ ! -f "${file}" ]]; then
    echo "[demo][error] missing payload: ${file}" >&2
    exit 1
  fi
  echo "[demo] -> ${rel}"
  curl -fsS -X POST "${API_BASE}/api/alertmanager" \
    -H "Content-Type: application/json" \
    --data @"${file}" >/dev/null
done

echo "[demo] done. check incidents:"
echo "curl -s -u admin:change-me ${API_BASE}/api/incidents"
