import { useEffect, useMemo, useState } from 'react';
import { fetchEffectiveSettings, fetchIncidents, postUpdateNow } from '../lib/api';

type IncidentLite = {
  id: number;
  label: string;
};

export default function PostUpdateDialog() {
  const [incidentId, setIncidentId] = useState('');
  const [selectedDestinations, setSelectedDestinations] = useState<string[]>([]);
  const [message, setMessage] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [incidents, setIncidents] = useState<IncidentLite[]>([]);
  const [destinations, setDestinations] = useState<string[]>([]);

  useEffect(() => {
    fetchIncidents()
      .then((rows: any[]) =>
        setIncidents(
          rows
            .map((row) => ({
              id: Number(row.id ?? row.ID ?? 0),
              label: `#${row.id ?? row.ID} · ${row.alertName ?? row.AlertName} (${row.env ?? row.Env})`
            }))
            .slice(0, 25)
        )
      )
      .catch(() => setIncidents([]));

    fetchEffectiveSettings()
      .then((cfg) => {
        const tg = (cfg?.destinations?.telegram || []).map((d: any) => d.id).filter(Boolean);
        const mm = (cfg?.destinations?.mattermost || []).map((d: any) => d.id).filter(Boolean);
        setDestinations([...tg, ...mm]);
      })
      .catch(() => setDestinations([]));
  }, []);

  const suggestion = useMemo(() => incidents[0], [incidents]);

  const toggleDestination = (id: string) => {
    setSelectedDestinations((current) => {
      if (current.includes(id)) {
        return current.filter((v) => v !== id);
      }
      return [...current, id];
    });
  };

  const onSubmit = async () => {
    setSubmitting(true);
    try {
      const id = Number(incidentId);
      if (!Number.isFinite(id) || id <= 0) {
        setMessage('Incident ID must be a positive number');
        return;
      }
      await postUpdateNow(id, selectedDestinations);
      setMessage('Update queued. Force mode bypasses cooldown and refreshes dedup state.');
    } catch (e: any) {
      setMessage(e.message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <section className="panel">
      <h2 style={{ marginTop: 0 }}>Post update now</h2>
      <p className="hint">
        Use this when you need an immediate status push during escalation. Worker still applies routing and updates
        dedup state per destination.
      </p>

      <div className="row" style={{ maxWidth: 720 }}>
        <label className="label">
          Incident ID
          <div className="button-row">
            <input className="input" value={incidentId} onChange={(e) => setIncidentId(e.target.value)} />
            {suggestion && (
              <button className="btn" onClick={() => setIncidentId(String(suggestion.id))}>
                Use latest {suggestion.label}
              </button>
            )}
          </div>
        </label>

        <label className="label">
          Destination filter (optional)
          {destinations.length === 0 ? (
            <div className="hint">No destination IDs detected in effective settings. Empty selection means routing rules.</div>
          ) : (
            <div className="checkbox-grid">
              {destinations.map((id) => (
                <label key={id} className="check-item">
                  <input
                    type="checkbox"
                    checked={selectedDestinations.includes(id)}
                    onChange={() => toggleDestination(id)}
                  />
                  <span>{id}</span>
                </label>
              ))}
            </div>
          )}
        </label>

        <div className="button-row">
          <button className="btn primary" onClick={onSubmit} disabled={submitting}>
            {submitting ? 'Submitting…' : 'Post update now'}
          </button>
          <button className="btn" onClick={() => setSelectedDestinations([])}>
            Clear destination filter
          </button>
        </div>
      </div>

      {message && <p className="security-note">{message}</p>}
    </section>
  );
}
