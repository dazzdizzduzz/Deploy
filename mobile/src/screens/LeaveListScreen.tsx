import { useFocusEffect } from '@react-navigation/native';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useCallback, useState } from 'react';
import { ActivityIndicator, Pressable, RefreshControl, ScrollView, StyleSheet, Text, View } from 'react-native';
import { ApiError } from '../api/client';
import { getLeaveQuota, listLeaveRequests, type LeaveQuota, type LeaveRequest, type LeaveStatus } from '../api/leave';
import { Alert } from '../components/ui/Alert';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { formatRange, leaveStatusBadge, leaveTypeLabels } from '../leave/labels';
import type { RootStackParamList } from '../navigation/types';
import { useStyles, useTheme, type Theme } from '../theme/ThemeProvider';

type Props = NativeStackScreenProps<RootStackParamList, 'LeaveRequests'>;

type ListState =
  | { kind: 'loading' }
  | { kind: 'ready'; items: LeaveRequest[]; page: number; totalPages: number }
  | { kind: 'unavailable' }
  | { kind: 'error'; message: string };

const filters: { value: LeaveStatus | null; label: string }[] = [
  { value: null, label: 'Semua' },
  { value: 'menunggu', label: 'Menunggu' },
  { value: 'disetujui', label: 'Disetujui' },
  { value: 'ditolak', label: 'Ditolak' },
];

const isMissingEndpoint = (error: unknown) => error instanceof ApiError && (error.status === 404 || error.status === 405);

function listError(error: unknown): ListState {
  if (isMissingEndpoint(error)) return { kind: 'unavailable' };
  if (error instanceof ApiError && error.status > 0 && error.status < 500) return { kind: 'error', message: error.message };
  return { kind: 'error', message: 'Daftar pengajuan belum bisa dimuat. Periksa koneksi lalu coba lagi.' };
}

/** FR-12 · Daftar pengajuan izin, sakit, dan cuti milik karyawan beserta statusnya. */
export function LeaveListScreen({ navigation }: Props) {
  const styles = useStyles(makeStyles);
  const { color } = useTheme();
  const [status, setStatus] = useState<LeaveStatus | null>(null);
  const [list, setList] = useState<ListState>({ kind: 'loading' });
  const [quota, setQuota] = useState<LeaveQuota | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);

  const load = useCallback(async (filter: LeaveStatus | null) => {
    const [listResult, quotaResult] = await Promise.allSettled([listLeaveRequests(filter ?? undefined), getLeaveQuota()]);
    if (listResult.status === 'fulfilled') {
      const { data, meta } = listResult.value;
      setList({ kind: 'ready', items: data, page: meta.page, totalPages: meta.total_pages });
    } else {
      setList(listError(listResult.reason));
    }
    // Kuota hanya pelengkap: bila gagal dimuat, kartunya tidak ditampilkan.
    setQuota(quotaResult.status === 'fulfilled' ? quotaResult.value : null);
  }, []);

  // Muat ulang setiap kali layar tampil, termasuk setelah kembali dari form pengajuan.
  useFocusEffect(
    useCallback(() => {
      void load(status);
    }, [load, status]),
  );

  function changeFilter(next: LeaveStatus | null) {
    if (next === status) return;
    setList({ kind: 'loading' });
    setStatus(next);
  }

  async function refresh() {
    setRefreshing(true);
    await load(status);
    setRefreshing(false);
  }

  async function loadMore() {
    if (list.kind !== 'ready') return;
    setLoadingMore(true);
    try {
      const { data, meta } = await listLeaveRequests(status ?? undefined, list.page + 1);
      setList({ kind: 'ready', items: [...list.items, ...data], page: meta.page, totalPages: meta.total_pages });
    } catch (error) {
      setList(listError(error));
    } finally {
      setLoadingMore(false);
    }
  }

  const activeLabel = filters.find((filter) => filter.value === status)?.label.toLowerCase();

  return (
    <ScrollView
      style={styles.screen}
      contentContainerStyle={styles.content}
      refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => void refresh()} />}
    >
      <Button fullWidth label="Ajukan Izin" onPress={() => navigation.navigate('LeaveForm')} />

      {quota && (
        <Card title={`Kuota cuti ${quota.year}`}>
          <View style={styles.quota}>
            <View accessible accessibilityLabel={`Sisa ${quota.remaining_days} dari ${quota.total_days} hari`} style={styles.quotaMain}>
              <Text style={styles.quotaValue}>{quota.remaining_days}</Text>
              <Text style={styles.quotaLabel}>sisa dari {quota.total_days} hari</Text>
            </View>
            <View style={styles.quotaSide}>
              <Text style={styles.meta}>Terpakai {quota.used_days} hari</Text>
              <Text style={styles.meta}>Menunggu persetujuan {quota.pending_days} hari</Text>
            </View>
          </View>
        </Card>
      )}

      <View style={styles.filters} accessibilityRole="radiogroup" accessibilityLabel="Filter status">
        {filters.map((filter) => {
          const active = filter.value === status;
          return (
            <Pressable
              key={filter.label}
              accessibilityRole="radio"
              accessibilityState={{ checked: active }}
              onPress={() => changeFilter(filter.value)}
              style={({ pressed }) => [styles.chip, active && styles.chipActive, pressed && !active && styles.pressed]}
            >
              <Text style={[styles.chipLabel, active && styles.chipLabelActive]}>{filter.label}</Text>
            </Pressable>
          );
        })}
      </View>

      {list.kind === 'loading' && (
        <View style={styles.center}>
          <ActivityIndicator color={color.text.primary} />
        </View>
      )}

      {list.kind === 'unavailable' && (
        <Alert tone="info" title="Pengajuan izin belum tersedia di server">
          Endpoint izin dan cuti belum aktif, jadi daftar pengajuan belum bisa ditampilkan.
        </Alert>
      )}

      {list.kind === 'error' && (
        <Alert tone="danger" title="Daftar belum bisa dimuat" action={{ label: 'Coba lagi', onPress: () => void refresh() }}>
          {list.message}
        </Alert>
      )}

      {list.kind === 'ready' && list.items.length === 0 && (
        <Text style={styles.empty}>
          {status ? `Tidak ada pengajuan berstatus ${activeLabel}.` : 'Belum ada pengajuan. Tekan Ajukan Izin untuk membuat yang pertama.'}
        </Text>
      )}

      {list.kind === 'ready' && list.items.length > 0 && (
        <Card>
          <View>
            {list.items.map((item, index) => {
              const badge = leaveStatusBadge(item.status);
              const range = formatRange(item.start_date, item.end_date);
              return (
                <Pressable
                  key={item.id}
                  accessibilityRole="button"
                  accessibilityLabel={`${leaveTypeLabels[item.type]}, ${range}, ${item.working_days} hari kerja, ${badge.label}. Buka detail.`}
                  onPress={() => navigation.navigate('LeaveDetail', { id: item.id, request: item })}
                  style={({ pressed }) => [styles.row, index > 0 && styles.rowBorder, pressed && styles.pressed]}
                >
                  <View style={styles.rowText}>
                    <Text style={styles.rowTitle}>{leaveTypeLabels[item.type]}</Text>
                    <Text style={styles.meta}>
                      {range} · {item.working_days} hari kerja
                    </Text>
                  </View>
                  <Badge tone={badge.tone}>{badge.label}</Badge>
                </Pressable>
              );
            })}
          </View>
        </Card>
      )}

      {list.kind === 'ready' && list.page < list.totalPages && (
        <Button variant="secondary" fullWidth label="Muat lebih banyak" loading={loadingMore} onPress={() => void loadMore()} />
      )}
    </ScrollView>
  );
}

const makeStyles = ({ color, radius, size, space, text }: Theme) =>
  StyleSheet.create({
    screen: { flex: 1, backgroundColor: color.bg.canvas },
    content: { gap: space[16], padding: space[20], paddingBottom: space[48] },
    quota: { flexDirection: 'row', alignItems: 'center', gap: space[16] },
    quotaMain: { gap: space[2] },
    quotaValue: { ...text.headingH1, color: color.text.primary },
    quotaLabel: { ...text.caption, color: color.text.secondary },
    quotaSide: { flex: 1, gap: space[4] },
    filters: { flexDirection: 'row', flexWrap: 'wrap', gap: space[8] },
    chip: {
      minHeight: size.iconTarget,
      justifyContent: 'center',
      paddingHorizontal: space[16],
      borderWidth: 1,
      borderColor: color.border.default,
      borderRadius: radius.full,
      backgroundColor: color.bg.surface,
    },
    chipActive: { borderColor: color.bg.inverse, backgroundColor: color.bg.inverse },
    chipLabel: { ...text.labelM, color: color.text.primary },
    chipLabelActive: { color: color.text.inverse },
    center: { paddingVertical: space[32], alignItems: 'center' },
    empty: { ...text.bodyM, color: color.text.secondary, textAlign: 'center', paddingVertical: space[24] },
    row: { flexDirection: 'row', alignItems: 'center', gap: space[12], minHeight: size.iconTarget + space[16], paddingVertical: space[8] },
    rowBorder: { borderTopWidth: 1, borderTopColor: color.border.default },
    pressed: { backgroundColor: color.bg.hover },
    rowText: { flex: 1, gap: space[2] },
    rowTitle: { ...text.labelM, color: color.text.primary },
    meta: { ...text.caption, color: color.text.secondary },
  });
