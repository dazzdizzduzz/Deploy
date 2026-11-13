import type { Metadata } from 'next';
import { redirect } from 'next/navigation';
import { Suspense } from 'react';
import { AttendanceRecap } from '@/features/attendance-recap/AttendanceRecap';
import { canManageHr, getSession } from '@/lib/auth/server';

export const metadata: Metadata = { title: 'Rekap Kehadiran · On-Track' };

export default async function AttendanceRecapPage() {
  const session = await getSession();
  if (!session) redirect('/login');
  // Ekspor PDF hanya untuk owner/hrd (x-rbac); team_lead tetap bisa melihat rekap.
  return (
    <Suspense>
      <AttendanceRecap canExport={canManageHr(session.role)} />
    </Suspense>
  );
}
