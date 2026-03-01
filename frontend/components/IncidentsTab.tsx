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
  updatedAt: string;
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
    runbookName: raw.runbookName ?? raw.RunbookName ?? 'n/a',
    updatedAt: raw.updatedAt ?? raw.UpdatedAt ?? ''
  };
}

function normalizeDetails(raw: any): IncidentDetails {
  return {
    incident: raw.incident ?? raw.Incident ?? {},
    steps: raw.steps ?? raw.Steps ?? []
  };
}

function relativeTime(value: string): string {
  const ts = Date.parse(value);
  if (!Number.isFinite(ts)) {
    return 'n/a';
  }
  const diffMin = Math.round((Date.now() - ts) / 60000);
  if (diffMin < 1) {
    return 'just now';
  }
  if (diffMin < 60) {
    return `${diffMin}m ago`;
  }
  const diffH = Math.round(diffMin / 60);
  if (diffH < 48) {
    return `${diffH}h ago`;
  }
  const diffD = Math.round(diffH / 24);
  return `${diffD}d ago`;
}

export default function IncidentsTab() {
  const [items, setItems] = useState<IncidentRow[]>([]);
  const [selected, setSelected] = useState<IncidentDetails | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [busyDemo, setBusyDemo] = useState(false);
  const [query, setQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [envFilter, setEnvFilter] = useState('all');
  const [severityFilter, setSeverityFilter] = useState('all');
  const [showRaw, setShowRaw] = useState(false);

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

  const openCount = useMemo(() => items.filter((item) => item.status === 'open').length, [items]);
  const criticalCount = useMemo(() => items.filter((item) => item.severity === 'critical').length, [items]);
  const recentCount = useMemo(() => items.filter((item) => relativeTime(item.updatedAt).includes('m ago')).length, [items]);

  const envOptions = useMemo(
    () => ['all', ...Array.from(new Set(items.map((item) => item.env))).sort()],
    [items]
  );
  const severityOptions = useMemo(
    () => ['all', ...Array.from(new Set(items.map((item) => item.severity))).sort()],
    [items]
  );

  const filtered = useMemo(() => {
    return [...items]
      .filter((item) => {
        if (statusFilter !== 'all' && item.status !== statusFilter) return false;
        if (envFilter !== 'all' && item.env !== envFilter) return false;
        if (severityFilter !== 'all' && item.severity !== severityFilter) return false;
        if (!query.trim()) return true;
        const haystack = `${item.id} ${item.alertName} ${item.service} ${item.env} ${item.brief}`.toLowerCase();
        return haystack.includes(query.toLowerCase());
      })
      .sort((a, b) => (a.updatedAt < b.updatedAt ? 1 : -1));
  }, [items, statusFilter, envFilter, severityFilter, query]);

  return (
    <section className="panel">
      <div className="section-head">
        <div>
          <h2 style={{ margin: 0 }}>Incidents</h2>
          <p className="hint" style={{ margin: '6px 0 0' }}>
            Correlated incident stream with runbook execution timeline and factual brief.
          </p>
        </div>
        <div className="button-row">
          <button className="btn" onClick={loadIncidents} disabled={loading}>Refresh</button>
          <button className="btn primary" onClick={onGenerateDemo} disabled={busyDemo || loading}>
            {busyDemo ? 'Generating…' : 'Generate demo incidents'}
          </button>
        </div>
      </div>

      <div className="stats-inline">
        <span className="mini-pill">Total: {items.length}</span>
        <span className="mini-pill">Open: {openCount}</span>
        <span className="mini-pill">Critical: {criticalCount}</span>
        <span className="mini-pill">Updated &lt; 60m: {recentCount}</span>
      </div>

      <div className="filters">
        <input
          className="input"
          placeholder="Search by alert/service/env/brief…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <select className="input" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
          <option value="all">Status: all</option>
          <option value="open">Status: open</option>
          <option value="resolved">Status: resolved</option>
        </select>
        <select className="input" value={envFilter} onChange={(e) => setEnvFilter(e.target.value)}>
          {envOptions.map((item) => (
            <option key={item} value={item}>
              {item === 'all' ? 'Env: all' : `Env: ${item}`}
            </option>
          ))}
        </select>
        <select className="input" value={severityFilter} onChange={(e) => setSeverityFilter(e.target.value)}>
          {severityOptions.map((item) => (
            <option key={item} value={item}>
              {item === 'all' ? 'Severity: all' : `Severity: ${item}`}
            </option>
          ))}
        </select>
      </div>

      {error && <p className="security-note">Error: {error}</p>}

      <div className="grid-2">
        <div className="row">
          {loading && <p className="hint">Loading incidents...</p>}
          <ul className="list">
            {filtered.map((it) => (
              <li key={it.id}>
                <button className="incident-btn" onClick={() => pickIncident(it.id)}>
                  <div className="incident-head">
                    <strong>#{it.id} · {it.alertName}</strong>
                    <span className={`pill ${it.status === 'resolved' ? 'resolved' : 'open'}`}>{it.status}</span>
                  </div>
                  <div className="hint">
                    {it.service} / {it.env} · severity: {it.severity} · updated {relativeTime(it.updatedAt)}
                  </div>
                  <div className="hint" style={{ marginTop: 6 }}>
                    Runbook: {it.runbookName || 'n/a'}
                  </div>
                  {it.brief && <div style={{ marginTop: 8 }}>{it.brief}</div>}
                </button>
              </li>
            ))}
          </ul>
        </div>

        <article className="row">
          {!selected && <p className="hint">Select an incident to inspect timeline, runbook and raw context.</p>}
          {selected && (
            <>
              <div className="panel panel-compact">
                <h3 style={{ margin: '0 0 8px' }}>Incident details</h3>
                <div className="hint">
                  Alert: {selected.incident.alertName ?? selected.incident.AlertName ?? 'unknown'} · Runbook:{' '}
                  {selected.incident.runbookName ?? selected.incident.RunbookName ?? 'n/a'}
                </div>
                <div style={{ marginTop: 8 }}>
                  {selected.incident.brief ?? selected.incident.Brief ?? 'No brief yet'}
                </div>
              </div>

              <div className="panel panel-compact">
                <h3 style={{ margin: '0 0 8px' }}>Step timeline</h3>
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
                        {(step.Error ?? step.error) && <div style={{ marginTop: 6, color: 'var(--danger)' }}>{step.Error ?? step.error}</div>}
                      </li>
                    ))}
                  </ul>
                )}
              </div>

              <div className="button-row">
                <button className="btn" onClick={() => setShowRaw((v) => !v)}>
                  {showRaw ? 'Hide raw JSON' : 'Show raw JSON'}
                </button>
              </div>
              {showRaw && <pre className="code-block">{JSON.stringify(selected, null, 2)}</pre>}
            </>
          )}
        </article>
      </div>
    </section>
  );
}
