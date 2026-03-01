import { FormEvent, useState } from 'react';
import { clearAuth, setBasicAuth, setJWTAuth } from '../lib/api';

type Props = {
  onAuthChanged: () => void;
};

type Mode = 'basic' | 'jwt';

export default function AuthPanel({ onAuthChanged }: Props) {
  const [mode, setMode] = useState<Mode>('basic');
  const [user, setUser] = useState('admin');
  const [pass, setPass] = useState('change-me');
  const [token, setToken] = useState('');
  const [message, setMessage] = useState('API auth required. Set credentials to unlock incidents and settings.');

  const onSave = (event: FormEvent) => {
    event.preventDefault();
    if (mode === 'basic') {
      if (!user.trim() || !pass.trim()) {
        setMessage('Basic auth requires user and password.');
        return;
      }
      setBasicAuth(user.trim(), pass);
    } else {
      if (!token.trim()) {
        setMessage('JWT token is empty.');
        return;
      }
      setJWTAuth(token.trim());
    }
    setMessage('Auth saved in current browser session.');
    onAuthChanged();
  };

  const onClear = () => {
    clearAuth();
    setMessage('Auth cleared. API requests will return 401 until credentials are set again.');
    onAuthChanged();
  };

  return (
    <section className="panel auth-panel">
      <div className="section-head" style={{ marginBottom: 10 }}>
        <div>
          <h2 style={{ margin: 0 }}>API Access</h2>
          <p className="hint" style={{ margin: '6px 0 0' }}>
            Security notes: credentials are kept in `sessionStorage` only.
          </p>
        </div>
        <span className="mini-pill">Current mode: {mode}</span>
      </div>
      <form className="row" onSubmit={onSave}>
        <label className="label">
          Auth mode
          <select className="input" value={mode} onChange={(e) => setMode(e.target.value as Mode)}>
            <option value="basic">Basic Auth</option>
            <option value="jwt">JWT Bearer</option>
          </select>
        </label>

        {mode === 'basic' ? (
          <div className="grid-settings">
            <label className="label">
              Username
              <input className="input" value={user} onChange={(e) => setUser(e.target.value)} />
            </label>
            <label className="label">
              Password
              <input className="input" type="password" value={pass} onChange={(e) => setPass(e.target.value)} />
            </label>
          </div>
        ) : (
          <label className="label">
            JWT token
            <input className="input" value={token} onChange={(e) => setToken(e.target.value)} />
          </label>
        )}

        <div className="button-row">
          <button className="btn primary" type="submit">Save API credentials</button>
          <button className="btn danger" type="button" onClick={onClear}>Clear</button>
        </div>
      </form>
      {message && <p className="security-note">{message}</p>}
    </section>
  );
}
