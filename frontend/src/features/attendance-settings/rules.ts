/** Batas radius geofence dari kontrak (`radius_meters` 25–500). */
export const RADIUS_MIN = 25;
export const RADIUS_MAX = 500;
export const RADIUS_STEP = 5;

/** Urutan hari kerja; disimpan backend sebagai teks "Sen,Sel,Rab,Kam,Jum". */
export const DAYS = ['Sen', 'Sel', 'Rab', 'Kam', 'Jum', 'Sab', 'Min'] as const;

export const parseDays = (value: string) =>
  value
    .split(',')
    .map((day) => day.trim())
    .filter((day) => (DAYS as readonly string[]).includes(day));

export const joinDays = (days: string[]) => DAYS.filter((day) => days.includes(day)).join(',');

/** Ringkas hari kerja, misalnya "Sen–Jum" atau "Sen, Rab, Jum". */
export function summarizeDays(value: string) {
  const days = parseDays(value);
  if (days.length === 0) return '—';
  const indexes = days.map((day) => DAYS.indexOf(day as (typeof DAYS)[number]));
  const consecutive = indexes.every((index, i) => i === 0 || index === indexes[i - 1] + 1);
  return consecutive && days.length > 2 ? `${days[0]}–${days[days.length - 1]}` : days.join(', ');
}

/** "08:00" → menit sejak tengah malam, atau null bila format salah. */
export function toMinutes(time: string): number | null {
  const match = /^([01]\d|2[0-3]):([0-5]\d)$/.exec(time);
  return match ? Number(match[1]) * 60 + Number(match[2]) : null;
}

/** Backend bisa mengirim "08:00:00"; input jam memakai "08:00". */
export const toTimeInput = (value: string) => value.slice(0, 5);

export function formatMinutes(total: number) {
  if (total < 60) return `${total} menit`;
  const hours = Math.floor(total / 60);
  const minutes = total % 60;
  return minutes ? `${hours} jam ${minutes} menit` : `${hours} jam`;
}

/** Validasi koordinat: lintang -90..90, bujur -180..180. */
export const isLatitude = (value: number) => Number.isFinite(value) && value >= -90 && value <= 90;
export const isLongitude = (value: number) => Number.isFinite(value) && value >= -180 && value <= 180;
