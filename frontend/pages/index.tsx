import { useEffect, useState } from 'react';
import AuthPanel from '../components/AuthPanel';
import IncidentsTab from '../components/IncidentsTab';
import SettingsTab from '../components/SettingsTab';
import PostUpdateDialog from '../components/PostUpdateDialog';
import { fetchIncidents } from '../lib/api';

type Tab = 'incidents' | 'settings' | 'post';

export default function Home() {
  const [tab, setTab] = useState<Tab>('incidents');
  const [incidentCount, setIncidentCount] = useState<number>(0);
  const [authVersion, setAuthVersion] = useState(0);

  useEffect(() => {
    fetchIncidents().then((items) => setIncidentCount(items.length)).catch(() => setIncidentCount(0));
  }, [tab, authVersion]);

  return (
    <main>
      <section className="hero">
        <p className="eyebrow">Runbook Hunter Console</p>
        <h1>Incident hunting, but with signal over noise.</h1>
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
            <span className="hint">Open + historical incidents</span>
            <strong>{incidentCount}</strong>
          </article>
          <article className="stat">
            <span className="hint">Config precedence</span>
            <strong>UI &gt; ConfigMap &gt; Defaults</strong>
          </article>
          <article className="stat">
            <span className="hint">Worker mode</span>
            <strong>Always-on loop</strong>
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
