/**
 * Helper format lokal On-Track (NFR-12): Rupiah, tanggal DD/MM/YYYY, jam 24 jam, dan konversi
 * waktu UTC dari API ke zona waktu tenant. Tanpa dependensi; dipakai web dan mobile.
 *
 * Sumber tunggal: shared/format.ts. Salinan di frontend/ dan mobile/ dibuat oleh shared/sync.mjs.
 * Ditulis dengan sintaks TypeScript yang bisa dihapus tipenya (tanpa enum/namespace) agar bisa diuji
 * langsung dengan Node.
 */

// ---------------------------------------------------------------------------
// Uang: string desimal dari API ("18000.00"), tidak pernah diubah ke float.
// ---------------------------------------------------------------------------

type Decimal = { negative: boolean; whole: bigint; cents: bigint };

// Konstanta BigInt tanpa literal `1n`, agar tetap terkompilasi untuk target ES2017 (web).
const ZERO = BigInt(0);
const ONE = BigInt(1);
const TEN = BigInt(10);
const FIFTY = BigInt(50);
const HUNDRED = BigInt(100);

/** "18000.00" | "-1500.5" | "7250000" → bagian bulat dan sen. `null` bila bukan angka desimal. */
export function parseDecimal(value: string | null | undefined): Decimal | null {
  if (value === null || value === undefined) return null;
  const match = /^(-)?(\d+)(?:\.(\d+))?$/.exec(value.trim());
  if (!match) return null;
  const fraction = (match[3] ?? '').padEnd(3, '0');
  // Bulatkan ke 2 desimal (setengah ke atas) memakai digit ketiga.
  let cents = BigInt(fraction.slice(0, 2));
  let whole = BigInt(match[2]);
  if (Number(fraction[2]) >= 5) cents += ONE;
  if (cents === HUNDRED) {
    cents = ZERO;
    whole += ONE;
  }
  return { negative: Boolean(match[1]) && (whole > ZERO || cents > ZERO), whole, cents };
}

const groupThousands = (digits: string) => digits.replace(/\B(?=(\d{3})+(?!\d))/g, '.');

export type RupiahOptions = {
  /** Tampilkan sen ("Rp 1.500,50"). Bawaan: tidak, dibulatkan ke rupiah terdekat. */
  cents?: boolean;
  /** Teks pengganti bila nilai kosong atau tidak valid. Bawaan: "—". */
  fallback?: string;
};

/**
 * "7250000.00" → "Rp 7.250.000"; "-1500.50" → "-Rp 1.501"; dengan `cents`: "Rp 1.500,50".
 * Pemisah ribuan titik dan desimal koma, sesuai kebiasaan Indonesia.
 */
export function formatRupiah(value: string | null | undefined, options: RupiahOptions = {}): string {
  const parsed = parseDecimal(value);
  if (!parsed) return options.fallback ?? '—';
  const { cents } = parsed;
  let { whole } = parsed;
  if (!options.cents && cents >= FIFTY) whole += ONE;
  const body = groupThousands(whole.toString());
  const tail = options.cents ? `,${cents.toString().padStart(2, '0')}` : '';
  const negative = parsed.negative && (whole > ZERO || (options.cents === true && cents > ZERO));
  return `${negative ? '-' : ''}Rp ${body}${tail}`;
}

/** Ringkas untuk kartu ringkasan: "118000000.00" → "Rp 118 jt", "2500000000" → "Rp 2,5 M". */
export function formatRupiahCompact(value: string | null | undefined, fallback = '—'): string {
  const parsed = parseDecimal(value);
  if (!parsed) return fallback;
  const units: [bigint, string][] = [
    [BigInt('1000000000000'), 'T'],
    [BigInt('1000000000'), 'M'],
    [BigInt('1000000'), 'jt'],
    [BigInt('1000'), 'rb'],
  ];
  const sign = parsed.negative ? '-' : '';
  for (const [size, label] of units) {
    if (parsed.whole >= size) {
      // Satu angka di belakang koma, tanpa ",0".
      const tenths = (parsed.whole * TEN) / size;
      const integer = tenths / TEN;
      const decimal = tenths % TEN;
      return `${sign}Rp ${groupThousands(integer.toString())}${decimal > ZERO ? `,${decimal}` : ''} ${label}`;
    }
  }
  return formatRupiah(value, { fallback });
}

/** Isian form "7.250.000" atau "Rp 7.250.000" → "7250000.00" untuk dikirim ke API. */
export function toDecimalString(input: string): string | null {
  const digits = input.replace(/\D/g, '').replace(/^0+(?=\d)/, '');
  return digits ? `${digits}.00` : null;
}

// ---------------------------------------------------------------------------
// Waktu: API mengirim ISO 8601 UTC; tampilkan di zona waktu tenant.
// ---------------------------------------------------------------------------

/** Zona waktu bawaan sampai backend mengirim zona waktu tenant. */
export const DEFAULT_TIME_ZONE = 'Asia/Makassar';

/** Indonesia tidak memakai DST, jadi selisihnya tetap dan tidak bergantung pada dukungan Intl. */
const INDONESIA_ZONES: Record<string, { offsetMinutes: number; label: string }> = {
  'Asia/Jakarta': { offsetMinutes: 7 * 60, label: 'WIB' },
  'Asia/Pontianak': { offsetMinutes: 7 * 60, label: 'WIB' },
  'Asia/Makassar': { offsetMinutes: 8 * 60, label: 'WITA' },
  'Asia/Jayapura': { offsetMinutes: 9 * 60, label: 'WIT' },
  UTC: { offsetMinutes: 0, label: 'UTC' },
};

export const MONTHS = [
  'Januari', 'Februari', 'Maret', 'April', 'Mei', 'Juni',
  'Juli', 'Agustus', 'September', 'Oktober', 'November', 'Desember',
] as const;

export const WEEKDAYS = ['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'] as const;

/** Tanggal dan jam di zona waktu tertentu. `month` 1–12, `weekday` 0 = Minggu. */
export type ZonedParts = {
  year: number;
  month: number;
  day: number;
  hour: number;
  minute: number;
  second: number;
  weekday: number;
};

const DATE_ONLY = /^(\d{4})-(\d{2})-(\d{2})$/;

/**
 * Mengubah nilai dari API menjadi Date. Tanggal tanpa jam ("2025-01-02") tidak punya zona waktu,
 * jadi dikembalikan `null` di sini dan ditangani sebagai tanggal kalender oleh pemanggil.
 */
function toInstant(value: string | number | Date): Date | null {
  if (value instanceof Date) return Number.isNaN(value.getTime()) ? null : value;
  if (typeof value === 'number') return Number.isFinite(value) ? new Date(value) : null;
  if (DATE_ONLY.test(value.trim())) return null;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? null : date;
}

function partsWithIntl(date: Date, timeZone: string): ZonedParts | null {
  try {
    const parts = new Intl.DateTimeFormat('en-US', {
      timeZone,
      hourCycle: 'h23',
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      weekday: 'short',
    }).formatToParts(date);
    const get = (type: string) => parts.find((part) => part.type === type)?.value ?? '';
    const weekday = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].indexOf(get('weekday'));
    return {
      year: Number(get('year')),
      month: Number(get('month')),
      day: Number(get('day')),
      hour: Number(get('hour')) % 24,
      minute: Number(get('minute')),
      second: Number(get('second')),
      weekday,
    };
  } catch {
    return null;
  }
}

/** Komponen tanggal dan jam di zona waktu `timeZone`. `null` bila nilai tidak valid. */
export function toZonedParts(
  value: string | number | Date | null | undefined,
  timeZone: string = DEFAULT_TIME_ZONE,
): ZonedParts | null {
  if (value === null || value === undefined || value === '') return null;
  if (typeof value === 'string') {
    const match = DATE_ONLY.exec(value.trim());
    if (match) {
      const [year, month, day] = [Number(match[1]), Number(match[2]), Number(match[3])];
      const check = new Date(Date.UTC(year, month - 1, day));
      if (check.getUTCMonth() !== month - 1) return null;
      return { year, month, day, hour: 0, minute: 0, second: 0, weekday: check.getUTCDay() };
    }
  }
  const date = toInstant(value);
  if (!date) return null;

  const fixed = INDONESIA_ZONES[timeZone];
  if (fixed) {
    const shifted = new Date(date.getTime() + fixed.offsetMinutes * 60_000);
    return {
      year: shifted.getUTCFullYear(),
      month: shifted.getUTCMonth() + 1,
      day: shifted.getUTCDate(),
      hour: shifted.getUTCHours(),
      minute: shifted.getUTCMinutes(),
      second: shifted.getUTCSeconds(),
      weekday: shifted.getUTCDay(),
    };
  }
  return partsWithIntl(date, timeZone);
}

const pad = (value: number) => String(value).padStart(2, '0');

/** "2026-11-10T16:30:00Z" → "11/11/2026" di WITA. Tanggal tanpa jam tidak digeser zona waktu. */
export function formatDate(value: string | number | Date | null | undefined, timeZone?: string, fallback = '—') {
  const parts = toZonedParts(value, timeZone);
  return parts ? `${pad(parts.day)}/${pad(parts.month)}/${parts.year}` : fallback;
}

/** "2026-11-10T00:05:00Z" → "08:05" di WITA (format 24 jam). */
export function formatTime(value: string | number | Date | null | undefined, timeZone?: string, fallback = '—') {
  if (typeof value === 'string' && DATE_ONLY.test(value.trim())) return fallback;
  const parts = toZonedParts(value, timeZone);
  return parts ? `${pad(parts.hour)}:${pad(parts.minute)}` : fallback;
}

/** "11/11/2026 00:30". */
export function formatDateTime(value: string | number | Date | null | undefined, timeZone?: string, fallback = '—') {
  const date = formatDate(value, timeZone, '');
  const time = formatTime(value, timeZone, '');
  return date && time ? `${date} ${time}` : date || fallback;
}

/** "Rabu, 11 November 2026" untuk judul halaman dan sapaan. */
export function formatLongDate(value: string | number | Date | null | undefined, timeZone?: string, fallback = '—') {
  const parts = toZonedParts(value, timeZone);
  return parts ? `${WEEKDAYS[parts.weekday]}, ${parts.day} ${MONTHS[parts.month - 1]} ${parts.year}` : fallback;
}

/** Kunci tanggal lokal "YYYY-MM-DD" untuk mengelompokkan data per hari di zona waktu tenant. */
export function toLocalDateKey(value: string | number | Date | null | undefined, timeZone?: string): string | null {
  const parts = toZonedParts(value, timeZone);
  return parts ? `${parts.year}-${pad(parts.month)}-${pad(parts.day)}` : null;
}

/** "WITA" untuk Asia/Makassar; zona lain ditulis apa adanya. */
export function timeZoneLabel(timeZone: string = DEFAULT_TIME_ZONE): string {
  return INDONESIA_ZONES[timeZone]?.label ?? timeZone;
}
