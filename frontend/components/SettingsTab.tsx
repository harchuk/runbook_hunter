import { useEffect, useState } from 'react';
import { fetchEffectiveSettings, fetchOverrides, saveOverrides } from '../lib/api';
import ResetOverridesButton from './ResetOverridesButton';

const PRESETS: Record<string, any> = {
  'Demo Local Routing': {
    routing: {
      rules: [
        {
          name: 'local-telegram',
          routeKey: 'local-telegram',
          enabled: true,
          priority: 10,
          matchLabels: { env: 'local' },
          destinations: ['tg-local']
        }
      ]
    }
  },
  'Safe Limits': {
    limits: {
      stepTimeout: 8000000000,
      dedupCooldown: 180000000000,
      maxSteps: 3
    },
    security: {
      requestTimeout: 10000000000,
      retryCount: 2
    }
  },
  'JWT Auth Mode': {
    auth: {
      mode: 'jwt',
      jwtSecret: 'replace-in-secret-or-env'
    }
  }
};

function formatDurationNs(value: any): string {
  const n = Number(value);
  if (!Number.isFinite(n) || n <= 0) return 'n/a';
  const sec = Math.round(n / 1_000_000_000);
  if (sec < 60) return `${sec}s`;
  const min = Math.round(sec / 60);
  return `${min}m`;
}

export default function SettingsTab() {
  const [effective, setEffective] = useState<any>(null);
  const [overrides, setOverrides] = useState<any>({});
  const [editor, setEditor] = useState('{}');
  const [message, setMessage] = useState('');
  const [saving, setSaving] = useState(false);

  const refresh = async () => {
    const [eff, ov] = await Promise.all([fetchEffectiveSettings(), fetchOverrides()]);
    setEffective(eff);
    setOverrides(ov);
    setEditor(JSON.stringify(ov, null, 2));
  };

  useEffect(() => {
    refresh().catch((e) => setMessage(e.message));
  }, []);

  const onSave = async () => {
    setSaving(true);
    try {
      const parsed = JSON.parse(editor);
      await saveOverrides(parsed);
      setMessage('Overrides saved. Effective config updated.');
      await refresh();
    } catch (e: any) {
      setMessage(e.message);
    } finally {
      setSaving(false);
    }
  };

  const applyPreset = (name: string) => {
    setEditor(JSON.stringify(PRESETS[name], null, 2));
    setMessage(`Preset loaded: ${name}. Review values before Save.`);
  };

  const destinations = {
    tg: Number(effective?.destinations?.telegram?.length || 0),
    mm: Number(effective?.destinations?.mattermost?.length || 0)
  };

  return (
    <section className="panel">
      <h2 style={{ marginTop: 0 }}>Settings</h2>
      <p className="hint" style={{ marginTop: 0 }}>
        Precedence: <strong>UI overrides</strong> {'>'} <strong>ConfigMap/Secret defaults</strong> {'>'}{' '}
        <strong>built-in defaults</strong>
      </p>

      <div className="stats-inline">
        <span className="mini-pill">Runbook mode: {effective?.runbooks?.mode || 'n/a'}</span>
        <span className="mini-pill">Auth mode: {effective?.auth?.mode || 'n/a'}</span>
        <span className="mini-pill">Dedup cooldown: {formatDurationNs(effective?.limits?.dedupCooldown)}</span>
        <span className="mini-pill">Destinations: TG {destinations.tg} / MM {destinations.mm}</span>
      </div>

      {message && <p className="security-note">{message}</p>}

      <div className="grid-settings">
        <article className="panel panel-compact">
          <h3 style={{ marginTop: 0 }}>Current Overrides</h3>
          <p className="hint">Sensitive values are encrypted before DB write.</p>
          <pre className="code-block">{JSON.stringify(overrides, null, 2)}</pre>
          <ResetOverridesButton
            onReset={async () => {
              await refresh();
            }}
          />
        </article>

        <article className="panel panel-compact">
          <h3 style={{ marginTop: 0 }}>Edit Overrides (JSON)</h3>
          <div className="button-row">
            {Object.keys(PRESETS).map((name) => (
              <button key={name} className="btn" onClick={() => applyPreset(name)}>
                {name}
              </button>
            ))}
          </div>
          <textarea className="textarea" value={editor} onChange={(e) => setEditor(e.target.value)} />
          <div className="button-row" style={{ marginTop: 10 }}>
            <button className="btn primary" onClick={onSave} disabled={saving}>
              {saving ? 'Saving…' : 'Save Overrides'}
            </button>
          </div>
        </article>
      </div>

      <article className="panel panel-compact" style={{ marginTop: 12 }}>
        <h3 style={{ marginTop: 0 }}>Effective Settings</h3>
        <pre className="code-block">{JSON.stringify(effective, null, 2)}</pre>
      </article>
    </section>
  );
}
