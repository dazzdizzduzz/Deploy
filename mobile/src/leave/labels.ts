import type { LeaveStatus, LeaveType } from '../api/leave';
import { MONTHS, WEEKDAYS } from '../lib/format';

export const leaveTypeLabels: Record<LeaveType, string> = { cuti: 'Cuti tahunan', izin: 'Izin', sakit: 'Sakit' };

export const leaveTypeOptions = (['cuti', 'izin', 'sakit'] as const).map((value) => ({
  value,
  label: leaveTypeLabels[value],
}));

type Tone = 'success' | 'warning' | 'danger' | 'neutral';

/** Warna badge selalu disertai teks status. */
export const leaveStatusBadges: Record<LeaveStatus, { label: string; tone: Tone }> = {
  menunggu: { label: 'Menunggu', tone: 'warning' },
  disetujui: { label: 'Disetujui', tone: 'success' },
  ditolak: { label: 'Ditolak', tone: 'danger' },
};

export const leaveStatusBadge = (status: string) =>
  leaveStatusBadges[status as LeaveStatus] ?? { label: status, tone: 'neutral' as const };

export const REASON_MAX = 200;
/** Batas ukuran lampiran di draf kontrak. */
export const ATTACHMENT_MAX_BYTES = 5 * 1024 * 1024;
export const ATTACHMENT_TYPES = ['image/jpeg', 'image/png', 'application/pdf'];

const pad = (value: number) => String(value).padStart(2, '0');

/** Tanggal kalender dari pemilih tanggal (zona perangkat) → "YYYY-MM-DD". */
export const toDateKey = (date: Date) => `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;

/** "YYYY-MM-DD" → Date tengah hari, supaya tidak bergeser hari karena zona waktu. */
export function fromDateKey(key: string) {
  const [year, month, day] = key.split('-').map(Number);
  return new Date(year, month - 1, day, 12);
}

/** "2026-11-16" → "Senin, 16 Nov 2026". */
export function formatDateKey(key: string) {
  const date = fromDateKey(key);
  return `${WEEKDAYS[date.getDay()]}, ${date.getDate()} ${MONTHS[date.getMonth()].slice(0, 3)} ${date.getFullYear()}`;
}

/** "16 Nov – 18 Nov 2026" atau satu tanggal bila sama. */
export function formatRange(start: string, end: string) {
  if (start === end) return formatDateKey(start);
  const a = fromDateKey(start);
  const b = fromDateKey(end);
  const left = `${a.getDate()} ${MONTHS[a.getMonth()].slice(0, 3)}${a.getFullYear() === b.getFullYear() ? '' : ` ${a.getFullYear()}`}`;
  return `${left} – ${b.getDate()} ${MONTHS[b.getMonth()].slice(0, 3)} ${b.getFullYear()}`;
}

/**
 * Perkiraan hari kerja Senin–Jumat di perangkat, dipakai selama endpoint pratinjau belum aktif.
 * Hari libur nasional belum diperhitungkan; angka final tetap dari server.
 */
export function countWorkingDays(start: string, end: string) {
  const from = fromDateKey(start);
  const to = fromDateKey(end);
  if (to < from) return 0;
  let count = 0;
  for (const day = new Date(from); day <= to; day.setDate(day.getDate() + 1)) {
    const weekday = day.getDay();
    if (weekday !== 0 && weekday !== 6) count += 1;
  }
  return count;
}

export function formatBytes(bytes: number) {
  if (bytes < 1024 * 1024) return `${Math.max(1, Math.round(bytes / 1024))} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1).replace('.', ',')} MB`;
}
