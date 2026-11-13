import type { Attendance } from '../api/attendance';

/** Status kehadiran (FR-08) dalam bahasa Indonesia untuk badge. */
export function statusLabel(record: Pick<Attendance, 'status' | 'late_minutes'>) {
  switch (record.status) {
    case 'on_time':
      return { label: 'Hadir', tone: 'success' as const };
    case 'late':
      return {
        label: record.late_minutes > 0 ? `Terlambat ${record.late_minutes} menit` : 'Terlambat',
        tone: 'warning' as const,
      };
    default:
      return { label: 'Tidak hadir', tone: 'danger' as const };
  }
}

/** 512 → "8 jam 32 menit", 45 → "45 menit", 0 → "0 menit". */
export function formatDuration(totalMinutes: number) {
  const minutes = Math.max(0, Math.round(totalMinutes));
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  if (hours === 0) return `${rest} menit`;
  return rest ? `${hours} jam ${rest} menit` : `${hours} jam`;
}

/** Lama kerja dari clock-in sampai clock-out (menit penuh), atau null bila belum clock-out. */
export function workedMinutes(record: Pick<Attendance, 'clock_in' | 'clock_out'>): number | null {
  if (!record.clock_out) return null;
  const start = Date.parse(record.clock_in);
  const end = Date.parse(record.clock_out);
  if (Number.isNaN(start) || Number.isNaN(end) || end < start) return null;
  return Math.floor((end - start) / 60_000);
}

type Tone = 'success' | 'warning' | 'danger' | 'neutral' | 'inverse';

/** Badge status harian di riwayat (FR-10). Warna selalu disertai teks. */
export const dayStatusBadges: Record<string, { label: string; tone: Tone }> = {
  on_time: { label: 'Hadir', tone: 'success' },
  late: { label: 'Terlambat', tone: 'warning' },
  izin: { label: 'Izin', tone: 'neutral' },
  cuti: { label: 'Cuti', tone: 'neutral' },
  sakit: { label: 'Sakit', tone: 'neutral' },
  alpha: { label: 'Alpha', tone: 'danger' },
  absent: { label: 'Alpha', tone: 'danger' },
  libur: { label: 'Libur', tone: 'inverse' },
};

export const dayStatusBadge = (status: string) => dayStatusBadges[status] ?? { label: status, tone: 'neutral' as const };
