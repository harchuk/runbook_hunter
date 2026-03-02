import { useEffect, useMemo, useState } from 'react';
import { fetchAlerts } from '../lib/api';

type AlertRow = {
  id: number;
  incidentId: number;
  alertName: string;
  status: string;
  service: string;
  env: string;
  severity: string;
  startsAt: string;
  createdAt: string;
};

function toMap(value: any): Record<string, string> {
  if (!value) return {};
  if (typeof value === 'string') {
    try {
      return JSON.parse(value);
    } catch {
      return {};
    }
  }
  return value;
}

function normalize(raw: any): AlertRow {
  const labels = toMap(raw.labels ?? raw.Labels);
  return {
    id: Number(raw.id ?? raw.ID ?? 0),
    incidentId: Number(raw.incidentId ?? raw.IncidentID ?? 0),
    alertName: raw.alertName ?? raw.AlertName ?? labels.alertname ?? 'unknown',
    status: raw.status ?? raw.Status ?? 'unknown',
    service: labels.service || 'unknown',
    env: labels.env || 'unknown',
    severity: labels.severity || 'unknown',
    startsAt: raw.startsAt ?? raw.StartsAt ?? '',
    createdAt: raw.createdAt ?? raw.CreatedAt ?? ''
  };
}

export default function AlertsTab() {
  const [rows, setRows] = useState<AlertRow[]>([]);
  const [query, setQuery] = useState('');
  const [status, setStatus] = useState('all');
  const [error, setError] = useState('');

  const load = async () => {
    const payload = await fetchAlerts();
    setRows((payload || []).map(normalize));
  };

  useEffect(() => {
    load().catch((e: any) => setError(e.message));
  }, []);

  const filtered = useMemo(() => {
    return rows.filter((row) => {
      if (status !== 'all' && row.status !== status) return false;
      if (!query.trim()) return true;
      const haystack = `${row.id} ${row.alertName} ${row.service} ${row.env} ${row.severity}`.toLowerCase();
      return haystack.includes(query.toLowerCase());
    });
  }, [rows, query, status]);

  return (
    <section className="panel stack-16">
      <div className="section-head">
        <div>
          <h2 className="title">Alerts</h2>
          <p className="hint">Atomic Alertmanager signals. Linked incidents are shown in a separate column.</p>
        </div>
        <button className="btn" onClick={() => load().catch((e: any) => setError(e.message))}>Refresh</button>
      </div>

      <div className="toolbar">
        <input className="input" placeholder="Search alerts" value={query} onChange={(e) => setQuery(e.target.value)} />
        <select className="input" value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value="all">All statuses</option>
          <option value="firing">Firing</option>
          <option value="resolved">Resolved</option>
        </select>
      </div>

      {error && <p className="security-note">{error}</p>}

      <div className="table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Incident</th>
              <th>Alert</th>
              <th>Status</th>
              <th>Scope</th>
              <th>Severity</th>
              <th>Start</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((row) => (
              <tr key={row.id}>
                <td>{row.id}</td>
                <td>#{row.incidentId}</td>
                <td>{row.alertName}</td>
                <td><span className={`badge state-${row.status}`}>{row.status}</span></td>
                <td>{row.service}/{row.env}</td>
                <td>{row.severity}</td>
                <td>{row.startsAt ? new Date(row.startsAt).toLocaleString() : '-'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
