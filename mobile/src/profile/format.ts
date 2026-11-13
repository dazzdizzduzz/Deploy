export const roleLabels: Record<string, string> = {
  super_admin: 'Super Admin',
  owner: 'Owner',
  hrd: 'HRD',
  team_lead: 'Team Lead',
  employee: 'Karyawan',
  cashier: 'Kasir',
};

export const employmentLabels: Record<string, string> = {
  tetap: 'Karyawan tetap',
  kontrak: 'Kontrak',
  magang: 'Magang',
};

export function initialsOf(name: string | null) {
  if (!name) return '?';
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((word) => word[0]?.toUpperCase())
    .join('');
}

export const firstName = (name: string | null) => name?.trim().split(/\s+/)[0] ?? null;

/** Salam sesuai jam di perangkat. */
export function greeting(date = new Date()) {
  const hour = date.getHours();
  if (hour < 11) return 'Selamat pagi';
  if (hour < 15) return 'Selamat siang';
  if (hour < 18) return 'Selamat sore';
  return 'Selamat malam';
}

const formatter = (options: Intl.DateTimeFormatOptions) => {
  try {
    return new Intl.DateTimeFormat('id-ID', options);
  } catch {
    return null;
  }
};

const longDate = formatter({ weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' });
const plainDate = formatter({ day: 'numeric', month: 'long', year: 'numeric' });

export const formatToday = (date = new Date()) => longDate?.format(date) ?? date.toDateString();

/** "2025-01-02" → "2 Januari 2025". Tanggal tanpa jam dibaca sebagai tanggal lokal. */
export function formatDate(value: string | null) {
  if (!value) return null;
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  const date = match ? new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3])) : new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return plainDate?.format(date) ?? value;
}
