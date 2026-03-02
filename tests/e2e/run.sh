#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

AUTH_USER="${E2E_AUTH_USER:-admin}"
AUTH_PASS="${E2E_AUTH_PASS:-change-me}"
API_BASE="${E2E_API_BASE:-http://localhost:8080}"
UI_BASE="${E2E_UI_BASE:-http://localhost:3000}"

log() {
  echo "[e2e] $*"
}

wait_for_http() {
  local name="$1"
  local url="$2"
  local attempts="${3:-60}"
  local sleep_sec="${4:-2}"

  for _ in $(seq 1 "$attempts"); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      log "$name is ready: $url"
      return 0
    fi
    sleep "$sleep_sec"
  done

  log "$name is not ready: $url"
  return 1
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local message="$3"

  if ! grep -q "$needle" <<<"$haystack"; then
    echo "[e2e][FAIL] $message"
    echo "[e2e][FAIL] expected to find: $needle"
    exit 1
  fi
}

wait_for_match() {
  local name="$1"
  local command="$2"
  local needle="$3"
  local attempts="${4:-45}"
  local sleep_sec="${5:-2}"

  for _ in $(seq 1 "$attempts"); do
    local out
    out="$(eval "$command")"
    if grep -q "$needle" <<<"$out"; then
      log "$name matched: $needle"
      return 0
    fi
    sleep "$sleep_sec"
  done

  log "$name did not match: $needle"
  return 1
}

incident_id_by_alertname() {
  local alert_name="$1"
  local body="$2"
  echo "$body" \
    | tr '{' '\n' \
    | grep "\"alertName\":\"${alert_name}\"" \
    | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p' \
    | head -n1
}

log "Building and starting containers"
docker compose up -d --build

wait_for_http "API health" "$API_BASE/healthz"
wait_for_http "API ready" "$API_BASE/readyz"
wait_for_http "UI" "$UI_BASE"

log "Posting Alertmanager sample payload"
post_resp="$(curl -fsS -X POST "$API_BASE/api/alertmanager" -H 'Content-Type: application/json' --data @examples/payloads/alertmanager-sample.json)"
assert_contains "$post_resp" '"processed"' "alertmanager ingest response did not include processed field"

log "Posting demo payload set"
./examples/local/generate-incidents.sh "$API_BASE" >/dev/null

log "Checking incidents list with auth"
incidents_resp="$(curl -fsS -u "$AUTH_USER:$AUTH_PASS" "$API_BASE/api/incidents")"
assert_contains "$incidents_resp" '"id":' "incidents list should contain at least one incident"
assert_contains "$incidents_resp" '"alertName":"DemoAPI5xxSpike"' "demo API incident should be present"
assert_contains "$incidents_resp" '"alertName":"DemoLatencyP95Spike"' "demo latency incident should be present"
assert_contains "$incidents_resp" '"alertName":"DemoDBPoolExhausted"' "demo DB incident should be present"

demo_api_id="$(incident_id_by_alertname "DemoAPI5xxSpike" "$incidents_resp")"
if [[ -z "${demo_api_id}" ]]; then
  echo "[e2e][FAIL] unable to find incident id for DemoAPI5xxSpike"
  exit 1
fi

log "Waiting for worker step runs on demo incident ${demo_api_id}"
wait_for_match \
  "demo incident step run" \
  "curl -fsS -u \"$AUTH_USER:$AUTH_PASS\" \"$API_BASE/api/incidents/${demo_api_id}\"" \
  '"Status":"ok"' \
  60 \
  2

demo_detail="$(curl -fsS -u "$AUTH_USER:$AUTH_PASS" "$API_BASE/api/incidents/${demo_api_id}")"
assert_contains "$demo_detail" '"StepName":"API health endpoint"' "demo runbook step should be recorded"
assert_contains "$demo_detail" '"Status":"ok"' "demo runbook should produce at least one successful step"

log "Checking alerts API"
alerts_resp="$(curl -fsS -u "$AUTH_USER:$AUTH_PASS" "$API_BASE/api/alerts")"
assert_contains "$alerts_resp" '"incidentId":' "alerts list should include incident linkage"

log "Checking runbook executions API"
exec_resp="$(curl -fsS -u "$AUTH_USER:$AUTH_PASS" "$API_BASE/api/incidents/${demo_api_id}/runbook-executions")"
assert_contains "$exec_resp" '"execution"' "incident runbook executions endpoint should return execution payload"

log "Checking closure criteria API"
criteria_resp="$(curl -fsS -u "$AUTH_USER:$AUTH_PASS" "$API_BASE/api/incidents/${demo_api_id}/closure-criteria")"
assert_contains "$criteria_resp" '"alerts_resolved"' "closure criteria response should include alerts_resolved"

log "Checking manual close/reopen flow"
curl -fsS -u "$AUTH_USER:$AUTH_PASS" -X POST "$API_BASE/api/incidents/${demo_api_id}/close" \
  -H 'Content-Type: application/json' \
  -d '{"reason":"e2e close"}' >/dev/null
closed_detail="$(curl -fsS -u "$AUTH_USER:$AUTH_PASS" "$API_BASE/api/incidents/${demo_api_id}")"
assert_contains "$closed_detail" '"closureState":"closed"' "incident should be closed after manual close"

curl -fsS -u "$AUTH_USER:$AUTH_PASS" -X POST "$API_BASE/api/incidents/${demo_api_id}/reopen" \
  -H 'Content-Type: application/json' \
  -d '{"reason":"e2e reopen"}' >/dev/null
reopened_detail="$(curl -fsS -u "$AUTH_USER:$AUTH_PASS" "$API_BASE/api/incidents/${demo_api_id}")"
assert_contains "$reopened_detail" '"closureState":"reopened"' "incident should be reopened after manual reopen"

log "Checking GitOps change request API"
change_payload='{"change_type":"settings","title":"e2e change","desired":{"routing":{"rules":[]}}}'
curl -fsS -u "$AUTH_USER:$AUTH_PASS" -X POST "$API_BASE/api/gitops/changes" \
  -H 'Content-Type: application/json' \
  -d "$change_payload" >/dev/null
changes_resp="$(curl -fsS -u "$AUTH_USER:$AUTH_PASS" "$API_BASE/api/gitops/changes")"
assert_contains "$changes_resp" '"Title":"e2e change"' "gitops changes list should include created change request"

log "Checking approvals API availability"
approvals_resp="$(curl -fsS -u "$AUTH_USER:$AUTH_PASS" "$API_BASE/api/approvals")"
if [[ "$approvals_resp" != \[* ]]; then
  echo "[e2e][FAIL] approvals endpoint should return JSON array"
  echo "[e2e][FAIL] got: $approvals_resp"
  exit 1
fi

log "Saving UI override"
overrides_payload='{"server":{"addr":":9099"}}'
curl -fsS -u "$AUTH_USER:$AUTH_PASS" -X PUT "$API_BASE/api/settings/overrides" \
  -H 'Content-Type: application/json' \
  -d "$overrides_payload" >/dev/null

effective_after_put="$(curl -fsS -u "$AUTH_USER:$AUTH_PASS" "$API_BASE/api/settings/effective")"
assert_contains "$effective_after_put" '"addr":":9099"' "effective settings should include override addr"

log "Resetting overrides"
curl -fsS -u "$AUTH_USER:$AUTH_PASS" -X POST "$API_BASE/api/settings/overrides/reset" \
  -H 'Content-Type: application/json' \
  -d '{"scope":"all"}' >/dev/null

effective_after_reset="$(curl -fsS -u "$AUTH_USER:$AUTH_PASS" "$API_BASE/api/settings/effective")"
assert_contains "$effective_after_reset" '"addr":":8080"' "effective settings should return to base addr after reset"

log "Checking UI markup"
ui_html="$(curl -fsS "$UI_BASE")"
assert_contains "$ui_html" 'Runbook Hunter Console' "UI homepage should include hero heading"
assert_contains "$ui_html" 'Changes (GitOps)' "UI should expose GitOps changes tab"
assert_contains "$ui_html" 'Alerts' "UI should expose alerts tab"

log "E2E smoke passed"
