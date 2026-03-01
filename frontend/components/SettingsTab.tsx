import { useEffect, useState } from 'react';
import { fetchEffectiveSettings, fetchOverrides, saveOverrides } from '../lib/api';
import ResetOverridesButton from './ResetOverridesButton';

export default function SettingsTab() {
  const [effective, setEffective] = useState<any>(null);
  const [overrides, setOverrides] = useState<any>({});
  const [editor, setEditor] = useState('{}');
  const [message, setMessage] = useState('');

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
    try {
      const parsed = JSON.parse(editor);
      await saveOverrides(parsed);
      setMessage('Overrides saved. Effective config updated.');
      await refresh();
    } catch (e: any) {
      setMessage(e.message);
    }
  };

  return (
    <section className="panel">
      <h2 style={{ marginTop: 0 }}>Settings</h2>
      <p className="hint" style={{ marginTop: 0 }}>
        Precedence: <strong>UI overrides</strong> {'>'} <strong>ConfigMap/Secret defaults</strong> {'>'}{' '}
        <strong>built-in defaults</strong>
      </p>

      {message && <p className="security-note">{message}</p>}

      <div className="grid-settings">
        <article className="panel" style={{ padding: 12 }}>
          <h3 style={{ marginTop: 0 }}>Current Overrides</h3>
          <p className="hint">Sensitive values are encrypted before DB write.</p>
          <pre className="code-block">{JSON.stringify(overrides, null, 2)}</pre>
          <ResetOverridesButton
            onReset={async () => {
              await refresh();
            }}
          />
        </article>

        <article className="panel" style={{ padding: 12 }}>
          <h3 style={{ marginTop: 0 }}>Edit Overrides (JSON)</h3>
          <textarea className="textarea" value={editor} onChange={(e) => setEditor(e.target.value)} />
          <div className="button-row" style={{ marginTop: 10 }}>
            <button className="btn primary" onClick={onSave}>Save Overrides</button>
          </div>
        </article>
      </div>

      <article className="panel" style={{ padding: 12, marginTop: 12 }}>
        <h3 style={{ marginTop: 0 }}>Effective Settings</h3>
        <pre className="code-block">{JSON.stringify(effective, null, 2)}</pre>
      </article>
    </section>
  );
}
