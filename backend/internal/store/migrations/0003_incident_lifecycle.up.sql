ALTER TABLE incidents
    ADD COLUMN IF NOT EXISTS closure_state TEXT NOT NULL DEFAULT 'open',
    ADD COLUMN IF NOT EXISTS closure_reason TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS closure_criteria JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS closure_ready_streak INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_incidents_closure_state ON incidents(closure_state);

CREATE TABLE IF NOT EXISTS runbook_executions (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    incident_id BIGINT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    runbook_name TEXT NOT NULL,
    runbook_version_hash TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NULL,
    trigger TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_runbook_executions_incident ON runbook_executions(incident_id);
CREATE INDEX IF NOT EXISTS idx_runbook_executions_status ON runbook_executions(status);

CREATE TABLE IF NOT EXISTS runbook_execution_steps (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    execution_id BIGINT NOT NULL REFERENCES runbook_executions(id) ON DELETE CASCADE,
    step_name TEXT NOT NULL,
    step_order INTEGER NOT NULL,
    kind TEXT NOT NULL,
    status TEXT NOT NULL,
    output TEXT NOT NULL DEFAULT '',
    error TEXT NOT NULL DEFAULT '',
    recommendation TEXT NOT NULL DEFAULT '',
    required BOOLEAN NOT NULL DEFAULT FALSE,
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    UNIQUE (execution_id, step_order)
);

CREATE INDEX IF NOT EXISTS idx_execution_steps_execution ON runbook_execution_steps(execution_id);
CREATE INDEX IF NOT EXISTS idx_execution_steps_status ON runbook_execution_steps(status);

ALTER TABLE approval_requests
    ADD COLUMN IF NOT EXISTS execution_id BIGINT NULL REFERENCES runbook_executions(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS step_order INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_approval_requests_execution ON approval_requests(execution_id);

CREATE TABLE IF NOT EXISTS incident_events (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    incident_id BIGINT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    message TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_incident_events_incident_id ON incident_events(incident_id);
CREATE INDEX IF NOT EXISTS idx_incident_events_event_type ON incident_events(event_type);

CREATE TABLE IF NOT EXISTS gitops_changes (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    change_type TEXT NOT NULL,
    title TEXT NOT NULL,
    status TEXT NOT NULL,
    branch TEXT NOT NULL DEFAULT '',
    pr_url TEXT NOT NULL DEFAULT '',
    pr_number BIGINT NOT NULL DEFAULT 0,
    desired JSONB NOT NULL DEFAULT '{}'::jsonb,
    applied JSONB NOT NULL DEFAULT '{}'::jsonb,
    drift_status TEXT NOT NULL DEFAULT 'unknown',
    error TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_gitops_changes_status ON gitops_changes(status);

ALTER TABLE delivery_states
    ADD COLUMN IF NOT EXISTS last_execution_hash TEXT NOT NULL DEFAULT '';
