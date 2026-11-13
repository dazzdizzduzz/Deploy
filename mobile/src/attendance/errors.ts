import { ApiError } from '../api/client';

export type Failure = { title: string; body: string; action?: { label: string; onPress: () => void } };

type Handlers = {
  retry: () => void;
  refreshLocation: () => void;
  retakeSelfie: () => void;
  /** Kode yang ditangani khusus oleh layar (misalnya ALREADY_CLOCKED_IN). */
  special?: Partial<Record<string, (title: string, body: string) => Failure>>;
};

/**
 * Menerjemahkan error clock-in/clock-out menjadi pesan dan aksi. Pesan 4xx dari backend sudah
 * berbahasa Indonesia; error 5xx diganti pesan umum agar detail server tidak tampil.
 */
export function attendanceFailure(error: unknown, action: string, handlers: Handlers): Failure {
  if (!(error instanceof ApiError)) return { title: `${action} gagal`, body: 'Terjadi kesalahan. Coba lagi.' };
  if (error.status === 0) {
    return { title: 'Tidak dapat terhubung', body: error.message, action: { label: 'Coba lagi', onPress: handlers.retry } };
  }
  if (error.status >= 500) {
    return {
      title: 'Server sedang bermasalah',
      body: `${action} belum tersimpan. Coba lagi beberapa saat lagi.`,
      action: { label: 'Coba lagi', onPress: handlers.retry },
    };
  }
  const title = (error.detail as { title?: string } | undefined)?.title ?? `${action} gagal`;
  const special = error.code ? handlers.special?.[error.code] : undefined;
  if (special) return special(title, error.message);

  switch (error.code) {
    case 'OUTSIDE_GEOFENCE':
      return { title, body: error.message, action: { label: 'Perbarui lokasi', onPress: handlers.refreshLocation } };
    case 'SELFIE_REQUIRED':
      return { title, body: 'Ambil selfie dulu, lalu kirim lagi.', action: { label: 'Ambil selfie', onPress: handlers.retakeSelfie } };
    case 'PHOTO_TOO_LARGE':
    case 'PHOTO_UNSUPPORTED':
      return { title, body: error.message, action: { label: 'Ambil ulang', onPress: handlers.retakeSelfie } };
    default:
      return { title, body: error.message };
  }
}
