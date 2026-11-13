import * as Notifications from 'expo-notifications';
import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { AppState } from 'react-native';
import { ApiError } from '../api/client';
import { getUnreadCount, markNotificationRead, registerDevice, unregisterDevice } from '../api/notifications';
import { addSignOutTask } from '../auth/signOutTasks';
import { navigationRef } from '../navigation/navigationRef';
import {
  configureNotifications,
  deviceName,
  getDeviceToken,
  pushSupport,
  readPushData,
  readPushPermission,
  requestPushPermission,
  type PushData,
  type PushPermission,
  type PushSupport,
} from './push';

type NotificationsState = {
  /** null selama belum diketahui atau endpoint belum aktif. */
  unreadCount: number | null;
  /** false bila endpoint notifikasi belum ada di server. */
  available: boolean;
  support: PushSupport;
  permission: PushPermission;
  refreshUnread: () => Promise<void>;
  setUnreadCount: (count: number) => void;
  /** Membuka tujuan notifikasi (Detail Pengajuan) dan menandainya sudah dibaca. */
  open: (target: PushData) => void;
  requestPermission: () => Promise<void>;
};

const NotificationsContext = createContext<NotificationsState | null>(null);

const missingEndpoint = (error: unknown) => error instanceof ApiError && (error.status === 404 || error.status === 405);

/** Tunggu navigator siap (misalnya saat aplikasi dibuka dari notifikasi), maksimal 5 detik. */
function navigateWhenReady(go: () => void, attempt = 0) {
  if (navigationRef.isReady()) go();
  else if (attempt < 25) setTimeout(() => navigateWhenReady(go, attempt + 1), 200);
}

/**
 * FR-15 · Jumlah notifikasi belum dibaca, pendaftaran token push ke server, dan penanganan tap
 * notifikasi. Dipasang hanya setelah login.
 */
export function NotificationsProvider({ children }: { children: ReactNode }) {
  const support = useMemo(pushSupport, []);
  const [unreadCount, setUnreadCount] = useState<number | null>(null);
  const [available, setAvailable] = useState(true);
  const [permission, setPermission] = useState<PushPermission>('undetermined');
  const tokenRef = useRef<string | null>(null);

  const refreshUnread = useCallback(async () => {
    try {
      const { unread_count } = await getUnreadCount();
      setUnreadCount(unread_count);
      setAvailable(true);
    } catch (error) {
      if (missingEndpoint(error)) setAvailable(false);
    }
  }, []);

  const register = useCallback(async () => {
    const device = await getDeviceToken();
    if (!device || device.token === tokenRef.current) return;
    try {
      await registerDevice({ ...device, device_name: deviceName() });
      tokenRef.current = device.token;
    } catch {
      // Endpoint belum ada atau jaringan putus: dicoba lagi saat aplikasi aktif kembali.
    }
  }, []);

  const open = useCallback(
    (target: PushData) => {
      if (target.notification_id) {
        void markNotificationRead(target.notification_id)
          .then(refreshUnread)
          .catch(() => undefined);
      }
      const leaveId = target.leave_request_id;
      if (leaveId) navigateWhenReady(() => navigationRef.navigate('LeaveDetail', { id: leaveId }));
    },
    [refreshUnread],
  );

  const requestPermission = useCallback(async () => {
    const next = await requestPushPermission();
    setPermission(next);
    if (next === 'granted') await register();
  }, [register]);

  // Izin dan token: minta izin sekali setelah login, lalu daftarkan token ke server.
  useEffect(() => {
    if (support !== 'supported') return;
    let active = true;
    void (async () => {
      await configureNotifications();
      let current = await readPushPermission();
      if (current === 'undetermined') current = await requestPushPermission();
      if (!active) return;
      setPermission(current);
      if (current === 'granted') await register();
    })();
    const tokenListener = Notifications.addPushTokenListener(() => {
      tokenRef.current = null;
      void register();
    });
    return () => {
      active = false;
      tokenListener.remove();
    };
  }, [support, register]);

  // Saat logout, hapus token perangkat di server agar push akun ini berhenti.
  useEffect(
    () =>
      addSignOutTask(async () => {
        if (tokenRef.current) await unregisterDevice(tokenRef.current);
      }),
    [],
  );

  // Jumlah belum dibaca: saat login, saat aplikasi aktif kembali, dan saat notifikasi masuk.
  useEffect(() => {
    void refreshUnread();
    const appState = AppState.addEventListener('change', (state) => {
      if (state !== 'active') return;
      void refreshUnread();
      if (support === 'supported') {
        void readPushPermission().then((next) => {
          setPermission(next);
          if (next === 'granted') void register();
        });
      }
    });
    if (support === 'web') return () => appState.remove();
    const received = Notifications.addNotificationReceivedListener(() => void refreshUnread());
    return () => {
      appState.remove();
      received.remove();
    };
  }, [refreshUnread, register, support]);

  // Tap notifikasi: saat aplikasi berjalan, dan saat aplikasi dibuka dari notifikasi.
  useEffect(() => {
    if (support === 'web') return;
    const handled = new Set<string>();
    const handle = (response: Notifications.NotificationResponse | null) => {
      const data = readPushData(response);
      const key = response?.notification.request.identifier;
      if (!data || !key || handled.has(key)) return;
      handled.add(key);
      open(data);
    };
    handle(Notifications.getLastNotificationResponse());
    const subscription = Notifications.addNotificationResponseReceivedListener(handle);
    return () => subscription.remove();
  }, [open, support]);

  const value = useMemo(
    () => ({ unreadCount, available, support, permission, refreshUnread, setUnreadCount, open, requestPermission }),
    [unreadCount, available, support, permission, refreshUnread, open, requestPermission],
  );
  return <NotificationsContext.Provider value={value}>{children}</NotificationsContext.Provider>;
}

export function useNotifications() {
  const context = useContext(NotificationsContext);
  if (!context) throw new Error('useNotifications harus dipakai di dalam <NotificationsProvider>.');
  return context;
}
