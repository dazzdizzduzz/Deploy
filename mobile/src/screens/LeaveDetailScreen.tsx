import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useCallback, useEffect, useState } from 'react';
import { ActivityIndicator, Linking, RefreshControl, ScrollView, StyleSheet, Text, View } from 'react-native';
import { ApiError } from '../api/client';
import { getLeaveRequest, type LeaveRequest } from '../api/leave';
import { Alert } from '../components/ui/Alert';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { formatRange, leaveStatusBadge, leaveTypeLabels } from '../leave/labels';
import { formatDateTime } from '../lib/format';
import type { RootStackParamList } from '../navigation/types';
import { useStyles, useTheme, type Theme } from '../theme/ThemeProvider';

type Props = NativeStackScreenProps<RootStackParamList, 'LeaveDetail'>;

/** FR-12 · Detail satu pengajuan: jenis, tanggal, alasan, lampiran, dan keputusan atasan. */
export function LeaveDetailScreen({ route }: Props) {
  const styles = useStyles(makeStyles);
  const { color } = useTheme();
  const { id, request: initial, created } = route.params;
  const [request, setRequest] = useState<LeaveRequest | null>(initial ?? null);
  const [error, setError] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const [showCreated, setShowCreated] = useState(Boolean(created));

  const load = useCallback(async () => {
    try {
      setRequest(await getLeaveRequest(id));
      setError(null);
    } catch (failure) {
      // Data dari daftar tetap ditampilkan bila detail gagal dimuat.
      if (failure instanceof ApiError && (failure.status === 404 || failure.status === 405) && initial) return;
      setError(
        failure instanceof ApiError && failure.status > 0 && failure.status < 500
          ? failure.message
          : 'Detail pengajuan belum bisa dimuat. Periksa koneksi lalu coba lagi.',
      );
    }
  }, [id, initial]);

  useEffect(() => {
    void load();
  }, [load]);

  async function refresh() {
    setRefreshing(true);
    await load();
    setRefreshing(false);
  }

  if (!request) {
    return (
      <View style={styles.center}>
        {error ? (
          <Alert tone="danger" title="Pengajuan tidak dapat dibuka" action={{ label: 'Coba lagi', onPress: () => void refresh() }}>
            {error}
          </Alert>
        ) : (
          <ActivityIndicator color={color.text.primary} />
        )}
      </View>
    );
  }

  const badge = leaveStatusBadge(request.status);
  const downloadUrl = request.attachment?.download_url;

  return (
    <ScrollView
      style={styles.screen}
      contentContainerStyle={styles.content}
      refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => void refresh()} />}
    >
      {showCreated && (
        <Alert tone="success" title="Pengajuan terkirim" onDismiss={() => setShowCreated(false)}>
          {request.approver_name
            ? `Menunggu persetujuan ${request.approver_name}.`
            : 'Menunggu persetujuan atasan atau HRD.'}
        </Alert>
      )}

      <View style={styles.header}>
        <Text accessibilityRole="header" style={styles.title}>
          {leaveTypeLabels[request.type]}
        </Text>
        <Badge tone={badge.tone}>{badge.label}</Badge>
      </View>

      <Card>
        <Row label="Tanggal" value={formatRange(request.start_date, request.end_date)} />
        <Row label="Lama" value={`${request.working_days} hari kerja`} />
        <Row label="Alasan" value={request.reason} />
        <Row label="Diajukan" value={formatDateTime(request.created_at)} />
        {request.approver_name && <Row label="Ditujukan ke" value={request.approver_name} />}
      </Card>

      {request.attachment && (
        <Card title="Surat dokter">
          <Text style={styles.value} numberOfLines={1}>
            {request.attachment.file_name}
          </Text>
          {downloadUrl ? (
            <Button variant="secondary" label="Buka lampiran" onPress={() => void Linking.openURL(downloadUrl)} />
          ) : (
            <Text style={styles.meta}>Tautan lampiran belum tersedia.</Text>
          )}
        </Card>
      )}

      {request.status !== 'menunggu' && (
        <Card title={request.status === 'ditolak' ? 'Alasan penolakan' : 'Keputusan'}>
          <Text style={styles.value}>
            {request.decision_note || (request.status === 'ditolak' ? 'Tidak ada alasan tercatat.' : 'Disetujui.')}
          </Text>
          {request.decided_at && <Text style={styles.meta}>Diputuskan {formatDateTime(request.decided_at)}</Text>}
        </Card>
      )}

      {error && (
        <Alert tone="warning" title="Data mungkin belum terbaru" action={{ label: 'Coba lagi', onPress: () => void refresh() }}>
          {error}
        </Alert>
      )}
    </ScrollView>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  const styles = useStyles(makeStyles);
  return (
    <View style={styles.row} accessible accessibilityLabel={`${label}: ${value}`}>
      <Text style={styles.meta}>{label}</Text>
      <Text style={styles.value}>{value}</Text>
    </View>
  );
}

const makeStyles = ({ color, space, text }: Theme) =>
  StyleSheet.create({
    screen: { flex: 1, backgroundColor: color.bg.canvas },
    content: { gap: space[16], padding: space[20], paddingBottom: space[48] },
    center: { flex: 1, justifyContent: 'center', padding: space[20], backgroundColor: color.bg.canvas },
    header: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', gap: space[12] },
    title: { ...text.headingH2, color: color.text.primary, flexShrink: 1 },
    row: { gap: space[2], paddingVertical: space[4] },
    value: { ...text.bodyM, color: color.text.primary },
    meta: { ...text.caption, color: color.text.secondary },
  });
