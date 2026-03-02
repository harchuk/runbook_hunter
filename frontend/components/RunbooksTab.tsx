import { useEffect, useMemo, useState } from 'react';
import { fetchIncident, fetchIncidents } from '../lib/api';

export default function RunbooksTab() {
  const [incidents, setIncidents] = useState<any[]>([]);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [detail, setDetail] = useState<any>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    fetchIncidents()
      .then((rows) => {
        setIncidents(rows || []);
        const first = rows?.[0];
        const id = Number(first?.id ?? first?.ID ?? 0);
        if (id > 0) {
          setSelectedId(id);
        }
      })
      .catch((e: any) => setError(e.message));
  }, []);

  useEffect(() => {
    if (!selectedId) return;
    fetchIncident(selectedId)
      .then((payload) => setDetail(payload))
      .catch((e: any) => setError(e.message));
  }, [selectedId]);

  const execution = useMemo(() => {
    const list = detail?.executions ?? detail?.Executions ?? [];
    return list[0] || null;
  }, [detail]);

  const steps = execution?.steps ?? execution?.Steps ?? [];

  return (
    <section className="panel stack-16">
      <div className="section-head">
        <div>
          <h2 className="title">Runbooks</h2>
          <p className="hint">Runbook execution is incident-centric. This view highlights step status and recommendations.</p>
        </div>
        <select
          className="input"
          value={selectedId ?? ''}
          onChange={(e) => setSelectedId(Number(e.target.value) || null)}
          style={{ minWidth: 300 }}
        >
          {incidents.map((row) => {
            const id = Number(row.id ?? row.ID ?? 0);
            const label = row.alertName ?? row.AlertName ?? 'incident';
            return (
              <option key={id} value={id}>
                #{id} · {label}
              </option>
            );
          })}
        </select>
      </div>

      {error && <p className="security-note">{error}</p>}

      {!execution && <p className="hint">No runbook execution found for selected incident.</p>}
      {execution && (
        <div className="card stack-10">
          <h3>
            {execution.execution?.runbookName ?? execution.Execution?.RunbookName ?? 'runbook'}
          </h3>
          <p className="hint">
            status: {execution.execution?.status ?? execution.Execution?.Status ?? 'running'}
          </p>
          <ul className="step-list">
            {steps.map((step: any) => {
              const status = String(step.status ?? step.Status ?? 'pending');
              return (
                <li key={`${step.executionID ?? step.ExecutionID}-${step.stepOrder ?? step.StepOrder}`} className={`step ${status.replace(/_/g, '-')}`}>
                  <div className="step-head">
                    <input type="checkbox" checked={status === 'ok'} readOnly />
                    <strong>{step.stepName ?? step.StepName}</strong>
                    <span className={`badge state-${status.replace(/_/g, '-')}`}>{status}</span>
                  </div>
                  {(step.recommendation ?? step.Recommendation) && <p className="hint">{step.recommendation ?? step.Recommendation}</p>}
                  {(step.error ?? step.Error) && <p className="error-line">{step.error ?? step.Error}</p>}
                </li>
              );
            })}
          </ul>
        </div>
      )}
    </section>
  );
}
