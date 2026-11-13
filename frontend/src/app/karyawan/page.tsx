import type { Metadata } from 'next';

export const metadata: Metadata = { title: 'Beranda Karyawan · On-Track' };

/** Sementara: halaman tujuan karyawan sampai portal karyawan dibuat. */
export default function EmployeeHomePage() {
  return (
    <main className="home">
      <h1>Beranda karyawan</h1>
      <p>Absen, izin, task, dan slip gaji. Halaman ini sedang disiapkan.</p>
    </main>
  );
}
