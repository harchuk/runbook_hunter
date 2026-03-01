# AGENTS.md

## Mandatory quality gate

When changing code in this repository, agents must keep end-to-end scenarios valid.

### E2E suite

- Primary E2E script: `tests/e2e/run.sh`
- Scope: dockerized app startup, API health/readiness, Alertmanager ingest, incidents API auth flow, settings override save/reset, UI availability.

### Required behavior for every agent

1. If changes touch `backend/`, `frontend/`, `docker-compose.yml`, `examples/`, or deployment/runtime config, run:
   - `tests/e2e/run.sh`
2. If E2E fails, fix code and/or tests in the same change set.
3. Do not merge changes that break E2E expectations.
4. When intentional behavior changes affect E2E assertions, update E2E tests accordingly and document it in the PR/change summary.

### Additional checks

- Backend unit tests: `cd backend && go test ./...`
- Helm lint: `helm lint deploy/helm/runbook-hunter`

Security notes:
- Do not log secrets when debugging E2E failures.
- Keep read-only execution behavior intact in MVP.
