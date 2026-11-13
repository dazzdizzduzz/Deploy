import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { ScrollView, StyleSheet, Text, View } from 'react-native';
import { dayStatusBadge } from '../attendance/labels';
import { WorkSummary } from '../attendance/WorkSummary';
import { Badge } from '../components/ui/Badge';
import { Card } from '../components/ui/Card';
import { formatLongDate } from '../lib/format';
import type { RootStackParamList } from '../navigation/types';
import { useStyles, type Theme } from '../theme/ThemeProvider';

type Props = NativeStackScreenProps<RootStackParamList, 'AttendanceDay'>;

const leaveLabels = { izin: 'Izin', cuti: 'Cuti', sakit: 'Sakit' } as const;

/** FR-10 · Detail satu hari: jam masuk, jam pulang, total jam kerja, dan keterangan izin. */
export function AttendanceDayScreen({ route }: Props) {
  const styles = useStyles(makeStyles);
  const { day } = route.params;
  const badge = dayStatusBadge(day.status);
  const record = day.attendance;

  return (
    <ScrollView style={styles.screen} contentContainerStyle={styles.content}>
      <View style={styles.header}>
        <Text accessibilityRole="header" style={styles.title}>
          {formatLongDate(day.date)}
        </Text>
        <Badge tone={badge.tone}>{badge.label}</Badge>
      </View>

      {record && (
        <Card title="Jam kerja">
          <WorkSummary record={record} />
        </Card>
      )}

      {record && (day.office_location_name || record.distance_in_meters != null) && (
        <Card title="Lokasi">
          {day.office_location_name && <Text style={styles.value}>{day.office_location_name}</Text>}
          {record.distance_in_meters != null && (
            <Text style={styles.meta}>Clock in {record.distance_in_meters} m dari titik kantor</Text>
          )}
          {record.distance_out_meters != null && (
            <Text style={styles.meta}>Clock out {record.distance_out_meters} m dari titik kantor</Text>
          )}
        </Card>
      )}

      {day.leave && (
        <Card title={leaveLabels[day.leave.type]}>
          <Text style={styles.value}>{day.leave.reason || 'Tanpa keterangan'}</Text>
        </Card>
      )}

      {!record && !day.leave && (
        <Text style={styles.meta}>
          {day.status === 'libur' ? 'Bukan hari kerja.' : 'Tidak ada absensi maupun izin tercatat untuk hari ini.'}
        </Text>
      )}
    </ScrollView>
  );
}

const makeStyles = ({ color, space, text }: Theme) =>
  StyleSheet.create({
    screen: { flex: 1, backgroundColor: color.bg.canvas },
    content: { gap: space[16], padding: space[20], paddingBottom: space[48] },
    header: { gap: space[8], alignItems: 'flex-start' },
    title: { ...text.headingH2, color: color.text.primary },
    value: { ...text.bodyM, color: color.text.primary },
    meta: { ...text.bodyS, color: color.text.secondary },
  });
