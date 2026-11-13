import { NextResponse, type NextRequest } from 'next/server';
import { PROBLEM_CONTENT_TYPE } from '@/lib/api/problem';
import { isSameOrigin } from '@/lib/api/same-origin';
import { API_URL, NO_STORE, problem, problemFromUpstream, unreachable } from '@/lib/api/upstream';
import { AUTH_COOKIE } from '@/lib/auth/constants';

/**
 * Meneruskan permintaan browser ke backend Go sambil menempelkan token dari cookie httpOnly.
 * Token tidak pernah sampai ke JavaScript browser. Hanya resource di bawah ini yang diteruskan,
 * agar route ini tidak menjadi proxy terbuka ke seluruh API.
 */
const ALLOWED_RESOURCES = new Set(['departments', 'positions', 'users', 'locations', 'schedules', 'attendance', 'leave-requests']);
const UPSTREAM_TIMEOUT_MS = 10_000;
const SEGMENT = /^[A-Za-z0-9_-]+$/;

type Context = { params: Promise<{ path: string[] }> };

async function forward(request: NextRequest, { params }: Context) {
  const { path } = await params;
  if (!ALLOWED_RESOURCES.has(path[0]) || !path.every((segment) => SEGMENT.test(segment))) {
    return problem(404, 'Tidak ditemukan', 'Resource API tidak tersedia.');
  }

  const token = request.cookies.get(AUTH_COOKIE)?.value;
  if (!token) return problem(401, 'Sesi berakhir', 'Silakan masuk kembali.');

  const method = request.method;
  const hasBody = method === 'POST' || method === 'PATCH';
  if (method !== 'GET') {
    // Permintaan yang mengubah data harus berasal dari halaman aplikasi ini sendiri.
    if (!isSameOrigin(request)) {
      return problem(403, 'Permintaan ditolak', 'Asal permintaan tidak dikenal.');
    }
    if (hasBody && !request.headers.get('content-type')?.startsWith('application/json')) {
      return problem(415, 'Format tidak didukung', 'Kirim data dalam format JSON.');
    }
  }

  let upstream: Response;
  try {
    upstream = await fetch(`${API_URL}/${path.join('/')}${request.nextUrl.search}`, {
      method,
      headers: {
        Accept: 'application/json',
        Authorization: `Bearer ${token}`,
        ...(hasBody && { 'Content-Type': 'application/json' }),
      },
      body: hasBody ? await request.text() : undefined,
      cache: 'no-store',
      signal: AbortSignal.timeout(UPSTREAM_TIMEOUT_MS),
    });
  } catch {
    return unreachable();
  }

  if (upstream.status === 401) {
    // Token ditolak backend (kedaluwarsa atau dicabut): hapus cookie agar middleware mengarahkan ke /login.
    const response = await problemFromUpstream(upstream, { 401: 'Sesi berakhir' });
    response.cookies.delete(AUTH_COOKIE);
    return response;
  }

  if (!upstream.ok) {
    const contentType = upstream.headers.get('content-type') ?? '';
    if (upstream.status < 500 && contentType.startsWith(PROBLEM_CONTENT_TYPE)) {
      // Sudah RFC 7807 (termasuk `errors` per field untuk 422): teruskan apa adanya.
      return new NextResponse(await upstream.text(), {
        status: upstream.status,
        headers: { ...NO_STORE, 'Content-Type': PROBLEM_CONTENT_TYPE },
      });
    }
    return problemFromUpstream(upstream, {
      403: 'Tidak punya akses',
      404: 'Data tidak ditemukan',
      409: 'Data sudah ada',
      422: 'Data tidak valid',
    });
  }

  return new NextResponse(upstream.status === 204 ? null : await upstream.text(), {
    status: upstream.status,
    headers: { ...NO_STORE, 'Content-Type': upstream.headers.get('content-type') ?? 'application/json' },
  });
}

export { forward as GET, forward as POST, forward as PATCH, forward as DELETE };
