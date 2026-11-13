import type { EmployeeRole, EmploymentType, UserStatus } from './types';

export const roleLabels: Record<EmployeeRole | 'super_admin', string> = {
  super_admin: 'Super Admin',
  owner: 'Owner',
  hrd: 'HRD',
  team_lead: 'Team Lead',
  employee: 'Karyawan',
  cashier: 'Kasir',
};

export const employmentLabels: Record<EmploymentType, string> = {
  tetap: 'Tetap',
  kontrak: 'Kontrak',
  magang: 'Magang',
};

export const statusBadges: Record<UserStatus, { label: string; tone: 'success' | 'neutral' | 'warning' }> = {
  aktif: { label: 'Aktif', tone: 'success' },
  nonaktif: { label: 'Nonaktif', tone: 'neutral' },
  diundang: { label: 'Diundang', tone: 'warning' },
};

export function initialsOf(name: string) {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((word) => word[0]?.toUpperCase())
    .join('');
}
