import type { Metadata } from 'next';
import Image from 'next/image';
import { ButtonLink } from '@/components/ui/Button';
import { AuthHeading, AuthIconBadge, AuthStack } from '@/features/auth/AuthParts';
import { AuthShell } from '@/features/auth/AuthShell';
import { employeeBrand, sessionEnded } from '@/features/auth/copy';

export const metadata: Metadata = { title: 'Sesi Berakhir · On-Track' };

type Props = { searchParams: Promise<{ portal?: string; alasan?: string }> };

/** W24 (admin) dan KW07 (karyawan) · Sesi berakhir karena tidak aktif atau token kedaluwarsa. */
export default async function SessionEndedPage({ searchParams }: Props) {
  const { portal, alasan } = await searchParams;
  const employee = portal === 'karyawan';
  const message =
    alasan === 'idle' ? sessionEnded.idle[employee ? 'employee' : 'admin'] : sessionEnded.expired;

  return (
    <AuthShell brand={employee ? employeeBrand : undefined}>
      <AuthStack>
        <AuthIconBadge icon="timer" tone="neutral" />
        <AuthHeading title={sessionEnded.title}>{message}</AuthHeading>
        <ButtonLink fullWidth href="/login">
          {sessionEnded.action}
          <Image src="/icons/arrow-right-inverse.svg" alt="" width={20} height={20} unoptimized />
        </ButtonLink>
      </AuthStack>
    </AuthShell>
  );
}
