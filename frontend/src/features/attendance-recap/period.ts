import { MONTHS, toLocalDateKey } from '@/lib/format';

/** Bulan berjalan "YYYY-MM" di zona waktu tenant. */
export const currentMonth = () => (toLocalDateKey(new Date()) ?? new Date().toISOString()).slice(0, 7);

export function shiftMonth(month: string, delta: number) {
  const [year, mon] = month.split('-').map(Number);
  const index = year * 12 + (mon - 1) + delta;
  return `${Math.floor(index / 12)}-${String((index % 12) + 1).padStart(2, '0')}`;
}

export function monthLabel(month: string) {
  const [year, mon] = month.split('-').map(Number);
  return `${MONTHS[mon - 1] ?? '?'} ${year}`;
}

/** 12 bulan terakhir sampai bulan berjalan, terbaru di atas. */
export const recentMonths = (count = 12) => {
  const now = currentMonth();
  return Array.from({ length: count }, (_, index) => shiftMonth(now, -index));
};

export const isMonth = (value: string | null): value is string => Boolean(value && /^\d{4}-(0[1-9]|1[0-2])$/.test(value));

/** 10350 → "172 j 30 m"; ringkas untuk kolom tabel. */
export function shortDuration(totalMinutes: number) {
  const minutes = Math.max(0, Math.round(totalMinutes));
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  if (hours === 0) return `${rest} m`;
  return rest ? `${hours} j ${rest} m` : `${hours} j`;
}
