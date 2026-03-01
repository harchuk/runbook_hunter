import { useEffect, useState } from 'react';
import { fetchEffectiveSettings, fetchOverrides, saveOverrides } from '../lib/api';
import ResetOverridesButton from './ResetOverridesButton';

export default function SettingsTab() {
  const [effective, setEffective] = useState<any>(null);
  const [overrides, setOverrides] = useState<any>({});
  const [editor, setEditor] = useState('{}');
  const [message, setMessage] = useState('');

  useEffect(() => {
    Promise.all([fetchEffectiveSettings(), fetchOverrides()]).then(([eff, ov]) => {
      setEffective(eff);
      setOverrides(ov);
      setEditor(JSON.stringify(ov, null, 2));
    });
  }, []);

  const onSave = async () => {
    try {
      const parsed = JSON.parse(editor);
      await saveOverrides(parsed);
      setMessage('Overrides saved');
    } catch (e: any) {
      setMessage(e.message);
    }
  };

  return (
    <div>
      <h2>Settings</h2>
      <p style={{ marginBottom: 8 }}>Priority: UI overrides &gt; ConfigMap/Secret defaults &gt; built-in defaults.</p>
      <p style={{ fontSize: 13 }}><b>Security notes:</b> secrets in UI overrides are encrypted at rest via backend key from Kubernetes Secret.</p>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
        <div>
          <h3>Current Overrides</h3>
          <pre style={{ background: '#111', color: '#e8e8e8', padding: 12, borderRadius: 8, overflowX: 'auto' }}>
            {JSON.stringify(overrides, null, 2)}
          </pre>
          <ResetOverridesButton onReset={async () => {
            const ov = await fetchOverrides();
            setOverrides(ov);
            setEditor(JSON.stringify(ov, null, 2));
          }} />
        </div>
        <div>
          <h3>Edit Overrides (JSON)</h3>
          <textarea
            value={editor}
            onChange={(e) => setEditor(e.target.value)}
            style={{ width: '100%', minHeight: 280, fontFamily: 'monospace' }}
          />
          <button onClick={onSave}>Save Overrides</button>
          {message && <p>{message}</p>}
          <h3>Effective Settings</h3>
          <pre style={{ background: '#111', color: '#e8e8e8', padding: 12, borderRadius: 8, overflowX: 'auto' }}>
            {JSON.stringify(effective, null, 2)}
          </pre>
        </div>
      </div>
    </div>
  );
}
