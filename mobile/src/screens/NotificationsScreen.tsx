import { useFocusEffect } from '@react-navigation/native';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useCallback, useState } from 'react';
import { ActivityIndicator, Linking, Pressable, RefreshControl, ScrollView, StyleSheet, Text, View } from 'react-native';
import { ApiError } from '../api/client';
import { listNotifications, markAllNotificationsRead, type AppNotification } from '../api/notifications';
import { Alert } from '../components/ui/Alert';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { formatDateTime } from '../lib/format';
import type { RootStackParamList } from '../navigation/types';
import { useNotifications } from '../notifications/NotificationsProvider';
import { useStyles, useTheme, type Theme } from '../theme/ThemeProvider';

type Props = NativeStackScreenProps<RootStackParamList, 'Notifications'>;

type State =
  | { kind: 'loading' }
  | { kind: 'ready'; items: AppNotification[]; page: number; totalPages: number }
  | { kind: 'unavailable' }
  | { kind: 'error'; message: string };

function toErrorState(error: unknown): State {
  if (error instanceof ApiError && (error.status === 404 || error.status === 405)) return { kind: 'unavailable' };
  if (error instanceof ApiError && error.status > 0 && error.status < 500) return { kind: 'error', message: error.message };
  return { kind: 'error', message: 'Notifikasi belum bisa dimuat. Periksa koneksi lalu coba lagi.' };
}

/** FR-15 · Daftar notifikasi in-app, misalnya pengajuan izin disetujui atau ditolak. */
export function NotificationsScreen(_props: Props) {
  const styles = useStyles(makeStyles);
  const { color } = useTheme();
  const notifications = useNotifications();
  const { setUnreadCount, open } = notifications;
  const [state, setState] = useState<State>({ kind: 'loading' });
  const [refreshing, setRefreshing] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [markingAll, setMarkingAll] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const { data, meta, unread_count } = await listNotifications();
      setState({ kind: 'ready', items: data, page: meta.page, totalPages: meta.total_pages });
      setUnreadCount(unread_count);
    } catch (error) {
      setState(toErrorState(error));
    }
  }, [setUnreadCount]);

  useFocusEffect(
    useCallback(() => {
      void load();
    }, [load]),
  );

  async function refresh() {
    setRefreshing(true);
    await load();
    setRefreshing(false);
  }

  async function loadMore() {
    if (state.kind !== 'ready') return;
    setLoadingMore(true);
    try {
      const { data, meta } = await listNotifications(state.page + 1);
      setState({ kind: 'ready', items: [...state.items, ...data], page: meta.page, totalPages: meta.total_pages });
    } catch (error) {
      setState(toErrorState(error));
    } finally {
      setLoadingMore(false);
    }
  }

  async function markAll() {
    if (state.kind !== 'ready') return;
    setMarkingAll(true);
    setActionError(null);
    try {
      await markAllNotificationsRead();
      const now = new Date().toISOString();
      setState({ ...state, items: state.items.map((item) => ({ ...item, read_at: item.read_at ?? now })) });
      setUnreadCount(0);
    } catch {
      setActionError('Belum bisa menandai semua notifikasi. Coba lagi.');
    } finally {
      setMarkingAll(false);
    }
  }

  function openItem(item: AppNotification) {
    if (state.kind === 'ready' && !item.read_at) {
      const now = new Date().toISOString();
      setState({ ...state, items: state.items.map((entry) => (entry.id === item.id ? { ...entry, read_at: now } : entry)) });
    }
    open({ notification_id: item.read_at ? undefined : item.id, leave_request_id: item.leave_request_id ?? undefined });
  }

  const hasUnread = state.kind === 'ready' && state.items.some((item) => !item.read_at);

  return (
    <ScrollView
      style={styles.screen}
      contentContainerStyle={styles.content}
      refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => void refresh()} />}
    >
      <PushStatus />

      {state.kind === 'loading' && (
        <View style={styles.center}>
          <ActivityIndicator color={color.text.primary} />
        </View>
      )}

      {state.kind === 'unavailable' && (
        <Alert tone="info" title="Notifikasi belum tersedia di server">
          Endpoint notifikasi belum aktif, jadi daftar ini masih kosong.
        </Alert>
      )}

      {state.kind === 'error' && (
        <Alert tone="danger" title="Notifikasi belum bisa dimuat" action={{ label: 'Coba lagi', onPress: () => void refresh() }}>
          {state.message}
        </Alert>
      )}

      {hasUnread && (
        <View style={styles.toolbar}>
          <Button variant="secondary" label="Tandai semua dibaca" loading={markingAll} onPress={() => void markAll()} />
        </View>
      )}

      {actionError && (
        <Alert tone="danger" title="Gagal" onDismiss={() => setActionError(null)}>
          {actionError}
        </Alert>
      )}

      {state.kind === 'ready' && state.items.length === 0 && (
        <Text style={styles.empty}>Belum ada notifikasi. Kabar pengajuan izinmu akan muncul di sini.</Text>
      )}

      {state.kind === 'ready' && state.items.length > 0 && (
        <Card>
          <View>
            {state.items.map((item, index) => {
              const unread = !item.read_at;
              return (
                <Pressable
                  key={item.id}
                  accessibilityRole="button"
                  accessibilityLabel={`${unread ? 'Belum dibaca. ' : ''}${item.title}. ${item.body}. ${formatDateTime(item.created_at)}`}
                  accessibilityHint={item.leave_request_id ? 'Buka detail pengajuan' : undefined}
                  onPress={() => openItem(item)}
                  style={({ pressed }) => [styles.row, index > 0 && styles.rowBorder, pressed && styles.pressed]}
                >
                  <View style={[styles.dot, unread && styles.dotUnread]} importantForAccessibility="no" />
                  <View style={styles.rowText}>
                    <Text style={[styles.title, unread && styles.titleUnread]}>{item.title}</Text>
                    <Text style={styles.body}>{item.body}</Text>
                    <Text style={styles.meta}>{formatDateTime(item.created_at)}</Text>
                  </View>
                </Pressable>
              );
            })}
          </View>
        </Card>
      )}

      {state.kind === 'ready' && state.page < state.totalPages && (
        <Button variant="secondary" fullWidth label="Muat lebih banyak" loading={loadingMore} onPress={() => void loadMore()} />
      )}
    </ScrollView>
  );
}

/** Keterangan bila push tidak bisa diterima di perangkat ini. Daftar in-app tetap berjalan. */
function PushStatus() {
  const { support, permission, requestPermission } = useNotifications();
  if (support === 'expo-go') {
    return (
      <Alert tone="info" title="Push notification tidak aktif di Expo Go">
        Notifikasi tetap tercatat di daftar ini. Push ke layar kunci butuh development build aplikasi.
      </Alert>
    );
  }
  if (support !== 'supported' || permission === 'granted') return null;
  if (permission === 'blocked') {
    return (
      <Alert tone="warning" title="Notifikasi dimatikan" action={{ label: 'Buka Pengaturan', onPress: () => void Linking.openSettings() }}>
        Izinkan notifikasi di Pengaturan agar kabar pengajuan izin langsung muncul di HP.
      </Alert>
    );
  }
  return (
    <Alert tone="warning" title="Notifikasi belum diizinkan" action={{ label: 'Izinkan', onPress: () => void requestPermission() }}>
      Tanpa izin ini, kabar pengajuan izin hanya terlihat saat kamu membuka aplikasi.
    </Alert>
  );
}

const makeStyles = ({ color, radius, size, space, text }: Theme) =>
  StyleSheet.create({
    screen: { flex: 1, backgroundColor: color.bg.canvas },
    content: { gap: space[16], padding: space[20], paddingBottom: space[48] },
    center: { paddingVertical: space[32], alignItems: 'center' },
    toolbar: { flexDirection: 'row', justifyContent: 'flex-end' },
    empty: { ...text.bodyM, color: color.text.secondary, textAlign: 'center', paddingVertical: space[24] },
    row: { flexDirection: 'row', alignItems: 'flex-start', gap: space[12], minHeight: size.iconTarget + space[16], paddingVertical: space[12] },
    rowBorder: { borderTopWidth: 1, borderTopColor: color.border.default },
    pressed: { backgroundColor: color.bg.hover },
    dot: { width: 8, height: 8, marginTop: 7, borderRadius: radius.full, backgroundColor: 'transparent' },
    dotUnread: { backgroundColor: color.text.primary },
    rowText: { flex: 1, gap: space[2] },
    title: { ...text.bodyM, color: color.text.primary },
    titleUnread: { ...text.labelM, fontSize: text.bodyM.fontSize },
    body: { ...text.bodyS, color: color.text.secondary },
    meta: { ...text.caption, color: color.text.secondary },
  });
