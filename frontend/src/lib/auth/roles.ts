/** Peran dari klaim `role` JWT. Harus sama dengan `models.UserRole` di backend. */
export const ROLES = ['super_admin', 'owner', 'hrd', 'team_lead', 'employee', 'cashier'] as const;

export type Role = (typeof ROLES)[number];

export const isRole = (value: unknown): value is Role =>
  typeof value === 'string' && (ROLES as readonly string[]).includes(value);

/** Rute yang dilindungi dan peran yang boleh membukanya. */
export const PROTECTED_AREAS = [
  { prefix: '/dashboard', roles: ['owner', 'hrd', 'team_lead'] },
  { prefix: '/karyawan', roles: ['employee'] },
] as const satisfies readonly { prefix: string; roles: readonly Role[] }[];

/**
 * Halaman awal tiap peran di web. `null` berarti peran itu belum punya portal web:
 * super_admin dikelola di level platform, cashier memakai aplikasi kasir.
 */
export const ROLE_HOME: Record<Role, string | null> = {
  owner: '/dashboard',
  hrd: '/dashboard',
  team_lead: '/dashboard',
  employee: '/karyawan',
  super_admin: null,
  cashier: null,
};
