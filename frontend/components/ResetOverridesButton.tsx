import { useState } from 'react';
import { resetOverrides } from '../lib/api';

export default function ResetOverridesButton({ onReset }: { onReset: () => Promise<void> }) {
  const [keys, setKeys] = useState('');
  const [message, setMessage] = useState('');

  const resetAll = async () => {
    await resetOverrides('all');
    await onReset();
    setMessage('All overrides reset to ConfigMap/defaults.');
  };

  const resetKeys = async () => {
    const list = keys.split(',').map((v) => v.trim()).filter(Boolean);
    await resetOverrides('keys', list);
    await onReset();
    setMessage('Selected override keys reset.');
  };

  return (
    <div className="row" style={{ marginTop: 10 }}>
      <h4 style={{ margin: 0 }}>Reset overrides</h4>
      <div className="button-row">
        <button className="btn danger" onClick={resetAll}>Reset all</button>
      </div>
      <label className="label">
        Reset specific keys
        <input
          className="input"
          placeholder="routing.rules,destinations.telegram"
          value={keys}
          onChange={(e) => setKeys(e.target.value)}
        />
      </label>
      <div className="button-row">
        <button className="btn" onClick={resetKeys}>Reset selected keys</button>
      </div>
      {message && <p className="hint">{message}</p>}
    </div>
  );
}
