import type { Metadata } from 'next';
import { OrganizationPage } from '@/features/hr/OrganizationPage';
import { canManageHr, getSession } from '@/lib/auth/server';

export const metadata: Metadata = { title: 'Struktur Organisasi · On-Track' };

type PageProps = { searchParams: Promise<{ tab?: string }> };

export default async function OrganizationRoute({ searchParams }: PageProps) {
  const [session, { tab }] = await Promise.all([getSession(), searchParams]);
  return (
    <OrganizationPage tab={tab === 'jabatan' ? 'jabatan' : 'departemen'} canManage={canManageHr(session?.role)} />
  );
}
