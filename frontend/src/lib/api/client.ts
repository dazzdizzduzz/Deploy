import { sessionExpiredUrl } from '@/lib/auth/constants';
import { isProblem, type Problem } from './problem';

/** Galat dari API dalam bentuk RFC 7807. `status` 0 berarti jaringan gagal. */
export class ApiProblem extends Error {
  constructor(readonly problem: Problem & { errors?: Record<string, unknown> }) {
    super(problem.detail ?? problem.title);
    this.name = 'ApiProblem';
  }

  get status() {
    return this.problem.status;
  }

  /** Pesan galat per field dari 422, misalnya `{ email: "Format email tidak valid" }`. */
  get fieldErrors(): Record<string, string> {
    const entries = Object.entries(this.problem.errors ?? {});
    return Object.fromEntries(
      entries.map(([key, value]) => [key, Array.isArray(value) ? String(value[0]) : String(value)]),
    );
  }
}

type Query = Record<string, string | number | undefined | null>;

type RequestOptions = { method?: 'GET' | 'POST' | 'PATCH' | 'DELETE'; body?: unknown; query?: Query };

/** Memanggil backend lewat `/api/proxy`, yang menempelkan token dari cookie httpOnly. */
export async function apiRequest<T>(path: string, { method = 'GET', body, query }: RequestOptions = {}) {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query ?? {})) {
    if (value !== undefined && value !== null && value !== '') params.set(key, String(value));
  }
  const search = params.size ? `?${params}` : '';

  let response: Response;
  try {
    response = await fetch(`/api/proxy/${path}${search}`, {
      method,
      headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
      credentials: 'same-origin',
      cache: 'no-store',
    });
  } catch {
    throw new ApiProblem({
      type: 'about:blank',
      title: 'Tidak dapat terhubung',
      status: 0,
      detail: 'Periksa koneksi internet lalu coba lagi.',
    });
  }

  const data: unknown = response.status === 204 ? null : await response.json().catch(() => null);
  if (response.ok) return data as T;

  if (response.status === 401) {
    // Cookie sudah dihapus proxy; tampilkan Sesi Berakhir (W24/KW07).
    const employee = window.location.pathname.startsWith('/karyawan');
    window.location.assign(sessionExpiredUrl({ employee, reason: 'expired' }));
  }
  throw new ApiProblem(
    isProblem(data)
      ? data
      : { type: 'about:blank', title: 'Permintaan gagal', status: response.status },
  );
}
