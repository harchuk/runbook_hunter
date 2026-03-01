import type { NextApiRequest, NextApiResponse } from 'next';

export const config = {
  api: {
    bodyParser: false,
    externalResolver: true
  }
};

const HOP_BY_HOP_HEADERS = new Set([
  'connection',
  'keep-alive',
  'proxy-authenticate',
  'proxy-authorization',
  'te',
  'trailer',
  'transfer-encoding',
  'upgrade',
  'host',
  'content-length'
]);

function targetBase(): string {
  return (process.env.API_INTERNAL_BASE || 'http://localhost:8080').replace(/\/+$/, '');
}

function extractPath(input: string[] | string | undefined): string {
  if (!input) return '';
  if (Array.isArray(input)) return input.join('/');
  return input;
}

function buildTargetURL(req: NextApiRequest): string {
  const base = targetBase();
  const path = extractPath(req.query.path);
  const url = new URL(`${base}/api/${path}`);

  for (const [key, value] of Object.entries(req.query)) {
    if (key === 'path') continue;
    if (Array.isArray(value)) {
      for (const item of value) {
        url.searchParams.append(key, item);
      }
      continue;
    }
    if (typeof value === 'string') {
      url.searchParams.set(key, value);
    }
  }
  return url.toString();
}

function copyRequestHeaders(req: NextApiRequest): Headers {
  const headers = new Headers();
  for (const [key, value] of Object.entries(req.headers)) {
    const lowerKey = key.toLowerCase();
    if (HOP_BY_HOP_HEADERS.has(lowerKey) || typeof value === 'undefined') {
      continue;
    }
    if (Array.isArray(value)) {
      headers.set(key, value.join(','));
      continue;
    }
    headers.set(key, value);
  }
  return headers;
}

async function readRawBody(req: NextApiRequest): Promise<Buffer> {
  const chunks: Buffer[] = [];
  for await (const chunk of req) {
    chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
  }
  return Buffer.concat(chunks);
}

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  try {
    const url = buildTargetURL(req);
    const headers = copyRequestHeaders(req);

    let body: Buffer | undefined;
    if (req.method !== 'GET' && req.method !== 'HEAD') {
      body = await readRawBody(req);
    }

    const upstream = await fetch(url, {
      method: req.method,
      headers,
      body
    });

    for (const [key, value] of upstream.headers.entries()) {
      if (HOP_BY_HOP_HEADERS.has(key.toLowerCase())) {
        continue;
      }
      res.setHeader(key, value);
    }
    const payload = Buffer.from(await upstream.arrayBuffer());
    res.status(upstream.status).send(payload);
  } catch (err: any) {
    res.status(502).json({ error: `api proxy failed: ${err?.message || 'unknown error'}` });
  }
}
