import { NextResponse, type NextRequest } from 'next/server';
import type { components } from '@/lib/api/schema';
import {
  NO_STORE,
  postUpstream,
  problem,
  problemFromUpstream,
  readJsonBody,
  unreachable,
} from '@/lib/api/upstream';
import { AUTH_COOKIE } from '@/lib/auth/constants';
import { ROLE_HOME } from '@/lib/auth/roles';
import { readSession } from '@/lib/auth/session';

type LoginRequest = components['schemas']['LoginRequest'];
type LoginResponse = components['schemas']['LoginResponse'];

/** Judul per status. 401 sengaja generik: backend memakai 401 untuk sandi salah dan akun nonaktif. */
const titles: Record<number, string> = {
  400: 'Permintaan tidak valid',
  401: 'Gagal masuk',
  423: 'Akun dikunci sementara',
};

export async function POST(request: NextRequest) {
  const parsed = await readJsonBody(request);
  if ('error' in parsed) return parsed.error;

  const { email, password } = parsed.body;
  if (typeof email !== 'string' || typeof password !== 'string' || !email.trim() || !password) {
    return problem(400, titles[400], 'Email dan kata sandi wajib diisi.');
  }

  const upstream = await postUpstream('/auth/login', {
    email: email.trim(),
    password,
  } satisfies LoginRequest);
  if (!upstream) return unreachable();
  if (!upstream.ok) return problemFromUpstream(upstream, titles);

  const data = (await upstream.json().catch(() => null)) as Partial<LoginResponse> | null;
  if (!data?.access_token || typeof data.expires_in !== 'number') {
    return problem(502, 'Server bermasalah', 'Respons login dari server tidak lengkap.');
  }

  // Peran tanpa portal web (super_admin, cashier) akan langsung dilempar balik ke /login oleh
  // middleware. Tolak di sini dengan pesan yang jelas dan jangan simpan tokennya.
  const session = readSession(data.access_token);
  if (!session) {
    return problem(502, 'Server bermasalah', 'Token dari server tidak dapat dibaca.');
  }
  if (!ROLE_HOME[session.role]) {
    return problem(
      403,
      'Akun tidak memakai portal web',
      'Akun ini tidak bisa masuk lewat web. Gunakan aplikasi yang sesuai dengan peran Anda atau hubungi admin.',
    );
  }

  const secure =
    request.nextUrl.protocol === 'https:' || request.headers.get('x-forwarded-proto') === 'https';

  // Token hanya hidup di cookie httpOnly; badan respons tidak membawanya ke JavaScript browser.
  const response = NextResponse.json({ ok: true }, { headers: NO_STORE });
  response.cookies.set({
    name: AUTH_COOKIE,
    value: data.access_token,
    httpOnly: true,
    secure,
    sameSite: 'lax',
    path: '/',
    maxAge: data.expires_in,
  });
  return response;
}
