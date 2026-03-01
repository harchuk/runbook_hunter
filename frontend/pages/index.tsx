import { useState } from 'react';
import IncidentsTab from '../components/IncidentsTab';
import SettingsTab from '../components/SettingsTab';
import PostUpdateDialog from '../components/PostUpdateDialog';

type Tab = 'incidents' | 'settings' | 'post';

export default function Home() {
  const [tab, setTab] = useState<Tab>('incidents');

  return (
    <main style={{ fontFamily: 'ui-sans-serif, system-ui', padding: 24 }}>
      <h1>Runbook Hunter Admin</h1>
      <p>Always-on hunter for Alertmanager incidents with deduplicated updates.</p>
      <p style={{ fontSize: 13 }}><b>Security notes:</b> write actions are disabled in MVP; only read-only checks and notifications are available.</p>
      <div style={{ display: 'flex', gap: 8, marginBottom: 16 }}>
        <button onClick={() => setTab('incidents')}>Incidents</button>
        <button onClick={() => setTab('settings')}>Settings</button>
        <button onClick={() => setTab('post')}>Post update now</button>
      </div>
      {tab === 'incidents' && <IncidentsTab />}
      {tab === 'settings' && <SettingsTab />}
      {tab === 'post' && <PostUpdateDialog />}
    </main>
  );
}
