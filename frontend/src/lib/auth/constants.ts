/** Nama cookie httpOnly berisi access token. Hanya Route Handler yang menyentuhnya. */
export const AUTH_COOKIE = 'at';

/** Tujuan setelah login. Middleware meneruskan `/` ke halaman awal sesuai peran. */
export const LOGIN_REDIRECT = '/';

/** Halaman Sesi Berakhir (W24, KW07). */
export const SESSION_EXPIRED_PATH = '/sesi-berakhir';

/** Sesi web berakhir setelah 30 menit tanpa aktivitas. */
export const IDLE_TIMEOUT_MS = 30 * 60 * 1000;

export type SessionEndReason = 'idle' | 'expired';

/** Alamat Sesi Berakhir sesuai portal (admin atau karyawan) dan penyebabnya. */
export function sessionExpiredUrl({ employee, reason }: { employee: boolean; reason: SessionEndReason }) {
  const query = new URLSearchParams({ alasan: reason });
  if (employee) query.set('portal', 'karyawan');
  return `${SESSION_EXPIRED_PATH}?${query}`;
}
