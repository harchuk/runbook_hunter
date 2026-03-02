const API_BASE_ENV = (process.env.NEXT_PUBLIC_API_BASE || '').replace(/\/+$/, '');
const AUTH_MODE_KEY = 'rh_auth_mode';
const AUTH_BASIC_KEY = 'rh_auth_basic';
const AUTH_JWT_KEY = 'rh_auth_jwt';

type AuthMode = 'basic' | 'jwt';

function buildURL(path: string): string {
  const normalized = path.startsWith('/') ? path : `/${path}`;
  const base = resolveApiBase();
  return `${base}${normalized}`;
}

function resolveApiBase(): string {
  if (API_BASE_ENV) {
    return API_BASE_ENV;
  }
  return '';
}

function getAuthHeader(): string | undefined {
  if (typeof window === 'undefined') {
    return undefined;
  }
  const mode = (window.sessionStorage.getItem(AUTH_MODE_KEY) as AuthMode | null) || 'basic';
  if (mode === 'jwt') {
    const token = window.sessionStorage.getItem(AUTH_JWT_KEY);
    if (!token) {
      return undefined;
    }
    return `Bearer ${token}`;
  }
  const basic = window.sessionStorage.getItem(AUTH_BASIC_KEY);
  if (!basic) {
    return undefined;
  }
  return `Basic ${basic}`;
}

function withAuth(headers?: Record<string, string>): Record<string, string> {
  const next: Record<string, string> = { ...(headers || {}) };
  const auth = getAuthHeader();
  if (auth) {
    next.Authorization = auth;
  }
  return next;
}

async function request(path: string, init?: RequestInit): Promise<Response> {
  const headers = withAuth((init?.headers as Record<string, string> | undefined) || undefined);
  return fetch(buildURL(path), {
    credentials: 'include',
    ...init,
    headers
  });
}

async function parseJSON<T>(res: Response): Promise<T> {
  const text = await res.text();
  const payload = text ? JSON.parse(text) : {};
  if (!res.ok) {
    const message = payload?.error || payload?.message || `request failed: ${res.status}`;
    throw new Error(message);
  }
  return payload as T;
}

export function setBasicAuth(user: string, pass: string) {
  if (typeof window === 'undefined') {
    return;
  }
  const encoded = window.btoa(`${user}:${pass}`);
  window.sessionStorage.setItem(AUTH_MODE_KEY, 'basic');
  window.sessionStorage.setItem(AUTH_BASIC_KEY, encoded);
  window.sessionStorage.removeItem(AUTH_JWT_KEY);
}

export function setJWTAuth(token: string) {
  if (typeof window === 'undefined') {
    return;
  }
  window.sessionStorage.setItem(AUTH_MODE_KEY, 'jwt');
  window.sessionStorage.setItem(AUTH_JWT_KEY, token.trim());
  window.sessionStorage.removeItem(AUTH_BASIC_KEY);
}

export function clearAuth() {
  if (typeof window === 'undefined') {
    return;
  }
  window.sessionStorage.removeItem(AUTH_MODE_KEY);
  window.sessionStorage.removeItem(AUTH_BASIC_KEY);
  window.sessionStorage.removeItem(AUTH_JWT_KEY);
}

export async function fetchIncidents() {
  return parseJSON<any[]>(await request('/api/incidents'));
}

export async function fetchIncident(id: number) {
  return parseJSON<any>(await request(`/api/incidents/${id}`));
}

export async function fetchIncidentAlerts(id: number) {
  return parseJSON<any[]>(await request(`/api/incidents/${id}/alerts`));
}

export async function fetchIncidentRunbookExecutions(id: number) {
  return parseJSON<any[]>(await request(`/api/incidents/${id}/runbook-executions`));
}

export async function fetchIncidentClosureCriteria(id: number) {
  return parseJSON<any>(await request(`/api/incidents/${id}/closure-criteria`));
}

export async function fetchIncidentEvents(id: number) {
  return parseJSON<any[]>(await request(`/api/incidents/${id}/events`));
}

export async function closeIncident(id: number, reason: string) {
  return parseJSON<any>(
    await request(`/api/incidents/${id}/close`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason })
    })
  );
}

export async function reopenIncident(id: number, reason: string) {
  return parseJSON<any>(
    await request(`/api/incidents/${id}/reopen`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason })
    })
  );
}

export async function rerunIncidentRunbook(id: number) {
  return parseJSON<any>(
    await request(`/api/incidents/${id}/runbook/rerun`, {
      method: 'POST'
    })
  );
}

export async function fetchAlerts() {
  return parseJSON<any[]>(await request('/api/alerts'));
}

export async function fetchAlert(id: number) {
  return parseJSON<any>(await request(`/api/alerts/${id}`));
}

export async function fetchApprovals() {
  return parseJSON<any[]>(await request('/api/approvals'));
}

export async function approveRequest(id: number) {
  return parseJSON<any>(
    await request(`/api/approvals/${id}/approve`, {
      method: 'POST'
    })
  );
}

export async function rejectRequest(id: number, reason: string) {
  return parseJSON<any>(
    await request(`/api/approvals/${id}/reject`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason })
    })
  );
}

export async function fetchGitOpsChanges() {
  return parseJSON<any[]>(await request('/api/gitops/changes'));
}

export async function fetchGitOpsChange(id: number) {
  return parseJSON<any>(await request(`/api/gitops/changes/${id}`));
}

export async function createGitOpsChange(payload: { change_type: string; title: string; desired: any }) {
  return parseJSON<any>(
    await request('/api/gitops/changes', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    })
  );
}

export async function fetchEffectiveSettings() {
  return parseJSON<any>(await request('/api/settings/effective'));
}

export async function fetchOverrides() {
  return parseJSON<any>(await request('/api/settings/overrides'));
}

export async function saveOverrides(payload: unknown) {
  return parseJSON<any>(
    await request('/api/settings/overrides', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    })
  );
}

export async function resetOverrides(scope: 'all' | 'keys', keys: string[] = []) {
  return parseJSON<any>(
    await request('/api/settings/overrides/reset', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ scope, keys })
    })
  );
}

export async function postUpdateNow(incidentId: number, destinationIds: string[]) {
  return parseJSON<any>(
    await request(`/api/incidents/${incidentId}/post-update`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ destinationIds })
    })
  );
}

export async function generateDemoIncidents() {
  const suffix = Date.now();
  const now = new Date().toISOString();
  const templates = [
    {
      alertname: 'DemoAPI5xxSpike',
      severity: 'critical',
      instance: `demo-api-${suffix}`,
      summary: 'Demo API 5xx above threshold'
    },
    {
      alertname: 'DemoLatencyP95Spike',
      severity: 'warning',
      instance: `demo-latency-${suffix}`,
      summary: 'Demo API latency p95 above threshold'
    },
    {
      alertname: 'DemoDBPoolExhausted',
      severity: 'critical',
      instance: `demo-db-${suffix}`,
      summary: 'Demo DB pool exhausted'
    }
  ];

  for (const item of templates) {
    const payload = {
      version: '4',
      groupKey: `{}:{alertname="${item.alertname}"}`,
      status: 'firing',
      receiver: 'runbook-hunter',
      groupLabels: { alertname: item.alertname },
      commonLabels: {
        alertname: item.alertname,
        service: 'api',
        env: 'local',
        severity: item.severity
      },
      commonAnnotations: { summary: item.summary },
      alerts: [
        {
          status: 'firing',
          labels: {
            alertname: item.alertname,
            service: 'api',
            env: 'local',
            severity: item.severity,
            instance: item.instance
          },
          annotations: { summary: item.summary },
          startsAt: now,
          generatorURL: `http://prometheus.local/graph?g0.expr=${encodeURIComponent(item.alertname)}`
        }
      ]
    };
    const res = await request('/api/alertmanager', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    if (!res.ok) {
      throw new Error(`demo ingest failed for ${item.alertname}: ${res.status}`);
    }
  }
}
