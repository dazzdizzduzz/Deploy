import { NextResponse, type NextRequest } from 'next/server';
import { AUTH_COOKIE, sessionExpiredUrl } from '@/lib/auth/constants';
import { PROTECTED_AREAS, ROLE_HOME } from '@/lib/auth/roles';
import { readExpiredRole, readSession } from '@/lib/auth/session';

const LOGIN_PATH = '/login';

const matchesPrefix = (pathname: string, prefix: string) =>
  pathname === prefix || pathname.startsWith(`${prefix}/`);

function redirect(request: NextRequest, pathname: string) {
  const url = request.nextUrl.clone();
  url.pathname = pathname;
  url.search = '';
  return NextResponse.redirect(url);
}

/** Ke /login sambil menghapus cookie yang kedaluwarsa, rusak, atau perannya tanpa portal web. */
function toLogin(request: NextRequest, clearCookie: boolean) {
  const response = redirect(request, LOGIN_PATH);
  if (clearCookie) response.cookies.delete(AUTH_COOKIE);
  return response;
}

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const token = request.cookies.get(AUTH_COOKIE)?.value;
  const session = readSession(token);
  const home = session ? ROLE_HOME[session.role] : null;
  const area = PROTECTED_AREAS.find(({ prefix }) => matchesPrefix(pathname, prefix));

  if (pathname === LOGIN_PATH) {
    // Sudah punya sesi yang bisa dipakai: tidak perlu login lagi.
    if (home) return redirect(request, home);
    // Hapus cookie yang tidak bisa dipakai agar tidak terus terbawa.
    if (token) {
      const response = NextResponse.next();
      response.cookies.delete(AUTH_COOKIE);
      return response;
    }
    return NextResponse.next();
  }

  if (pathname === '/') {
    return home ? redirect(request, home) : toLogin(request, Boolean(token));
  }

  if (!area) return NextResponse.next();

  // Token kedaluwarsa: tampilkan Sesi Berakhir (W24/KW07), bukan langsung form login.
  const expiredRole = session ? null : readExpiredRole(token);
  if (expiredRole) {
    const url = new URL(sessionExpiredUrl({ employee: expiredRole === 'employee', reason: 'expired' }), request.url);
    const response = NextResponse.redirect(url);
    response.cookies.delete(AUTH_COOKIE);
    return response;
  }

  // Rute dilindungi tanpa sesi yang bisa dipakai.
  if (!session || !home) return toLogin(request, Boolean(token));

  // Peran tidak boleh membuka area ini: arahkan ke halaman awal perannya sendiri.
  if (!(area.roles as readonly string[]).includes(session.role)) return redirect(request, home);

  return NextResponse.next();
}

export const config = {
  // Lewati API, aset Next.js, dan file statis di /public.
  matcher: ['/((?!api|_next/static|_next/image|favicon.ico|icons/).*)'],
};
