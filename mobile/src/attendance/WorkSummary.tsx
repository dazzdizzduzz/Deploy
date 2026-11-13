import { StyleSheet, Text, View } from 'react-native';
import type { Attendance } from '../api/attendance';
import { Badge } from '../components/ui/Badge';
import { formatTime, timeZoneLabel } from '../lib/format';
import { useStyles, type Theme } from '../theme/ThemeProvider';
import { formatDuration, statusLabel, workedMinutes } from './labels';

/** FR-07 · Ringkasan jam kerja hari ini: masuk, pulang, total, lembur, pulang cepat, terlambat. */
export function WorkSummary({ record }: { record: Attendance }) {
  const styles = useStyles(makeStyles);
  const worked = workedMinutes(record);
  const badge = statusLabel(record);
  const zone = timeZoneLabel();

  return (
    <View style={styles.wrap}>
      <Badge tone={badge.tone}>{badge.label}</Badge>
      <View style={styles.times}>
        <View style={styles.time}>
          <Text style={styles.label}>Masuk</Text>
          <Text style={styles.clock}>{formatTime(record.clock_in)}</Text>
        </View>
        <View style={styles.time}>
          <Text style={styles.label}>Pulang</Text>
          <Text style={styles.clock}>{record.clock_out ? formatTime(record.clock_out) : '—'}</Text>
        </View>
      </View>
      <Text style={styles.zone}>Waktu {zone}</Text>
      <View style={styles.list}>
        <Row label="Total jam kerja" value={worked === null ? '—' : formatDuration(worked)} strong />
        <Row label="Lembur" value={formatDuration(record.overtime_minutes)} accent={record.overtime_minutes > 0} />
        {record.early_leave_minutes > 0 && <Row label="Pulang cepat" value={formatDuration(record.early_leave_minutes)} />}
        {record.late_minutes > 0 && <Row label="Terlambat" value={formatDuration(record.late_minutes)} />}
      </View>
    </View>
  );
}

function Row({ label, value, strong, accent }: { label: string; value: string; strong?: boolean; accent?: boolean }) {
  const styles = useStyles(makeStyles);
  return (
    <View accessible style={styles.row}>
      <Text style={styles.rowLabel}>{label}</Text>
      <Text style={[styles.rowValue, strong && styles.strong, accent && styles.accent]}>{value}</Text>
    </View>
  );
}

const makeStyles = ({ color, space, text }: Theme) =>
  StyleSheet.create({
    wrap: { gap: space[12], alignItems: 'stretch' },
    times: { flexDirection: 'row', gap: space[16] },
    time: { flex: 1, gap: space[2] },
    label: { ...text.caption, color: color.text.secondary },
    clock: { ...text.numericKpi, color: color.text.primary },
    zone: { ...text.caption, color: color.text.secondary, marginTop: -space[8] },
    list: { borderTopWidth: 1, borderTopColor: color.border.default },
    row: {
      flexDirection: 'row',
      justifyContent: 'space-between',
      gap: space[12],
      paddingVertical: space[8],
      borderBottomWidth: 1,
      borderBottomColor: color.border.default,
    },
    rowLabel: { ...text.bodyS, color: color.text.secondary },
    rowValue: { ...text.bodyS, color: color.text.primary },
    strong: { ...text.labelM },
    accent: { ...text.labelM, color: color.text.accent },
  });
