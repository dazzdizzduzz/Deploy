import { Camera } from 'expo-camera';
import * as Location from 'expo-location';
import * as SecureStore from 'expo-secure-store';
import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import { AppState } from 'react-native';

export type Permission = 'lokasi' | 'kamera';

/** granted: diizinkan · undetermined: belum ditanya · denied: ditolak, masih bisa ditanya · blocked: harus lewat Pengaturan */
export type PermissionStatus = 'granted' | 'undetermined' | 'denied' | 'blocked';

type RawPermission = { granted: boolean; canAskAgain: boolean; status: string };

export type PermissionResult = {
  granted: boolean;
  missing: Permission[];
  /** `false` bila minimal satu izin hanya bisa diubah dari Pengaturan perangkat. */
  canAskAgain: boolean;
};

type PermissionsState = {
  location: PermissionStatus;
  camera: PermissionStatus;
  /** Layar penjelasan izin sudah pernah ditampilkan di perangkat ini. */
  introSeen: boolean;
  ready: boolean;
  request: () => Promise<PermissionResult>;
  refresh: () => Promise<void>;
  markIntroSeen: () => Promise<void>;
};

const INTRO_KEY = 'ontrack.permissions_intro_seen';

const PermissionsContext = createContext<PermissionsState | null>(null);

function toStatus(permission: RawPermission | null): PermissionStatus {
  if (!permission) return 'undetermined';
  if (permission.granted) return 'granted';
  if (permission.status === 'undetermined') return 'undetermined';
  return permission.canAskAgain ? 'denied' : 'blocked';
}

const safely = async (read: () => Promise<RawPermission>) => {
  try {
    return await read();
  } catch {
    return null;
  }
};

/**
 * Status izin lokasi dan kamera (NFR-06). Izin hanya diminta lewat `request()`, setelah pengguna
 * membaca penjelasan; status dicek ulang setiap aplikasi kembali aktif (misalnya dari Pengaturan).
 */
export function PermissionsProvider({ children }: { children: ReactNode }) {
  const [location, setLocation] = useState<PermissionStatus>('undetermined');
  const [camera, setCamera] = useState<PermissionStatus>('undetermined');
  const [introSeen, setIntroSeen] = useState(false);
  const [ready, setReady] = useState(false);

  const refresh = useCallback(async () => {
    const [loc, cam] = await Promise.all([
      safely(Location.getForegroundPermissionsAsync),
      safely(Camera.getCameraPermissionsAsync),
    ]);
    setLocation(toStatus(loc));
    setCamera(toStatus(cam));
  }, []);

  useEffect(() => {
    void Promise.all([
      refresh(),
      SecureStore.getItemAsync(INTRO_KEY)
        .then((value) => setIntroSeen(value === '1'))
        .catch(() => setIntroSeen(false)),
    ]).finally(() => setReady(true));

    const subscription = AppState.addEventListener('change', (state) => {
      if (state === 'active') void refresh();
    });
    return () => subscription.remove();
  }, [refresh]);

  const request = useCallback(async (): Promise<PermissionResult> => {
    // Berurutan: dialog sistem lokasi lalu kamera. Izin yang sudah diberikan tidak memunculkan dialog.
    const loc = await safely(Location.requestForegroundPermissionsAsync);
    const cam = await safely(Camera.requestCameraPermissionsAsync);
    const locStatus = toStatus(loc);
    const camStatus = toStatus(cam);
    setLocation(locStatus);
    setCamera(camStatus);

    const missing: Permission[] = [];
    if (locStatus !== 'granted') missing.push('lokasi');
    if (camStatus !== 'granted') missing.push('kamera');
    return { granted: missing.length === 0, missing, canAskAgain: locStatus !== 'blocked' && camStatus !== 'blocked' };
  }, []);

  const markIntroSeen = useCallback(async () => {
    setIntroSeen(true);
    await SecureStore.setItemAsync(INTRO_KEY, '1').catch(() => undefined);
  }, []);

  const value = useMemo(
    () => ({ location, camera, introSeen, ready, request, refresh, markIntroSeen }),
    [location, camera, introSeen, ready, request, refresh, markIntroSeen],
  );
  return <PermissionsContext.Provider value={value}>{children}</PermissionsContext.Provider>;
}

export function usePermissions() {
  const context = useContext(PermissionsContext);
  if (!context) throw new Error('usePermissions harus dipakai di dalam <PermissionsProvider>.');
  return context;
}

export const statusLabels: Record<PermissionStatus, string> = {
  granted: 'Diizinkan',
  undetermined: 'Belum diminta',
  denied: 'Ditolak',
  blocked: 'Ditolak · ubah di Pengaturan',
};
