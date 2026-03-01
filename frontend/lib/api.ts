const API_BASE = process.env.NEXT_PUBLIC_API_BASE || '';

export async function fetchIncidents() {
  const res = await fetch(`${API_BASE}/api/incidents`, { credentials: 'include' });
  if (!res.ok) throw new Error(`incidents request failed: ${res.status}`);
  return res.json();
}

export async function fetchIncident(id: number) {
  const res = await fetch(`${API_BASE}/api/incidents/${id}`, { credentials: 'include' });
  if (!res.ok) throw new Error(`incident request failed: ${res.status}`);
  return res.json();
}

export async function fetchEffectiveSettings() {
  const res = await fetch(`${API_BASE}/api/settings/effective`, { credentials: 'include' });
  if (!res.ok) throw new Error(`settings request failed: ${res.status}`);
  return res.json();
}

export async function fetchOverrides() {
  const res = await fetch(`${API_BASE}/api/settings/overrides`, { credentials: 'include' });
  if (!res.ok) throw new Error(`overrides request failed: ${res.status}`);
  return res.json();
}

export async function saveOverrides(payload: unknown) {
  const res = await fetch(`${API_BASE}/api/settings/overrides`, {
    method: 'PUT',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  if (!res.ok) throw new Error(`save overrides failed: ${res.status}`);
  return res.json();
}

export async function resetOverrides(scope: 'all' | 'keys', keys: string[] = []) {
  const res = await fetch(`${API_BASE}/api/settings/overrides/reset`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ scope, keys })
  });
  if (!res.ok) throw new Error(`reset overrides failed: ${res.status}`);
  return res.json();
}

export async function postUpdateNow(incidentId: number, destinationIds: string[]) {
  const res = await fetch(`${API_BASE}/api/incidents/${incidentId}/post-update`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ destinationIds })
  });
  if (!res.ok) throw new Error(`post update failed: ${res.status}`);
  return res.json();
}
