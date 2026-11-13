import { MONTHS, toLocalDateKey } from '../lib/format';

/** Bulan berjalan "YYYY-MM" di zona waktu tenant. */
export function currentMonth(now: Date = new Date()): string {
  return (toLocalDateKey(now) ?? now.toISOString()).slice(0, 7);
}

/** Geser bulan: shiftMonth("2026-01", -1) → "2025-12". */
export function shiftMonth(month: string, delta: number): string {
  const [year, mon] = month.split('-').map(Number);
  const index = year * 12 + (mon - 1) + delta;
  const nextYear = Math.floor(index / 12);
  const nextMonth = (index % 12) + 1;
  return `${nextYear}-${String(nextMonth).padStart(2, '0')}`;
}

/** "2026-11" → "November 2026". */
export function monthLabel(month: string): string {
  const [year, mon] = month.split('-').map(Number);
  return `${MONTHS[mon - 1] ?? '?'} ${year}`;
}
