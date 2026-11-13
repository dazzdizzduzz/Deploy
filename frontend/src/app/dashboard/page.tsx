import type { Metadata } from 'next';
import styles from '@/features/hr/hr.module.css';

export const metadata: Metadata = { title: 'Dashboard · On-Track' };

/** Sementara: ringkasan dashboard dibuat di ticket tersendiri. */
export default function DashboardPage() {
  return (
    <main className={styles.main}>
      <div className={styles.titleBlock}>
        <h1 className={styles.title}>Dashboard</h1>
        <p className={styles.subtitle}>Ringkasan untuk HRD, CEO/Owner, dan Team Lead sedang disiapkan.</p>
      </div>
    </main>
  );
}
