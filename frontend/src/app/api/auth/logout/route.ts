import { NextResponse, type NextRequest } from 'next/server';
import { API_URL, NO_STORE, problem } from '@/lib/api/upstream';
import { AUTH_COOKIE } from '@/lib/auth/constants';

const UPSTREAM_TIMEOUT_MS = 5_000;

/**
 * Keluar dari web: beri tahu backend (POST /auth/logout), lalu hapus cookie `at`.
 * Cookie tetap dihapus walaupun backend gagal dihubungi, agar pengguna tidak tertahan dalam sesi.
 */
export async function POST(request: NextRequest) {
  // Hanya dari halaman aplikasi ini sendiri, agar situs lain tidak bisa memaksa logout.
  const origin = request.headers.get('origin');
  if (origin && origin !== request.nextUrl.origin) {
    return problem(403, 'Permintaan ditolak', 'Asal permintaan tidak dikenal.');
  }

  const token = request.cookies.get(AUTH_COOKIE)?.value;
  if (token) {
    await fetch(`${API_URL}/auth/logout`, {
      method: 'POST',
      headers: { Accept: 'application/json', Authorization: `Bearer ${token}` },
      cache: 'no-store',
      signal: AbortSignal.timeout(UPSTREAM_TIMEOUT_MS),
    }).catch(() => undefined);
  }

  const response = NextResponse.json({ ok: true }, { headers: NO_STORE });
  response.cookies.delete(AUTH_COOKIE);
  return response;
}
