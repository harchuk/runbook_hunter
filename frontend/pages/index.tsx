import { useEffect, useState } from 'react';
import AuthPanel from '../components/AuthPanel';
import IncidentsTab from '../components/IncidentsTab';
import SettingsTab from '../components/SettingsTab';
import PostUpdateDialog from '../components/PostUpdateDialog';
import ThemeToggle from '../components/ThemeToggle';
import { fetchIncidents } from '../lib/api';

type Tab = 'incidents' | 'settings' | 'post';

export default function Home() {
  const [theme, setTheme] = useState<'light' | 'dark'>('dark');
  const [tab, setTab] = useState<Tab>('incidents');
  const [incidentCount, setIncidentCount] = useState(0);
  const [incidentOpenCount, setIncidentOpenCount] = useState(0);
  const [incidentCriticalCount, setIncidentCriticalCount] = useState(0);
  const [authVersion, setAuthVersion] = useState(0);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }
    const saved = window.localStorage.getItem('rh_theme');
    const next = saved === 'light' ? 'light' : 'dark';
    setTheme(next);
  }, []);

  useEffect(() => {
    if (typeof document !== 'undefined') {
      document.documentElement.setAttribute('data-theme', theme);
    }
    if (typeof window !== 'undefined') {
      window.localStorage.setItem('rh_theme', theme);
    }
  }, [theme]);

  useEffect(() => {
    fetchIncidents()
      .then((items) => {
        const rows = items as any[];
        const open = rows.filter((item) => (item.status ?? item.Status ?? 'open') === 'open').length;
        const critical = rows.filter((item) => (item.severity ?? item.Severity ?? '') === 'critical').length;
        setIncidentCount(rows.length);
        setIncidentOpenCount(open);
        setIncidentCriticalCount(critical);
      })
      .catch(() => {
        setIncidentCount(0);
        setIncidentOpenCount(0);
        setIncidentCriticalCount(0);
      });
  }, [tab, authVersion]);

  return (
    <main className="console-page">
      <header className="topbar">
        <div className="brand-wrap">
          <a className="brand" href="/">Runbook Hunter</a>
          <span className="hint">Admin Console</span>
        </div>
        <div className="topbar-controls">
          <ThemeToggle theme={theme} onToggle={() => setTheme((v) => (v === 'dark' ? 'light' : 'dark'))} />
        </div>
      </header>

      <section className="hero">
        <p className="eyebrow">Runbook Hunter Console</p>
        <h1>Operate incidents with signal, speed, and confidence.</h1>
        <p className="subtitle">
          Always-on correlation, read-only diagnostics, and deduplicated Telegram/Mattermost updates.
          Tune everything from config and safely override through UI.
        </p>
        <p className="security-note">
          Security notes: MVP runs read-only checks only. Write actions are disabled. Secrets in UI overrides are
          encrypted at rest via backend key from Kubernetes Secret.
        </p>
        <div className="stats">
          <article className="stat">
            <span className="hint">Total incidents</span>
            <strong>{incidentCount}</strong>
          </article>
          <article className="stat">
            <span className="hint">Open incidents</span>
            <strong>{incidentOpenCount}</strong>
          </article>
          <article className="stat">
            <span className="hint">Critical severity</span>
            <strong>{incidentCriticalCount}</strong>
          </article>
          <article className="stat">
            <span className="hint">Config precedence</span>
            <strong>UI &gt; ConfigMap &gt; Defaults</strong>
          </article>
        </div>
      </section>

      <AuthPanel onAuthChanged={() => setAuthVersion((value) => value + 1)} />

      <nav className="tabs" aria-label="Main tabs">
        <button className={`tab-btn ${tab === 'incidents' ? 'active' : ''}`} onClick={() => setTab('incidents')}>
          Incidents
        </button>
        <button className={`tab-btn ${tab === 'settings' ? 'active' : ''}`} onClick={() => setTab('settings')}>
          Settings
        </button>
        <button className={`tab-btn ${tab === 'post' ? 'active' : ''}`} onClick={() => setTab('post')}>
          Post Update Now
        </button>
      </nav>

      {tab === 'incidents' && <IncidentsTab key={`incidents-${authVersion}`} />}
      {tab === 'settings' && <SettingsTab key={`settings-${authVersion}`} />}
      {tab === 'post' && <PostUpdateDialog key={`post-${authVersion}`} />}
    </main>
  );
}
