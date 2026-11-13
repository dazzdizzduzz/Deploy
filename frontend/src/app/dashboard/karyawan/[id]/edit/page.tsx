import type { Metadata } from 'next';
import { redirect } from 'next/navigation';
import { EmployeeForm } from '@/features/hr/EmployeeForm';
import { canManageHr, getSession } from '@/lib/auth/server';

export const metadata: Metadata = { title: 'Edit Karyawan · On-Track' };

type PageProps = { params: Promise<{ id: string }> };

export default async function EditEmployeePage({ params }: PageProps) {
  const [session, { id }] = await Promise.all([getSession(), params]);
  if (!session || !canManageHr(session.role)) redirect('/dashboard/karyawan');
  return <EmployeeForm userId={id} currentRole={session.role} />;
}
