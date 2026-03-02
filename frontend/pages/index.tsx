import { useEffect, useState } from 'react';
import AlertsTab from '../components/AlertsTab';
import AuthPanel from '../components/AuthPanel';
import ChangesTab from '../components/ChangesTab';
import IncidentsTab from '../components/IncidentsTab';
import OverviewTab from '../components/OverviewTab';
import RunbooksTab from '../components/RunbooksTab';
import SettingsTab from '../components/SettingsTab';
import ThemeToggle from '../components/ThemeToggle';

type Tab = 'overview' | 'incidents' | 'alerts' | 'runbooks' | 'changes' | 'settings';

export default function Home() {
  const [theme, setTheme] = useState<'light' | 'dark'>('dark');
  const [tab, setTab] = useState<Tab>('overview');
  const [authVersion, setAuthVersion] = useState(0);

  useEffect(() => {
    if (typeof window === 'undefined') return;
    const saved = window.localStorage.getItem('rh_theme');
    setTheme(saved === 'light' ? 'light' : 'dark');
  }, []);

  useEffect(() => {
    if (typeof document !== 'undefined') {
      document.documentElement.setAttribute('data-theme', theme);
    }
    if (typeof window !== 'undefined') {
      window.localStorage.setItem('rh_theme', theme);
    }
  }, [theme]);

  const tabs: Array<{ key: Tab; label: string }> = [
    { key: 'overview', label: 'Overview' },
    { key: 'incidents', label: 'Incidents' },
    { key: 'alerts', label: 'Alerts' },
    { key: 'runbooks', label: 'Runbooks' },
    { key: 'changes', label: 'Changes (GitOps)' },
    { key: 'settings', label: 'Settings' }
  ];

  return (
    <main className="app-shell">
      <header className="app-topbar">
        <div className="brand-wrap">
          <a className="brand" href="/">Runbook Hunter</a>
          <span className="hint">Console</span>
        </div>
        <ThemeToggle theme={theme} onToggle={() => setTheme((v) => (v === 'dark' ? 'light' : 'dark'))} />
      </header>

      <section className="compact-banner">
        <div>
          <h1>Runbook Hunter Console</h1>
          <p className="hint">
            Alerts and incidents are separated. One active runbook execution lifecycle per incident. GitOps-first config flow.
          </p>
        </div>
        <div className="banner-note">
          Security notes: read-only diagnostics by default, secrets from Kubernetes Secret, write-actions only via approval policy.
        </div>
      </section>

      <AuthPanel onAuthChanged={() => setAuthVersion((value) => value + 1)} />

      <nav className="tabs" aria-label="Main tabs">
        {tabs.map((item) => (
          <button key={item.key} className={`tab-btn ${tab === item.key ? 'active' : ''}`} onClick={() => setTab(item.key)}>
            {item.label}
          </button>
        ))}
      </nav>

      {tab === 'overview' && <OverviewTab key={`overview-${authVersion}`} />}
      {tab === 'incidents' && <IncidentsTab key={`incidents-${authVersion}`} />}
      {tab === 'alerts' && <AlertsTab key={`alerts-${authVersion}`} />}
      {tab === 'runbooks' && <RunbooksTab key={`runbooks-${authVersion}`} />}
      {tab === 'changes' && <ChangesTab key={`changes-${authVersion}`} />}
      {tab === 'settings' && <SettingsTab key={`settings-${authVersion}`} />}
    </main>
  );
}
