import type { Metadata } from 'next';
import { redirect } from 'next/navigation';
import { LeaveApproval } from '@/features/leave-approval/LeaveApproval';
import { getSession } from '@/lib/auth/server';

export const metadata: Metadata = { title: 'Persetujuan Izin · On-Track' };

export default async function LeaveApprovalPage() {
  // /dashboard hanya untuk owner, hrd, dan team_lead (middleware); server membatasi team_lead ke timnya.
  const session = await getSession();
  if (!session) redirect('/login');
  return <LeaveApproval />;
}
