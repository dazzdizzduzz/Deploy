import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useCallback, useEffect, useState } from 'react';
import { ActivityIndicator, Pressable, RefreshControl, ScrollView, StyleSheet, Text, View } from 'react-native';
import { ApiError } from '../api/client';
import { getMonthlyAttendance, type AttendanceDay, type MonthlyAttendance } from '../api/history';
import { useAttendance } from '../attendance/AttendanceProvider';
import { dayStatusBadge, formatDuration, workedMinutes } from '../attendance/labels';
import { currentMonth, monthLabel, shiftMonth } from '../attendance/month';
import { Alert } from '../components/ui/Alert';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { formatLongDate, formatTime } from '../lib/format';
import type { RootStackParamList } from '../navigation/types';
import { useStyles, useTheme, type Theme } from '../theme/ThemeProvider';

type Props = NativeStackScreenProps<RootStackParamList, 'AttendanceHistory'>;

type State =
  | { kind: 'loading' }
  | { kind: 'ready'; data: MonthlyAttendance }
  | { kind: 'unavailable' }
  | { kind: 'error'; message: string };

/** Satu baris ringkas: jam masuk–pulang dan total jam kerja, atau jenis izin. */
function dayLine(day: AttendanceDay) {
  const record = day.attendance;
  if (record) {
    const worked = workedMinutes(record);
    const out = record.clock_out ? formatTime(record.clock_out) : 'belum clock out';
    return `${formatTime(record.clock_in)} – ${out}${worked !== null ? ` · ${formatDuration(worked)}` : ''}`;
  }
  if (day.leave?.reason) return day.leave.reason;
  if (day.status === 'alpha') return 'Tidak ada absensi maupun izin';
  return '';
}

/** FR-10 · Riwayat absensi pribadi per bulan dengan filter bulan. */
export function AttendanceHistoryScreen({ navigation }: Props) {
  const styles = useStyles(makeStyles);
  const { color } = useTheme();
  const { today } = useAttendance();
  const thisMonth = currentMonth();
  const [month, setMonth] = useState(thisMonth);
  const [state, setState] = useState<State>({ kind: 'loading' });
  const [refreshing, setRefreshing] = useState(false);

  const load = useCallback(async (target: string) => {
    try {
      const data = await getMonthlyAttendance(target);
      setState({ kind: 'ready', data });
    } catch (error) {
      if (error instanceof ApiError && (error.status === 404 || error.status === 405)) setState({ kind: 'unavailable' });
      else if (error instanceof ApiError && error.status > 0 && error.status < 500) setState({ kind: 'error', message: error.message });
      else setState({ kind: 'error', message: 'Riwayat belum bisa dimuat. Periksa koneksi lalu coba lagi.' });
    }
  }, []);

  useEffect(() => {
    setState({ kind: 'loading' });
    void load(month);
  }, [month, load]);

  async function refresh() {
    setRefreshing(true);
    await load(month);
    setRefreshing(false);
  }

  // Selama endpoint riwayat belum ada, tampilkan presensi hari ini yang tersimpan di perangkat.
  const localDays: AttendanceDay[] =
    state.kind === 'unavailable' && today && today.work_date.startsWith(month)
      ? [{ date: today.work_date, status: today.status === 'absent' ? 'alpha' : today.status, attendance: today }]
      : [];
  const days = state.kind === 'ready' ? state.data.days : localDays;
  const summary = state.kind === 'ready' ? state.data.summary : null;

  return (
    <ScrollView
      style={styles.screen}
      contentContainerStyle={styles.content}
      refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => void refresh()} />}
    >
      <View style={styles.monthBar} accessibilityRole="toolbar">
        <Button variant="secondary" label="‹ Sebelumnya" onPress={() => setMonth(shiftMonth(month, -1))} />
        <Text accessibilityRole="header" accessibilityLiveRegion="polite" style={styles.month}>
          {monthLabel(month)}
        </Text>
        <Button
          variant="secondary"
          label="Berikutnya ›"
          disabled={month >= thisMonth}
          onPress={() => setMonth(shiftMonth(month, 1))}
        />
      </View>

      {state.kind === 'loading' && (
        <View style={styles.center}>
          <ActivityIndicator color={color.text.primary} />
        </View>
      )}

      {state.kind === 'error' && (
        <Alert tone="danger" title="Riwayat belum bisa dimuat" action={{ label: 'Coba lagi', onPress: () => void refresh() }}>
          {state.message}
        </Alert>
      )}

      {state.kind === 'unavailable' && (
        <Alert tone="info" title="Riwayat belum tersedia di server">
          Endpoint riwayat absensi belum aktif.
          {localDays.length > 0 ? ' Di bawah ini presensi hari ini yang tersimpan di perangkat.' : ''}
        </Alert>
      )}

      {summary && (
        <Card title="Ringkasan bulan">
          <View style={styles.grid}>
            {(
              [
                ['Hadir', summary.hadir],
                ['Terlambat', summary.terlambat],
                ['Izin', summary.izin],
                ['Cuti', summary.cuti],
                ['Sakit', summary.sakit],
                ['Alpha', summary.alpha],
              ] as const
            ).map(([label, value]) => (
              <View key={label} accessible accessibilityLabel={`${label} ${value} hari`} style={styles.stat}>
                <Text style={styles.statValue}>{value}</Text>
                <Text style={styles.statLabel}>{label}</Text>
              </View>
            ))}
          </View>
          <View style={styles.totals}>
            <Text style={styles.totalText}>Total jam kerja: {formatDuration(summary.total_work_minutes)}</Text>
            <Text style={styles.totalText}>Lembur: {formatDuration(summary.overtime_minutes)}</Text>
          </View>
        </Card>
      )}

      {state.kind === 'ready' && days.length === 0 && (
        <Text style={styles.empty}>Belum ada absensi di {monthLabel(month)}.</Text>
      )}

      {days.length > 0 && (
        <Card title="Per hari">
          <View>
            {days.map((day, index) => {
              const badge = dayStatusBadge(day.status);
              const line = dayLine(day);
              return (
                <Pressable
                  key={day.date}
                  accessibilityRole="button"
                  accessibilityLabel={`${formatLongDate(day.date)}, ${badge.label}${line ? `, ${line}` : ''}. Buka detail.`}
                  onPress={() => navigation.navigate('AttendanceDay', { day })}
                  style={({ pressed }) => [styles.row, index > 0 && styles.rowBorder, pressed && styles.pressed]}
                >
                  <View style={styles.rowText}>
                    <Text style={styles.rowDate}>{formatLongDate(day.date)}</Text>
                    {line ? <Text style={styles.rowMeta}>{line}</Text> : null}
                  </View>
                  <Badge tone={badge.tone}>{badge.label}</Badge>
                </Pressable>
              );
            })}
          </View>
        </Card>
      )}
    </ScrollView>
  );
}

const makeStyles = ({ color, size, space, text }: Theme) =>
  StyleSheet.create({
    screen: { flex: 1, backgroundColor: color.bg.canvas },
    content: { gap: space[16], padding: space[20], paddingBottom: space[48] },
    monthBar: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', gap: space[8] },
    month: { ...text.headingH3, color: color.text.primary, flexShrink: 1, textAlign: 'center' },
    center: { paddingVertical: space[32], alignItems: 'center' },
    grid: { flexDirection: 'row', flexWrap: 'wrap', rowGap: space[12] },
    stat: { width: '33.33%', gap: space[2] },
    statValue: { ...text.headingH2, color: color.text.primary },
    statLabel: { ...text.caption, color: color.text.secondary },
    totals: { gap: space[4], paddingTop: space[12], borderTopWidth: 1, borderTopColor: color.border.default },
    totalText: { ...text.bodyS, color: color.text.primary },
    empty: { ...text.bodyM, color: color.text.secondary, textAlign: 'center', paddingVertical: space[24] },
    row: { flexDirection: 'row', alignItems: 'center', gap: space[12], minHeight: size.iconTarget + space[16], paddingVertical: space[8] },
    rowBorder: { borderTopWidth: 1, borderTopColor: color.border.default },
    pressed: { backgroundColor: color.bg.hover },
    rowText: { flex: 1, gap: space[2] },
    rowDate: { ...text.labelM, color: color.text.primary },
    rowMeta: { ...text.caption, color: color.text.secondary },
  });
