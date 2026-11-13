import { cookies } from 'next/headers';
import { AUTH_COOKIE } from './constants';
import type { Role } from './roles';
import { readSession, type Session } from './session';

/** Sesi dari cookie untuk Server Component. Tanpa verifikasi tanda tangan: hanya untuk tampilan. */
export async function getSession(): Promise<Session | null> {
  const store = await cookies();
  return readSession(store.get(AUTH_COOKIE)?.value);
}

/** Peran yang boleh menambah dan mengubah data SDM (sesuai `x-rbac` di kontrak). */
export const canManageHr = (role: Role | undefined) =>
  role === 'owner' || role === 'hrd' || role === 'super_admin';
