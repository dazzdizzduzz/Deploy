import { NextResponse, type NextRequest } from 'next/server';
import { PROBLEM_CONTENT_TYPE, type Problem } from '@/lib/api/problem';

/** Alamat backend Go. Dibaca di server saja, tidak pernah sampai ke browser. */
export const API_URL = (process.env.API_URL ?? 'http://localhost:8080').replace(/\/$/, '');

const UPSTREAM_TIMEOUT_MS = 10_000;

export const NO_STORE = { 'Cache-Control': 'no-store' };

export function problem(status: number, title: string, detail?: string) {
  const body: Problem = { type: 'about:blank', title, status, ...(detail && { detail }) };
  return NextResponse.json(body, {
    status,
    headers: { ...NO_STORE, 'Content-Type': PROBLEM_CONTENT_TYPE },
  });
}

/**
 * Backend saat ini membalas `{ "message": "..." }`, bukan problem+json. Fungsi ini menerima
 * kedua bentuk dan selalu membalas problem+json ke browser.
 */
export async function problemFromUpstream(upstream: Response, titles: Record<number, string>) {
  let title: string | undefined;
  let detail: string | undefined;
  try {
    const data: unknown = await upstream.json();
    if (typeof data === 'object' && data !== null) {
      const record = data as Record<string, unknown>;
      if (typeof record.title === 'string') title = record.title;
      detail = [record.detail, record.message].find(
        (value): value is string => typeof value === 'string',
      );
    }
  } catch {
    // Badan bukan JSON: pakai judul bawaan.
  }

  if (upstream.status >= 500) {
    // Jangan teruskan pesan internal server.
    return problem(502, 'Server bermasalah', 'Terjadi kesalahan pada server. Coba lagi beberapa saat lagi.');
  }
  return problem(upstream.status, title ?? titles[upstream.status] ?? 'Permintaan gagal', detail);
}

/** Baca badan JSON dari browser. Hanya `application/json`, agar `<form>` lintas situs ditolak. */
export async function readJsonBody(
  request: NextRequest,
): Promise<{ body: Record<string, unknown> } | { error: NextResponse }> {
  if (!request.headers.get('content-type')?.startsWith('application/json')) {
    return { error: problem(415, 'Format tidak didukung', 'Kirim data dalam format JSON.') };
  }
  try {
    const body: unknown = await request.json();
    if (typeof body === 'object' && body !== null && !Array.isArray(body)) {
      return { body: body as Record<string, unknown> };
    }
  } catch {
    // jatuh ke galat di bawah
  }
  return { error: problem(400, 'Permintaan tidak valid', 'Isi permintaan bukan JSON yang valid.') };
}

/** POST JSON ke backend. Mengembalikan `Response` upstream, atau `null` bila tidak terjangkau. */
export async function postUpstream(path: string, payload: unknown): Promise<Response | null> {
  try {
    return await fetch(`${API_URL}${path}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify(payload),
      cache: 'no-store',
      signal: AbortSignal.timeout(UPSTREAM_TIMEOUT_MS),
    });
  } catch {
    return null;
  }
}

export const unreachable = () =>
  problem(503, 'Server tidak dapat dihubungi', 'Periksa koneksi lalu coba lagi.');
