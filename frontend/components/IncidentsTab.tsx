import { useEffect, useMemo, useState } from 'react';
import { fetchIncident, fetchIncidents, generateDemoIncidents } from '../lib/api';

type IncidentRow = {
  id: number;
  alertName: string;
  service: string;
  env: string;
  status: string;
  severity: string;
  brief: string;
  runbookName: string;
};

type IncidentDetails = {
  incident: any;
  steps: any[];
};

function normalizeIncidentRow(raw: any): IncidentRow {
  return {
    id: Number(raw.id ?? raw.ID ?? 0),
    alertName: raw.alertName ?? raw.AlertName ?? 'unknown',
    service: raw.service ?? raw.Service ?? 'unknown',
    env: raw.env ?? raw.Env ?? 'unknown',
    status: raw.status ?? raw.Status ?? 'open',
    severity: raw.severity ?? raw.Severity ?? 'unknown',
    brief: raw.brief ?? raw.Brief ?? '',
    runbookName: raw.runbookName ?? raw.RunbookName ?? ''
  };
}

function normalizeDetails(raw: any): IncidentDetails {
  return {
    incident: raw.incident ?? raw.Incident ?? {},
    steps: raw.steps ?? raw.Steps ?? []
  };
}

export default function IncidentsTab() {
  const [items, setItems] = useState<IncidentRow[]>([]);
  const [selected, setSelected] = useState<IncidentDetails | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [busyDemo, setBusyDemo] = useState(false);

  const openCount = useMemo(() => items.filter((item) => item.status === 'open').length, [items]);

  const loadIncidents = async () => {
    setLoading(true);
    try {
      const rows = await fetchIncidents();
      setItems((rows as any[]).map(normalizeIncidentRow));
      setError('');
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadIncidents();
  }, []);

  const onGenerateDemo = async () => {
    setBusyDemo(true);
    try {
      await generateDemoIncidents();
      await new Promise((resolve) => setTimeout(resolve, 1200));
      await loadIncidents();
      setError('');
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusyDemo(false);
    }
  };

  const pickIncident = async (id: number) => {
    try {
      const details = await fetchIncident(id);
      setSelected(normalizeDetails(details));
      setError('');
    } catch (e: any) {
      setError(e.message);
    }
  };

  return (
    <section className="panel">
      <div className="incident-head" style={{ marginBottom: 12 }}>
        <h2 style={{ margin: 0 }}>Incidents</h2>
        <div className="button-row">
          <button className="btn" onClick={loadIncidents} disabled={loading}>Refresh</button>
          <button className="btn primary" onClick={onGenerateDemo} disabled={busyDemo || loading}>
            {busyDemo ? 'Generating…' : 'Generate demo incidents'}
          </button>
          <div className="hint">Open: {openCount} / Total: {items.length}</div>
        </div>
      </div>

      {error && <p className="security-note">Error: {error}</p>}
      <div className="grid-2">
        <div className="row">
          {loading && <p className="hint">Loading incidents...</p>}
          <ul className="list">
            {items.map((it) => (
              <li key={it.id}>
                <button className="incident-btn" onClick={() => pickIncident(it.id)}>
                  <div className="incident-head">
                    <strong>#{it.id} · {it.alertName}</strong>
                    <span className={`pill ${it.status === 'resolved' ? 'resolved' : 'open'}`}>{it.status}</span>
                  </div>
                  <div className="hint">{it.service} / {it.env} · severity: {it.severity}</div>
                  {it.brief && <div style={{ marginTop: 8 }}>{it.brief}</div>}
                </button>
              </li>
            ))}
          </ul>
        </div>

        <article className="row">
          {!selected && <p className="hint">Select incident to inspect timeline, runbook and step runs.</p>}
          {selected && (
            <>
              <div className="panel" style={{ padding: 12 }}>
                <h3 style={{ marginTop: 0, marginBottom: 8 }}>Incident details</h3>
                <div className="hint">
                  Alert: {selected.incident.alertName ?? selected.incident.AlertName ?? 'unknown'} · Runbook:{' '}
                  {selected.incident.runbookName ?? selected.incident.RunbookName ?? 'n/a'}
                </div>
                <div style={{ marginTop: 8 }}>
                  {selected.incident.brief ?? selected.incident.Brief ?? 'No brief yet'}
                </div>
              </div>

              <div className="panel" style={{ padding: 12 }}>
                <h3 style={{ marginTop: 0, marginBottom: 8 }}>Step timeline</h3>
                {selected.steps.length === 0 ? (
                  <p className="hint">No step runs yet.</p>
                ) : (
                  <ul className="list">
                    {selected.steps.map((step: any) => (
                      <li key={`${step.ID ?? step.id}-${step.StepName ?? step.stepName}`} className="incident-btn">
                        <div className="incident-head">
                          <strong>{step.StepName ?? step.stepName}</strong>
                          <span className={`pill ${(step.Status ?? step.status) === 'ok' ? 'resolved' : 'open'}`}>
                            {step.Status ?? step.status}
                          </span>
                        </div>
                        <div className="hint">Tool: {step.Tool ?? step.tool}</div>
                        {(step.Output ?? step.output) && <div style={{ marginTop: 6 }}>{step.Output ?? step.output}</div>}
                        {(step.Error ?? step.error) && <div style={{ marginTop: 6, color: '#ffc2cc' }}>{step.Error ?? step.error}</div>}
                      </li>
                    ))}
                  </ul>
                )}
              </div>

              <pre className="code-block">{JSON.stringify(selected, null, 2)}</pre>
            </>
          )}
        </article>
      </div>
    </section>
  );
}
