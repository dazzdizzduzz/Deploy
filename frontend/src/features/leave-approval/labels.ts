import { MONTHS } from '@/lib/format';
import type { LeaveStatus, LeaveType } from './api';

export const leaveTypeLabels: Record<LeaveType, string> = { cuti: 'Cuti tahunan', izin: 'Izin', sakit: 'Sakit' };

type Tone = 'success' | 'warning' | 'danger' | 'neutral';

/** Warna badge selalu disertai teks status. */
export const leaveStatusBadges: Record<LeaveStatus, { label: string; tone: Tone }> = {
  menunggu: { label: 'Menunggu', tone: 'warning' },
  disetujui: { label: 'Disetujui', tone: 'success' },
  ditolak: { label: 'Ditolak', tone: 'danger' },
};

export const NOTE_MAX = 200;

/** "2026-11-16" → "16 Nov 2026". Tanggal kalender, jadi tidak dikonversi zona waktu. */
export function formatDateKey(key: string) {
  const [year, month, day] = key.split('-').map(Number);
  if (!year || !month || !day) return key;
  return `${day} ${MONTHS[month - 1].slice(0, 3)} ${year}`;
}

/** "16–18 Nov 2026", "30 Des 2026 – 2 Jan 2027", atau satu tanggal bila sama. */
export function formatRange(start: string, end: string) {
  if (start === end) return formatDateKey(start);
  const [y1, m1, d1] = start.split('-').map(Number);
  const [y2, m2] = end.split('-').map(Number);
  if (y1 === y2 && m1 === m2) return `${d1}–${formatDateKey(end)}`;
  if (y1 === y2) return `${d1} ${MONTHS[m1 - 1].slice(0, 3)} – ${formatDateKey(end)}`;
  return `${formatDateKey(start)} – ${formatDateKey(end)}`;
}
