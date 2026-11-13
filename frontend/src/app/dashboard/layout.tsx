import type { ReactNode } from 'react';
import { AppShell } from '@/components/layout/AppShell';
import { AppSidebar, type SidebarItem } from '@/components/layout/AppSidebar';
import { roleLabels } from '@/features/hr/labels';
import { getSession } from '@/lib/auth/server';
import { SessionGuard } from '@/lib/auth/SessionGuard';

const allItems: SidebarItem[] = [
  { href: '/dashboard', label: 'Dashboard', icon: 'nav-grid', activeIcon: 'nav-grid-active', exact: true },
  { href: '/dashboard/karyawan', label: 'Karyawan', icon: 'nav-users', activeIcon: 'nav-users-active' },
  {
    href: '/dashboard/rekap-kehadiran',
    label: 'Rekap Kehadiran',
    icon: 'nav-calendar',
    activeIcon: 'nav-calendar-active',
  },
  {
    href: '/dashboard/persetujuan-izin',
    label: 'Persetujuan Izin',
    icon: 'nav-inbox',
    activeIcon: 'nav-inbox-active',
  },
  {
    href: '/dashboard/jadwal-lokasi',
    label: 'Jadwal & Lokasi',
    icon: 'nav-clock',
    activeIcon: 'nav-clock-active',
  },
  {
    href: '/dashboard/struktur-organisasi',
    label: 'Struktur Organisasi',
    icon: 'nav-building',
    activeIcon: 'nav-building-active',
  },
];

/**
 * Menu yang endpoint backend-nya belum aktif. Halamannya tetap ada, hanya tidak ditampilkan
 * di sidebar agar pengguna tidak membuka layar kosong. Hapus path dari daftar ini begitu
 * endpoint-nya tersedia di backend.
 */
const PENDING_BACKEND = new Set([
  '/dashboard/rekap-kehadiran', // GET /attendance belum ada (backend baru punya /attendances/clock-in|out)
  '/dashboard/persetujuan-izin', // /leave-requests belum ada
  '/dashboard/struktur-organisasi', // /departments dan /positions belum ada
]);

const items = allItems.filter((item) => !PENDING_BACKEND.has(item.href));

export default async function DashboardLayout({ children }: { children: ReactNode }) {
  const session = await getSession();
  const roleLabel = session ? roleLabels[session.role as keyof typeof roleLabels] : '';
  return (
    <AppShell sidebar={<AppSidebar items={items} roleLabel={roleLabel} />}>
      {session && <SessionGuard expiresAt={session.exp * 1000} />}
      {children}
    </AppShell>
  );
}
