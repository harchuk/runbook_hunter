ALTER TABLE delivery_states DROP COLUMN IF EXISTS last_execution_hash;

DROP TABLE IF EXISTS gitops_changes;
DROP TABLE IF EXISTS incident_events;

ALTER TABLE approval_requests
    DROP COLUMN IF EXISTS step_order,
    DROP COLUMN IF EXISTS execution_id;

DROP TABLE IF EXISTS runbook_execution_steps;
DROP TABLE IF EXISTS runbook_executions;

DROP INDEX IF EXISTS idx_incidents_closure_state;
ALTER TABLE incidents
    DROP COLUMN IF EXISTS closure_ready_streak,
    DROP COLUMN IF EXISTS resolved_at,
    DROP COLUMN IF EXISTS closure_criteria,
    DROP COLUMN IF EXISTS closure_reason,
    DROP COLUMN IF EXISTS closure_state;
