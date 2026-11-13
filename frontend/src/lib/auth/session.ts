import { isRole, type Role } from './roles';

export type Session = { role: Role; exp: number };

function decodeBase64Url(segment: string): string {
  const base64 = segment.replace(/-/g, '+').replace(/_/g, '/');
  const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), '=');
  const binary = atob(padded);
  const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
  return new TextDecoder().decode(bytes);
}

function readPayload(token: string | undefined): { role: Role; exp: number } | null {
  if (!token) return null;
  const parts = token.split('.');
  if (parts.length !== 3) return null;

  try {
    const payload: unknown = JSON.parse(decodeBase64Url(parts[1]));
    if (typeof payload !== 'object' || payload === null) return null;
    const { role, exp } = payload as Record<string, unknown>;
    if (!isRole(role) || typeof exp !== 'number') return null;
    return { role, exp };
  } catch {
    return null;
  }
}

/**
 * Membaca `role` dan `exp` dari JWT TANPA memverifikasi tanda tangan.
 *
 * Hasilnya hanya dipakai untuk mengarahkan halaman. Siapa pun bisa memalsukan isi token,
 * jadi keputusan akses data tetap di backend, yang memverifikasi RS256 di setiap request.
 */
export function readSession(token: string | undefined, now = Date.now()): Session | null {
  const payload = readPayload(token);
  if (!payload || payload.exp * 1000 <= now) return null;
  return payload;
}

/** Peran dari token yang formatnya valid tetapi sudah kedaluwarsa, untuk layar Sesi Berakhir. */
export function readExpiredRole(token: string | undefined, now = Date.now()): Role | null {
  const payload = readPayload(token);
  return payload && payload.exp * 1000 <= now ? payload.role : null;
}
