import { formatRupiahCompact } from '@/lib/format';

/**
 * Teks halaman login web admin (Figma W09). `brand.stats` sengaja berupa ilustrasi statis
 * dan JANGAN disambungkan ke API: panel ini tampil sebelum login, sehingga data asli
 * akan terlihat oleh siapa saja.
 */
export const adminLogin = {
  title: 'Masuk ke dashboard',
  subtitle: 'Untuk HRD, CEO/Owner, dan Team Lead.',
  note: 'Belum punya akun perusahaan? Daftarkan tenant (segera hadir).',
  brand: {
    headline: 'Kehadiran, task, dan payroll tim — real-time dalam satu dashboard.',
    subline: 'Dipakai Studio Tigapagi untuk 16 karyawan.',
    stats: [
      { value: '13/16', label: 'hadir hari ini' },
      { value: '1', label: 'task overdue' },
      { value: formatRupiahCompact('118000000.00'), label: 'payroll Oktober' },
    ],
  },
} as const;

/** Panel kiri portal karyawan (Figma KW07). Ilustrasi statis, sama alasannya dengan `adminLogin.brand`. */
export const employeeBrand = {
  headline: 'Absen, ajukan izin, dan cek slip gaji langsung dari browser.',
  subline: 'Portal karyawan Studio Tigapagi · gunakan email kantor kamu.',
  stats: [
    { value: '08:52', label: 'jam masuk kemarin' },
    { value: '9 hari', label: 'sisa cuti' },
    { value: '2 task', label: 'hari ini' },
  ],
} as const;

export type BrandCopy = { headline: string; subline: string; stats: readonly { value: string; label: string }[] };

/** Teks layar Sesi Berakhir (W24 admin, KW07 karyawan). */
export const sessionEnded = {
  title: 'Sesi kamu berakhir',
  idle: {
    admin: 'Demi keamanan data karyawan dan gaji, sesi otomatis berakhir setelah 30 menit tidak aktif.',
    employee: 'Demi keamanan data pribadi dan slip gaji kamu, sesi berakhir setelah 30 menit tidak aktif.',
  },
  expired: 'Masa berlaku login kamu sudah habis. Masuk lagi untuk melanjutkan.',
  action: 'Masuk lagi',
} as const;
