import type { Metadata } from 'next';
import { redirect } from 'next/navigation';
import { EmployeeForm } from '@/features/hr/EmployeeForm';
import { canManageHr, getSession } from '@/lib/auth/server';

export const metadata: Metadata = { title: 'Tambah Karyawan · On-Track' };

export default async function AddEmployeePage() {
  const session = await getSession();
  // Team Lead hanya boleh melihat daftar karyawan.
  if (!session || !canManageHr(session.role)) redirect('/dashboard/karyawan');
  return <EmployeeForm currentRole={session.role} />;
}
