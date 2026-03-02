import { FormEvent, useEffect, useState } from 'react';
import { createGitOpsChange, fetchGitOpsChanges } from '../lib/api';

export default function ChangesTab() {
  const [changes, setChanges] = useState<any[]>([]);
  const [title, setTitle] = useState('');
  const [kind, setKind] = useState('settings');
  const [desired, setDesired] = useState('{\n  "routing": {\n    "rules": []\n  }\n}');
  const [message, setMessage] = useState('');

  const load = async () => {
    const payload = await fetchGitOpsChanges();
    setChanges(payload || []);
  };

  useEffect(() => {
    load().catch((e: any) => setMessage(e.message));
  }, []);

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();
    try {
      const parsed = JSON.parse(desired);
      await createGitOpsChange({
        change_type: kind,
        title: title.trim() || 'Runbook Hunter change request',
        desired: parsed
      });
      setMessage('Change request created. Track PR status in list below.');
      setTitle('');
      await load();
    } catch (e: any) {
      setMessage(e.message);
    }
  };

  return (
    <section className="panel stack-16">
      <div className="section-head">
        <div>
          <h2 className="title">Changes (GitOps)</h2>
          <p className="hint">Create change requests instead of mutating runtime settings directly in strict mode.</p>
        </div>
        <button className="btn" onClick={() => load().catch((e: any) => setMessage(e.message))}>Refresh</button>
      </div>

      <form className="card stack-10" onSubmit={onSubmit}>
        <h3>Create Change Request</h3>
        <div className="toolbar">
          <select className="input" value={kind} onChange={(e) => setKind(e.target.value)}>
            <option value="settings">settings</option>
            <option value="runbook">runbook</option>
            <option value="routing">routing</option>
          </select>
          <input className="input" placeholder="Title" value={title} onChange={(e) => setTitle(e.target.value)} />
        </div>
        <textarea className="textarea" value={desired} onChange={(e) => setDesired(e.target.value)} />
        <div className="button-row">
          <button className="btn" type="submit">Create</button>
        </div>
      </form>

      {message && <p className="security-note">{message}</p>}

      <div className="table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Type</th>
              <th>Title</th>
              <th>Status</th>
              <th>PR</th>
              <th>Drift</th>
            </tr>
          </thead>
          <tbody>
            {changes.map((row) => (
              <tr key={row.id ?? row.ID}>
                <td>{row.id ?? row.ID}</td>
                <td>{row.changeType ?? row.ChangeType}</td>
                <td>{row.title ?? row.Title}</td>
                <td><span className={`badge state-${String(row.status ?? row.Status ?? '').replace(/_/g, '-')}`}>{row.status ?? row.Status}</span></td>
                <td>
                  {(row.prUrl ?? row.PRURL) ? (
                    <a href={row.prUrl ?? row.PRURL} target="_blank" rel="noreferrer">Open PR</a>
                  ) : (
                    '-'
                  )}
                </td>
                <td>{(row.driftStatus ?? row.DriftStatus) || '-'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
