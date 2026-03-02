import { useEffect, useMemo, useState } from 'react';
import {
  approveRequest,
  fetchAlerts,
  fetchApprovals,
  fetchIncidents,
  generateDemoIncidents,
  rejectRequest
} from '../lib/api';

export default function OverviewTab() {
  const [incidents, setIncidents] = useState<any[]>([]);
  const [alerts, setAlerts] = useState<any[]>([]);
  const [approvals, setApprovals] = useState<any[]>([]);
  const [error, setError] = useState('');
  const [busyDemo, setBusyDemo] = useState(false);

  const load = async () => {
    const [inc, al, app] = await Promise.all([fetchIncidents(), fetchAlerts(), fetchApprovals()]);
    setIncidents(inc || []);
    setAlerts(al || []);
    setApprovals(app || []);
  };

  useEffect(() => {
    load().catch((e: any) => setError(e.message));
  }, []);

  const stats = useMemo(() => {
    const open = incidents.filter((row) => (row.closureState ?? row.ClosureState ?? 'open') !== 'closed').length;
    const ready = incidents.filter((row) => Boolean(row.closure_ready ?? row.ClosureReady ?? false)).length;
    const blocked = incidents.filter((row) => {
      const closure = row.closureState ?? row.ClosureState ?? '';
      return closure === 'reopened' || closure === 'open';
    }).length;
    const firingAlerts = alerts.filter((row) => (row.status ?? row.Status) === 'firing').length;
    return { open, ready, blocked, firingAlerts };
  }, [incidents, alerts]);

  return (
    <section className="panel stack-16">
      <div className="section-head">
        <div>
          <h2 className="title">Overview</h2>
          <p className="hint">Operator entrypoint: open incidents, fresh alerts, closure readiness, approvals queue.</p>
        </div>
        <div className="button-row">
          <button className="btn" onClick={() => load().catch((e: any) => setError(e.message))}>Refresh</button>
          <button
            className="btn"
            disabled={busyDemo}
            onClick={async () => {
              setBusyDemo(true);
              try {
                await generateDemoIncidents();
                await load();
                setError('');
              } catch (e: any) {
                setError(e.message);
              } finally {
                setBusyDemo(false);
              }
            }}
          >
            {busyDemo ? 'Generating…' : 'Generate demo incidents'}
          </button>
        </div>
      </div>

      <div className="kpi-grid">
        <article className="kpi-card">
          <span className="hint">Open incidents</span>
          <strong>{stats.open}</strong>
        </article>
        <article className="kpi-card">
          <span className="hint">Firing alerts</span>
          <strong>{stats.firingAlerts}</strong>
        </article>
        <article className="kpi-card">
          <span className="hint">Ready to close</span>
          <strong>{stats.ready}</strong>
        </article>
        <article className="kpi-card">
          <span className="hint">Blocked / active</span>
          <strong>{stats.blocked}</strong>
        </article>
      </div>

      {error && <p className="security-note">{error}</p>}

      <section className="card stack-10">
        <h3>Pending approvals</h3>
        {approvals.length === 0 ? (
          <p className="hint">No pending approvals.</p>
        ) : (
          <ul className="plain-list">
            {approvals.map((item) => (
              <li key={item.id ?? item.ID} className="approval-item">
                <span>
                  #{item.id ?? item.ID} · incident #{item.incidentId ?? item.IncidentID} · action {item.action ?? item.Action}
                </span>
                <span className="button-row">
                  <button
                    className="btn"
                    onClick={async () => {
                      try {
                        await approveRequest(Number(item.id ?? item.ID));
                        await load();
                      } catch (e: any) {
                        setError(e.message);
                      }
                    }}
                  >
                    Approve
                  </button>
                  <button
                    className="btn"
                    onClick={async () => {
                      try {
                        await rejectRequest(Number(item.id ?? item.ID), 'rejected from overview');
                        await load();
                      } catch (e: any) {
                        setError(e.message);
                      }
                    }}
                  >
                    Reject
                  </button>
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>

      <p className="hint">
        Security notes: executor checks remain read-only; ansible action requires explicit approval and AWX credentials from Secret.
      </p>
    </section>
  );
}
