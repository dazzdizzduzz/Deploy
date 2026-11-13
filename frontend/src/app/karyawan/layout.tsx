import Image from 'next/image';
import type { ReactNode } from 'react';
import { LogoutButton } from '@/features/auth/LogoutButton';
import { getSession } from '@/lib/auth/server';
import { SessionGuard } from '@/lib/auth/SessionGuard';
import styles from './karyawan.module.css';

/** Kerangka portal web karyawan: bar atas dengan logo dan tombol Keluar. */
export default async function EmployeeLayout({ children }: { children: ReactNode }) {
  const session = await getSession();
  return (
    <div className={styles.layout}>
      <header className={styles.bar}>
        <div className={styles.logo}>
          <Image src="/icons/logo-mark.svg" alt="" width={14} height={14} unoptimized />
          <span>On-Track</span>
        </div>
        <LogoutButton className={styles.logout} />
      </header>
      {session && <SessionGuard expiresAt={session.exp * 1000} />}
      {children}
    </div>
  );
}
