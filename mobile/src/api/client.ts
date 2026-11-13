import { API_BASE_URL, API_TIMEOUT_MS } from './config';

/**
 * Error dari API atau dari jaringan (`status` 0). Backend saat ini membalas `{ "message": "..." }`;
 * bentuk RFC 7807 (`title`/`detail`) juga diterima.
 */
export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly detail?: unknown,
  ) {
    super(message);
    this.name = 'ApiError';
  }

  /** Kode error RFC 7807 dari backend, misalnya `OUTSIDE_GEOFENCE`. */
  get code(): string | undefined {
    const value = (this.detail as { code?: unknown } | undefined)?.code;
    return typeof value === 'string' ? value : undefined;
  }

  /** Data tambahan per error (`errors` di problem+json), misalnya jarak ke kantor. */
  get extra(): Record<string, unknown> | undefined {
    const value = (this.detail as { errors?: unknown } | undefined)?.errors;
    return value && typeof value === 'object' ? (value as Record<string, unknown>) : undefined;
  }
}

let accessToken: string | null = null;
let onUnauthorized: (() => void) | null = null;

/** Dipakai modul auth untuk menyimpan token akses. */
export function setAccessToken(token: string | null) {
  accessToken = token;
}

/** Dipanggil bila server menolak token (401) pada permintaan yang membawa token. */
export function setUnauthorizedHandler(handler: (() => void) | null) {
  onUnauthorized = handler;
}

type RequestOptions = {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  body?: unknown;
  signal?: AbortSignal;
  /** Batas waktu khusus, misalnya lebih lama untuk unggah foto. */
  timeoutMs?: number;
};

export async function request<T>(
  path: string,
  { method = 'GET', body, signal, timeoutMs = API_TIMEOUT_MS }: RequestOptions = {},
): Promise<T> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), timeoutMs);
  // FormData (multipart): Content-Type beserta boundary diisi otomatis oleh fetch.
  const isForm = typeof FormData !== 'undefined' && body instanceof FormData;
  signal?.addEventListener('abort', () => controller.abort());

  try {
    const response = await fetch(`${API_BASE_URL}${path}`, {
      method,
      headers: {
        Accept: 'application/json',
        ...(body !== undefined && !isForm && { 'Content-Type': 'application/json' }),
        ...(accessToken && { Authorization: `Bearer ${accessToken}` }),
      },
      body: body === undefined ? undefined : isForm ? (body as FormData) : JSON.stringify(body),
      signal: controller.signal,
    });

    const text = await response.text();
    let data: any;
    try {
      data = text ? JSON.parse(text) : undefined;
    } catch {
      data = undefined;
    }

    if (!response.ok) {
      // Token kedaluwarsa atau dicabut: akhiri sesi agar kembali ke layar login.
      if (response.status === 401 && accessToken) onUnauthorized?.();
      const title =
        data?.detail ?? data?.message ?? data?.title ?? data?.error ?? `Permintaan gagal (${response.status})`;
      throw new ApiError(title, response.status, data);
    }
    return data as T;
  } catch (error) {
    if (error instanceof ApiError) throw error;
    if (controller.signal.aborted) throw new ApiError('Server tidak merespons. Coba lagi.', 0);
    throw new ApiError('Tidak dapat terhubung ke server. Periksa jaringan dan alamat API.', 0, error);
  } finally {
    clearTimeout(timeout);
  }
}

export const api = {
  get: <T>(path: string, signal?: AbortSignal) => request<T>(path, { signal }),
  post: <T>(path: string, body?: unknown) => request<T>(path, { method: 'POST', body }),
  /** Unggah multipart (misalnya selfie); batas waktu 30 detik. */
  postForm: <T>(path: string, form: FormData) => request<T>(path, { method: 'POST', body: form, timeoutMs: 30_000 }),
  put: <T>(path: string, body?: unknown) => request<T>(path, { method: 'PUT', body }),
  patch: <T>(path: string, body?: unknown) => request<T>(path, { method: 'PATCH', body }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
};
