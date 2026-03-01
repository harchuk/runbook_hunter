import { useState } from 'react';
import { postUpdateNow } from '../lib/api';

export default function PostUpdateDialog() {
  const [incidentId, setIncidentId] = useState('');
  const [destinationIds, setDestinationIds] = useState('');
  const [message, setMessage] = useState('');

  const onSubmit = async () => {
    try {
      const id = Number(incidentId);
      if (!Number.isFinite(id) || id <= 0) {
        setMessage('Incident ID must be a positive number');
        return;
      }
      const dests = destinationIds.split(',').map((v) => v.trim()).filter(Boolean);
      await postUpdateNow(id, dests);
      setMessage('Update triggered. Worker dedup and routing rules still apply.');
    } catch (e: any) {
      setMessage(e.message);
    }
  };

  return (
    <section className="panel">
      <h2 style={{ marginTop: 0 }}>Post update now</h2>
      <p className="hint">Force a notification for an incident. Cooldown is bypassed, dedup state is updated.</p>
      <div className="row" style={{ maxWidth: 560 }}>
        <label className="label">
          Incident ID
          <input className="input" value={incidentId} onChange={(e) => setIncidentId(e.target.value)} />
        </label>
        <label className="label">
          Destination IDs (comma-separated, optional)
          <input
            className="input"
            placeholder="tg-default,mm-primary"
            value={destinationIds}
            onChange={(e) => setDestinationIds(e.target.value)}
          />
        </label>
        <div className="button-row">
          <button className="btn primary" onClick={onSubmit}>Post update now</button>
        </div>
      </div>
      {message && <p className="security-note">{message}</p>}
    </section>
  );
}
