import { sessionExpiredUrl, type SessionEndReason } from './constants';

/** Memanggil route logout. Kegagalan jaringan diabaikan: cookie di server tetap kedaluwarsa sendiri. */
export async function requestLogout() {
  await fetch('/api/auth/logout', { method: 'POST', credentials: 'same-origin', cache: 'no-store' }).catch(
    () => undefined,
  );
}

/** Portal karyawan memakai teks KW07; selain itu W24. */
export const isEmployeePortal = () => window.location.pathname.startsWith('/karyawan');

/** Akhiri sesi lalu buka layar Sesi Berakhir. */
export async function endSession(reason: SessionEndReason) {
  const employee = isEmployeePortal();
  if (reason === 'idle') await requestLogout();
  window.location.assign(sessionExpiredUrl({ employee, reason }));
}
