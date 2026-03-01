import { useState } from 'react';
import { resetOverrides } from '../lib/api';

export default function ResetOverridesButton({ onReset }: { onReset: () => Promise<void> }) {
  const [keys, setKeys] = useState('');
  const [message, setMessage] = useState('');

  const resetAll = async () => {
    await resetOverrides('all');
    await onReset();
    setMessage('All overrides reset');
  };

  const resetKeys = async () => {
    const list = keys.split(',').map((v) => v.trim()).filter(Boolean);
    await resetOverrides('keys', list);
    await onReset();
    setMessage('Selected keys reset');
  };

  return (
    <div>
      <h4>Reset overrides</h4>
      <button onClick={resetAll}>Reset all</button>
      <div>
        <input
          placeholder="routing.rules,destinations.telegram"
          value={keys}
          onChange={(e) => setKeys(e.target.value)}
        />
        <button onClick={resetKeys}>Reset keys</button>
      </div>
      {message && <p>{message}</p>}
    </div>
  );
}
