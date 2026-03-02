import { FormEvent, useEffect, useState } from 'react';
import { AuthMode, clearAuth, getStoredAuthMode, setBasicAuth, setJWTAuth, setOIDCAuth } from '../lib/api';

type Props = {
  onAuthChanged?: () => void;
};

export default function AuthControls({ onAuthChanged }: Props) {
  const [mode, setMode] = useState<AuthMode>('basic');
  const [user, setUser] = useState('admin');
  const [pass, setPass] = useState('change-me');
  const [token, setToken] = useState('');
  const [message, setMessage] = useState('');

  useEffect(() => {
    if (typeof window === 'undefined') return;
    setMode(getStoredAuthMode());
  }, []);

  const onLogin = (event: FormEvent) => {
    event.preventDefault();
    if (mode === 'basic') {
      if (!user.trim() || !pass.trim()) {
        setMessage('Basic login requires username and password.');
        return;
      }
      setBasicAuth(user.trim(), pass);
    } else if (mode === 'oidc') {
      if (!token.trim()) {
        setMessage('OIDC login requires bearer token.');
        return;
      }
      setOIDCAuth(token.trim());
    } else {
      if (!token.trim()) {
        setMessage('JWT login requires token.');
        return;
      }
      setJWTAuth(token.trim());
    }
    setMessage(`Logged in using ${mode}.`);
    onAuthChanged?.();
  };

  const onLogout = () => {
    clearAuth();
    setMessage('Logged out.');
    onAuthChanged?.();
  };

  return (
    <form className="auth-inline" onSubmit={onLogin}>
      <select className="input auth-mode" value={mode} onChange={(e) => setMode(e.target.value as AuthMode)}>
        <option value="basic">Basic</option>
        <option value="oidc">OIDC</option>
        <option value="jwt">JWT</option>
      </select>

      {mode === 'basic' ? (
        <>
          <input className="input auth-field" placeholder="user" value={user} onChange={(e) => setUser(e.target.value)} />
          <input className="input auth-field" placeholder="password" type="password" value={pass} onChange={(e) => setPass(e.target.value)} />
        </>
      ) : (
        <input
          className="input auth-token"
          placeholder={mode === 'oidc' ? 'OIDC bearer token' : 'JWT token'}
          value={token}
          onChange={(e) => setToken(e.target.value)}
        />
      )}

      <button className="btn" type="submit">Login</button>
      <button className="btn" type="button" onClick={onLogout}>Logout</button>
      {message && <span className="hint auth-msg">{message}</span>}
    </form>
  );
}
