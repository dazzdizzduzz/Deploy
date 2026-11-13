import Constants, { ExecutionEnvironment } from 'expo-constants';
import * as Device from 'expo-device';
import * as Notifications from 'expo-notifications';
import { Platform } from 'react-native';

/**
 * `expo-go`: Expo Go (SDK 53+) tidak menerima push dari server; perlu development build.
 * `simulator`: emulator/simulator tidak punya token push.
 */
export type PushSupport = 'supported' | 'expo-go' | 'simulator' | 'web';

export type PushPermission = 'granted' | 'undetermined' | 'denied' | 'blocked';

export const CHANNEL_ID = 'status-pengajuan';

export function pushSupport(): PushSupport {
  if (Platform.OS === 'web') return 'web';
  if (Constants.executionEnvironment === ExecutionEnvironment.StoreClient) return 'expo-go';
  if (!Device.isDevice) return 'simulator';
  return 'supported';
}

let configured = false;

/** Notifikasi yang datang saat aplikasi terbuka tetap tampil sebagai banner. */
export async function configureNotifications() {
  if (configured || Platform.OS === 'web') return;
  configured = true;
  Notifications.setNotificationHandler({
    handleNotification: async () => ({
      shouldShowBanner: true,
      shouldShowList: true,
      shouldPlaySound: true,
      shouldSetBadge: false,
    }),
  });
  if (Platform.OS === 'android') {
    // Android 8+ wajib punya channel; nama ini yang tampil di Pengaturan notifikasi aplikasi.
    await Notifications.setNotificationChannelAsync(CHANNEL_ID, {
      name: 'Status pengajuan',
      description: 'Pengajuan izin, sakit, dan cuti disetujui atau ditolak',
      importance: Notifications.AndroidImportance.HIGH,
    }).catch(() => undefined);
  }
}

function toPermission(result: Notifications.NotificationPermissionsStatus): PushPermission {
  if (result.granted) return 'granted';
  if (result.status === 'undetermined') return 'undetermined';
  return result.canAskAgain ? 'denied' : 'blocked';
}

export async function readPushPermission(): Promise<PushPermission> {
  try {
    return toPermission(await Notifications.getPermissionsAsync());
  } catch {
    return 'undetermined';
  }
}

export async function requestPushPermission(): Promise<PushPermission> {
  try {
    return toPermission(await Notifications.requestPermissionsAsync());
  } catch {
    return 'denied';
  }
}

/** Token FCM (Android) atau APNs (iOS) untuk dikirim ke backend; null bila belum bisa didapat. */
export async function getDeviceToken(): Promise<{ token: string; platform: 'android' | 'ios' } | null> {
  if (pushSupport() !== 'supported') return null;
  try {
    const { data, type } = await Notifications.getDevicePushTokenAsync();
    if (typeof data !== 'string' || (type !== 'android' && type !== 'ios')) return null;
    return { token: data, platform: type };
  } catch {
    // Build tanpa google-services.json / kredensial APNs gagal di sini.
    return null;
  }
}

export const deviceName = () => Device.modelName ?? undefined;

/** Isi `data` push (draf PushData); semua nilai string. */
export type PushData = { notification_id?: string; type?: string; leave_request_id?: string };

export function readPushData(response: Notifications.NotificationResponse | null): PushData | null {
  const data = response?.notification.request.content.data as Record<string, unknown> | undefined;
  if (!data) return null;
  const pick = (key: string) => (typeof data[key] === 'string' ? (data[key] as string) : undefined);
  return { notification_id: pick('notification_id'), type: pick('type'), leave_request_id: pick('leave_request_id') };
}
