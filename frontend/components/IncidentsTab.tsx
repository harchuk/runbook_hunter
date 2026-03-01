import { useEffect, useState } from 'react';
import { fetchIncident, fetchIncidents } from '../lib/api';

interface Incident {
  id: number;
  alertName: string;
  service: string;
  env: string;
  status: string;
  brief: string;
  runbookName: string;
}

export default function IncidentsTab() {
  const [items, setItems] = useState<Incident[]>([]);
  const [selected, setSelected] = useState<any>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    fetchIncidents().then(setItems).catch((e) => setError(e.message));
  }, []);

  return (
    <div>
      <h2>Incidents</h2>
      {error && <p style={{ color: 'crimson' }}>{error}</p>}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
        <ul>
          {items.map((it) => (
            <li key={it.id}>
              <button
                onClick={() => fetchIncident(it.id).then(setSelected).catch((e) => setError(e.message))}
                style={{ width: '100%', textAlign: 'left' }}
              >
                #{it.id} [{it.status}] {it.alertName} ({it.service}/{it.env})
              </button>
            </li>
          ))}
        </ul>
        <pre style={{ background: '#111', color: '#e8e8e8', padding: 12, borderRadius: 8, overflowX: 'auto' }}>
          {selected ? JSON.stringify(selected, null, 2) : 'Select incident'}
        </pre>
      </div>
    </div>
  );
}
