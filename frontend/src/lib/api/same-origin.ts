import type { NextRequest } from 'next/server';

const firstValue = (header: string | null) => header?.split(',')[0]?.trim() || null;

function hostOf(url: string | undefined): string | null {
  if (!url) return null;
  try {
    return new URL(url).host;
  } catch {
    return null;
  }
}

/**
 * Memastikan permintaan yang mengubah data berasal dari halaman aplikasi ini sendiri.
 *
 * Yang dibandingkan hanya host (nama domain + port), bukan protokol. Di belakang reverse proxy
 * HTTPS, Next.js menerima permintaan sebagai `http://`, sehingga `request.nextUrl.origin` bisa
 * berbeda protokol dengan `Origin` dari browser walau domainnya sama.
 *
 * Host yang diterima: host permintaan, `X-Forwarded-Host` dari reverse proxy, dan `APP_ORIGIN`
 * bila diatur. Browser tidak mengizinkan situs lain memalsukan header `Host`, dan header kustom
 * seperti `X-Forwarded-Host` dari situs lain memicu preflight CORS yang tidak kita jawab.
 * Permintaan tanpa `Origin` (bukan dari browser) diloloskan; backend tetap memeriksa token.
 */
export function isSameOrigin(request: NextRequest): boolean {
  const origin = request.headers.get('origin');
  if (!origin) return true;

  const originHost = hostOf(origin);
  if (!originHost) return false;

  const allowed = [
    request.nextUrl.host,
    firstValue(request.headers.get('x-forwarded-host')),
    firstValue(request.headers.get('host')),
    hostOf(process.env.APP_ORIGIN),
  ];
  return allowed.includes(originHost);
}
