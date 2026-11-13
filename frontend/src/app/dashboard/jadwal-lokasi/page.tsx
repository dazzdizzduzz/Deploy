import type { Metadata } from 'next';
import { AttendanceSettings } from '@/features/attendance-settings/AttendanceSettings';
import { canManageHr, getSession } from '@/lib/auth/server';

export const metadata: Metadata = { title: 'Jadwal & Lokasi · On-Track' };

export default async function AttendanceSettingsPage() {
  const session = await getSession();
  return <AttendanceSettings canManage={canManageHr(session?.role)} />;
}
