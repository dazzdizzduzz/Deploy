import * as SecureStore from 'expo-secure-store';

/** Token akses dan waktu kedaluwarsanya (epoch ms). */
export type StoredSession = { accessToken: string; expiresAt: number };

const KEY = 'ontrack.session';

// Hanya bisa dibaca saat perangkat tidak terkunci, dan tidak ikut backup ke perangkat lain.
const options: SecureStore.SecureStoreOptions = {
  keychainAccessible: SecureStore.WHEN_UNLOCKED_THIS_DEVICE_ONLY,
};

export async function saveSession(session: StoredSession) {
  await SecureStore.setItemAsync(KEY, JSON.stringify(session), options);
}

/** Mengembalikan sesi yang masih berlaku, atau `null` (sesi kedaluwarsa ikut dihapus). */
export async function loadSession(now = Date.now()): Promise<StoredSession | null> {
  const raw = await SecureStore.getItemAsync(KEY, options);
  if (!raw) return null;
  try {
    const session = JSON.parse(raw) as Partial<StoredSession>;
    if (typeof session.accessToken === 'string' && typeof session.expiresAt === 'number' && session.expiresAt > now) {
      return { accessToken: session.accessToken, expiresAt: session.expiresAt };
    }
  } catch {
    // isi rusak: anggap tidak ada sesi
  }
  await clearSession();
  return null;
}

export async function clearSession() {
  await SecureStore.deleteItemAsync(KEY, options);
}
