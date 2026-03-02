import { useEffect, useState } from 'react';
import { fetchEffectiveSettings, fetchOverrides, resetOverrides, saveOverrides } from '../lib/api';

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
    refresh().catch((e: any) => setMessage(e.message));
  }, []);

  const onSave = async () => {
    setSaving(true);
    try {
      const parsed = JSON.parse(editor);
      await saveOverrides(parsed);
      await refresh();
      setMessage('Overrides saved. Precedence: UI overrides > ConfigMap defaults > built-in defaults.');
    } catch (e: any) {
      setMessage(e.message);
    } finally {
      setSaving(false);
    }
  };

  const onResetAll = async () => {
    try {
      await resetOverrides('all');
      await refresh();
      setMessage('All UI overrides were reset. Effective values now come from ConfigMap/defaults.');
    } catch (e: any) {
      setMessage(e.message);
    }
  };

  const onResetKeys = async () => {
    const raw = prompt('Comma separated keys to reset (e.g. routing.rules,destinations.telegram)');
    if (!raw) return;
    const keys = raw.split(',').map((v) => v.trim()).filter(Boolean);
    if (keys.length === 0) return;
    try {
      await resetOverrides('keys', keys);
      await refresh();
      setMessage('Selected keys were reset from UI overrides.');
    } catch (e: any) {
      setMessage(e.message);
    }
  };

  const strictMode = String(effective?.gitops?.mode || '').toLowerCase() === 'strict' && Boolean(effective?.gitops?.enabled);

  return (
    <section className="panel stack-16">
      <div className="section-head">
        <div>
          <h2 className="title">Settings</h2>
          <p className="hint">Configuration precedence and override management with explicit reset paths.</p>
        </div>
      </div>

      {strictMode && (
        <p className="security-note">
          GitOps strict mode enabled: direct UI override writes are deprecated and blocked. Use Changes tab to open change requests.
        </p>
      )}
      {message && <p className="security-note">{message}</p>}

      <div className="workspace-grid">
        <div className="card stack-10">
          <h3>UI Overrides</h3>
          <p className="hint">Security notes: sensitive values are encrypted at rest in DB.</p>
          <textarea className="textarea" value={editor} onChange={(e) => setEditor(e.target.value)} disabled={strictMode} />
          <div className="button-row">
            <button className="btn" disabled={saving || strictMode} onClick={onSave}>
              {saving ? 'Saving…' : 'Save overrides'}
            </button>
            <button className="btn" onClick={onResetAll} disabled={strictMode}>Reset all</button>
            <button className="btn" onClick={onResetKeys} disabled={strictMode}>Reset keys</button>
          </div>
        </div>

        <div className="card stack-10">
          <h3>Effective Settings</h3>
          <p className="hint">Applied runtime view.</p>
          <pre className="code-block">{JSON.stringify(effective, null, 2)}</pre>
        </div>
      </div>

      <details className="card">
        <summary>Raw overrides JSON</summary>
        <pre className="code-block">{JSON.stringify(overrides, null, 2)}</pre>
      </details>
    </section>
  );
}
