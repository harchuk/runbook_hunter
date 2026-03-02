import { useEffect, useMemo, useState } from 'react';
import {
  closeIncident,
  fetchIncident,
  fetchIncidents,
  generateDemoIncidents,
  postUpdateNow,
  reopenIncident,
  rerunIncidentRunbook
} from '../lib/api';

type IncidentRow = {
  id: number;
  alertName: string;
  service: string;
  env: string;
  severity: string;
  status: string;
  closureState: string;
  alertsTotal: number;
  alertsFiring: number;
  closureReady: boolean;
  updatedAt: string;
  runbookName: string;
  brief: string;
};

function normalizeIncident(raw: any): IncidentRow {
  return {
    id: Number(raw.id ?? raw.ID ?? 0),
    alertName: raw.alertName ?? raw.AlertName ?? 'unknown',
    service: raw.service ?? raw.Service ?? 'unknown',
    env: raw.env ?? raw.Env ?? 'unknown',
    severity: raw.severity ?? raw.Severity ?? 'unknown',
    status: raw.status ?? raw.Status ?? 'open',
    closureState: raw.closureState ?? raw.ClosureState ?? 'open',
    alertsTotal: Number(raw.alerts_total ?? raw.AlertsTotal ?? 0),
    alertsFiring: Number(raw.alerts_firing ?? raw.AlertsFiring ?? 0),
    closureReady: Boolean(raw.closure_ready ?? raw.ClosureReady ?? false),
    updatedAt: raw.updatedAt ?? raw.UpdatedAt ?? '',
    runbookName: raw.runbookName ?? raw.RunbookName ?? 'n/a',
    brief: raw.brief ?? raw.Brief ?? ''
  };
}

function readCriteria(raw: any) {
  const defaultCriteria = {
    alerts_resolved: false,
    required_steps_passed: false,
    no_blockers: false,
    closure_ready: false
  };
  if (!raw) return defaultCriteria;
  return {
    alerts_resolved: Boolean(raw.alerts_resolved ?? raw.AlertsResolved ?? false),
    required_steps_passed: Boolean(raw.required_steps_passed ?? raw.RequiredStepsPassed ?? false),
    no_blockers: Boolean(raw.no_blockers ?? raw.NoBlockers ?? false),
    closure_ready: Boolean(raw.closure_ready ?? raw.ClosureReady ?? false)
  };
}

function formatAgo(value: string): string {
  const ts = Date.parse(value);
  if (!Number.isFinite(ts)) return 'n/a';
  const diffMin = Math.max(0, Math.round((Date.now() - ts) / 60000));
  if (diffMin < 1) return 'now';
  if (diffMin < 60) return `${diffMin}m`;
  const h = Math.round(diffMin / 60);
  if (h < 48) return `${h}h`;
  return `${Math.round(h / 24)}d`;
}

export default function IncidentsTab() {
  const [incidents, setIncidents] = useState<IncidentRow[]>([]);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [detail, setDetail] = useState<any>(null);
  const [query, setQuery] = useState('');
  const [status, setStatus] = useState('all');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const load = async () => {
    const rows = await fetchIncidents();
    const normalized = (rows || []).map(normalizeIncident);
    setIncidents(normalized);
    if (normalized.length > 0 && !selectedId) {
      setSelectedId(normalized[0].id);
    }
  };

  const loadDetail = async (id: number) => {
    const payload = await fetchIncident(id);
    setDetail(payload);
  };

  useEffect(() => {
    load().catch((e: any) => setError(e.message));
  }, []);

  useEffect(() => {
    if (!selectedId) return;
    loadDetail(selectedId).catch((e: any) => setError(e.message));
  }, [selectedId]);

  const filtered = useMemo(() => {
    return incidents
      .filter((it) => {
        if (status !== 'all' && it.closureState !== status && it.status !== status) return false;
        if (!query.trim()) return true;
        const haystack = `${it.id} ${it.alertName} ${it.service} ${it.env} ${it.brief}`.toLowerCase();
        return haystack.includes(query.toLowerCase());
      })
      .sort((a, b) => (a.updatedAt < b.updatedAt ? 1 : -1));
  }, [incidents, query, status]);

  const selectedIncident = useMemo(() => {
    if (!selectedId) return null;
    return incidents.find((item) => item.id === selectedId) || null;
  }, [incidents, selectedId]);

  const execution = useMemo(() => {
    const list = detail?.executions ?? detail?.Executions ?? [];
    return list[0] || null;
  }, [detail]);

  const steps = execution?.steps ?? execution?.Steps ?? [];
  const criteria = readCriteria(detail?.incident?.closureCriteria || detail?.incident?.ClosureCriteria);

  const runAction = async (fn: () => Promise<any>) => {
    if (!selectedId) return;
    setBusy(true);
    try {
      await fn();
      await load();
      await loadDetail(selectedId);
      setError('');
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <section className="panel stack-16">
      <div className="section-head">
        <div>
          <h2 className="title">Incidents</h2>
          <p className="hint">One incident aggregates many alerts. One active runbook execution lifecycle per incident.</p>
        </div>
        <div className="button-row">
          <button className="btn" onClick={() => load().catch((e: any) => setError(e.message))}>Refresh</button>
          <button
            className="btn"
            onClick={() =>
              runAction(async () => {
                await generateDemoIncidents();
              })
            }
          >
            Generate demo
          </button>
        </div>
      </div>

      <div className="toolbar">
        <input className="input" placeholder="Search incidents" value={query} onChange={(e) => setQuery(e.target.value)} />
        <select className="input" value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value="all">All states</option>
          <option value="open">Open</option>
          <option value="ready_to-close">Ready to close</option>
          <option value="closed">Closed</option>
          <option value="reopened">Reopened</option>
        </select>
      </div>

      {error && <p className="security-note">{error}</p>}

      <div className="workspace-grid">
        <div className="list-panel">
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>ID</th>
                  <th>Alert</th>
                  <th>Scope</th>
                  <th>State</th>
                  <th>Alerts</th>
                  <th>Updated</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((it) => (
                  <tr key={it.id} className={selectedId === it.id ? 'active' : ''} onClick={() => setSelectedId(it.id)}>
                    <td>#{it.id}</td>
                    <td>
                      <strong>{it.alertName}</strong>
                      <div className="hint">{it.severity}</div>
                    </td>
                    <td>{it.service}/{it.env}</td>
                    <td>
                      <span className={`badge state-${it.closureState.replace(/_/g, '-')}`}>{it.closureState}</span>
                    </td>
                    <td>{it.alertsFiring}/{it.alertsTotal}</td>
                    <td>{formatAgo(it.updatedAt)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        <div className="detail-panel stack-16">
          {!selectedIncident && <p className="hint">Select an incident.</p>}
          {selectedIncident && (
            <>
              <section className="card stack-10">
                <div className="section-head compact">
                  <h3>Incident #{selectedIncident.id}</h3>
                  <span className={`badge state-${selectedIncident.closureState.replace(/_/g, '-')}`}>{selectedIncident.closureState}</span>
                </div>
                <p className="hint">{selectedIncident.alertName} · {selectedIncident.service}/{selectedIncident.env}</p>
                <p>{selectedIncident.brief || 'No brief yet'}</p>
                <div className="button-row">
                  <button className="btn" disabled={busy} onClick={() => runAction(() => postUpdateNow(selectedIncident.id, []))}>Post update</button>
                  <button className="btn" disabled={busy} onClick={() => runAction(() => rerunIncidentRunbook(selectedIncident.id))}>Rerun runbook</button>
                  <button className="btn" disabled={busy} onClick={() => runAction(() => closeIncident(selectedIncident.id, 'manual close from UI'))}>Close</button>
                  <button className="btn" disabled={busy} onClick={() => runAction(() => reopenIncident(selectedIncident.id, 'manual reopen from UI'))}>Reopen</button>
                </div>
              </section>

              <section className="card stack-10">
                <h3>Closure Criteria</h3>
                <div className="criteria-grid">
                  <label className={`criteria ${criteria.alerts_resolved ? 'ok' : 'fail'}`}>
                    <input type="checkbox" checked={criteria.alerts_resolved} readOnly />
                    Alerts resolved
                  </label>
                  <label className={`criteria ${criteria.required_steps_passed ? 'ok' : 'fail'}`}>
                    <input type="checkbox" checked={criteria.required_steps_passed} readOnly />
                    Required steps passed
                  </label>
                  <label className={`criteria ${criteria.no_blockers ? 'ok' : 'fail'}`}>
                    <input type="checkbox" checked={criteria.no_blockers} readOnly />
                    No blockers
                  </label>
                </div>
              </section>

              <section className="card stack-10">
                <h3>Runbook Progress</h3>
                {!execution && <p className="hint">No execution yet.</p>}
                {execution && (
                  <>
                    <p className="hint">
                      {execution.execution?.runbookName ?? execution.Execution?.RunbookName ?? selectedIncident.runbookName} · status:{' '}
                      {execution.execution?.status ?? execution.Execution?.Status ?? 'running'}
                    </p>
                    <ul className="step-list">
                      {steps.map((step: any) => {
                        const statusLabel = String(step.status ?? step.Status ?? 'pending');
                        return (
                          <li key={`${step.executionID ?? step.ExecutionID}-${step.stepOrder ?? step.StepOrder}`} className={`step ${statusLabel.replace(/_/g, '-')}`}>
                            <div className="step-head">
                              <span className={`dot ${statusLabel.replace(/_/g, '-')}`} />
                              <strong>{step.stepName ?? step.StepName}</strong>
                              <span className="badge mono">{statusLabel}</span>
                            </div>
                            {(step.recommendation ?? step.Recommendation) && (
                              <p className="hint">Recommendation: {step.recommendation ?? step.Recommendation}</p>
                            )}
                            {(step.error ?? step.Error) && <p className="error-line">{step.error ?? step.Error}</p>}
                            {(step.output ?? step.Output) && <p className="hint">{String(step.output ?? step.Output).slice(0, 260)}</p>}
                          </li>
                        );
                      })}
                    </ul>
                  </>
                )}
              </section>
            </>
          )}
        </div>
      </div>
    </section>
  );
}
