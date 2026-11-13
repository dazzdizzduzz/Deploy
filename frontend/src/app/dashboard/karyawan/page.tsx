import type { Metadata } from 'next';
import { EmployeeList } from '@/features/hr/EmployeeList';
import { canManageHr, getSession } from '@/lib/auth/server';

export const metadata: Metadata = { title: 'Karyawan · On-Track' };

type PageProps = { searchParams: Promise<{ ditambahkan?: string; diperbarui?: string }> };

export default async function EmployeesPage({ searchParams }: PageProps) {
  const [session, { ditambahkan, diperbarui }] = await Promise.all([getSession(), searchParams]);
  return (
    <EmployeeList
      canManage={canManageHr(session?.role)}
      addedEmail={typeof ditambahkan === 'string' ? ditambahkan : undefined}
      updatedName={typeof diperbarui === 'string' ? diperbarui : undefined}
    />
  );
}
