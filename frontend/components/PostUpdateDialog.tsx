import { useState } from 'react';
import { postUpdateNow } from '../lib/api';

export default function PostUpdateDialog() {
  const [incidentId, setIncidentId] = useState('');
  const [destinationIds, setDestinationIds] = useState('');
  const [message, setMessage] = useState('');

  const onSubmit = async () => {
    try {
      const id = Number(incidentId);
      const dests = destinationIds.split(',').map((v) => v.trim()).filter(Boolean);
      await postUpdateNow(id, dests);
      setMessage('Update posted');
    } catch (e: any) {
      setMessage(e.message);
    }
  };

  return (
    <div>
      <h2>Post update now</h2>
      <label>
        Incident ID
        <input value={incidentId} onChange={(e) => setIncidentId(e.target.value)} />
      </label>
      <label>
        Destination IDs (comma-separated)
        <input value={destinationIds} onChange={(e) => setDestinationIds(e.target.value)} />
      </label>
      <button onClick={onSubmit}>Post update now</button>
      {message && <p>{message}</p>}
    </div>
  );
}
