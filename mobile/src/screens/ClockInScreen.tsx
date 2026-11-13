import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useEffect, useMemo, useState } from 'react';
import { ScrollView, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { clockIn, listLocations, listSchedules, type Attendance, type OfficeLocation, type WorkSchedule } from '../api/attendance';
import { useAttendance } from '../attendance/AttendanceProvider';
import { attendanceFailure, type Failure } from '../attendance/errors';
import { checkGeofence } from '../attendance/geo';
import { statusLabel } from '../attendance/labels';
import { LocationStatus } from '../attendance/LocationStatus';
import { SelfieCapture } from '../attendance/SelfieCapture';
import { useCurrentPosition } from '../attendance/useCurrentPosition';
import { Alert } from '../components/ui/Alert';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Select } from '../components/ui/Select';
import { formatLongDate, formatTime, timeZoneLabel } from '../lib/format';
import type { RootStackParamList } from '../navigation/types';
import { usePermissions } from '../permissions/PermissionsProvider';
import { useStyles, type Theme } from '../theme/ThemeProvider';

type Props = NativeStackScreenProps<RootStackParamList, 'ClockIn'>;

/**
 * FR-06 · Clock In dengan geofence dan selfie, maksimal 3 langkah (NFR-09):
 * 1) lokasi dibaca otomatis, 2) ambil selfie (bila diwajibkan atau ingin), 3) kirim.
 */
export function ClockInScreen({ navigation }: Props) {
  const styles = useStyles(makeStyles);
  const { bottom } = useSafeAreaInsets();
  const permissions = usePermissions();
  const { saveToday, markServerState } = useAttendance();
  const locationGranted = permissions.location === 'granted';
  const cameraGranted = permissions.camera === 'granted';
  const gps = useCurrentPosition(locationGranted);

  const [locations, setLocations] = useState<OfficeLocation[] | null>(null);
  const [schedules, setSchedules] = useState<WorkSchedule[]>([]);
  const [scheduleId, setScheduleId] = useState<string | null>(null);
  const [cameraOpen, setCameraOpen] = useState(false);
  const [selfieUri, setSelfieUri] = useState<string | null>(null);
  const [forceSelfie, setForceSelfie] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [failure, setFailure] = useState<Failure | null>(null);
  const [result, setResult] = useState<Attendance | null>(null);

  const activeSchedules = schedules.filter((schedule) => schedule.is_active);
  const geofence = useMemo(
    () => (gps.position && locations ? checkGeofence(gps.position, locations) : null),
    [gps.position, locations],
  );
  const selfieRequired = forceSelfie || Boolean(geofence?.location.selfie_required);

  // Data kantor dan jadwal dimuat sekali; dipakai untuk menampilkan status radius sebelum dikirim.
  useEffect(() => {
    listLocations()
      .then(({ data }) => setLocations(data))
      .catch(() => setLocations([]));
    listSchedules()
      .then(({ data }) => setSchedules(data))
      .catch(() => setSchedules([]));
  }, []);

  async function submit() {
    const position = gps.position;
    if (!position || submitting) return;
    if (selfieRequired && !selfieUri) {
      setCameraOpen(true);
      return;
    }
    if (activeSchedules.length > 1 && !scheduleId) {
      setFailure({ title: 'Pilih jadwal kerja', body: 'Ada lebih dari satu jadwal aktif. Pilih jadwal hari ini.' });
      return;
    }
    setSubmitting(true);
    setFailure(null);
    try {
      const record = await clockIn({
        latitude: position.latitude,
        longitude: position.longitude,
        workScheduleId: activeSchedules.length > 1 ? (scheduleId ?? undefined) : undefined,
        selfieUri: selfieUri ?? undefined,
      });
      await saveToday(record);
      setResult(record);
    } catch (error) {
      setFailure(
        attendanceFailure(error, 'Clock in', {
          retry: () => void submit(),
          refreshLocation: () => void gps.read(),
          retakeSelfie: () => {
            setForceSelfie(true);
            setSelfieUri(null);
            setCameraOpen(true);
          },
          special: {
            ALREADY_CLOCKED_IN: (title, body) => {
              markServerState('clocked_in');
              return { title, body, action: { label: 'Kembali ke Beranda', onPress: () => navigation.goBack() } };
            },
            SCHEDULE_AMBIGUOUS: (title, body) => ({
              title,
              body: activeSchedules.length > 1 ? 'Pilih jadwal kerja hari ini, lalu kirim lagi.' : body,
            }),
          },
        }),
      );
    } finally {
      setSubmitting(false);
    }
  }

  if (result) {
    const badge = statusLabel(result);
    const office = locations?.find((item) => item.id === result.office_location_id);
    return (
      <View style={styles.screen}>
        <ScrollView contentContainerStyle={styles.content}>
          <View style={styles.resultHeader}>
            <Text accessibilityRole="header" style={styles.title}>
              Clock in berhasil
            </Text>
            <Badge tone={badge.tone}>{badge.label}</Badge>
          </View>
          <Card>
            <View style={styles.resultRow}>
              <Text style={styles.label}>Jam masuk</Text>
              <Text style={styles.time}>
                {formatTime(result.clock_in)} {timeZoneLabel()}
              </Text>
            </View>
            <View style={styles.resultRow}>
              <Text style={styles.label}>Tanggal</Text>
              <Text style={styles.value}>{formatLongDate(result.work_date)}</Text>
            </View>
            {office && (
              <View style={styles.resultRow}>
                <Text style={styles.label}>Lokasi</Text>
                <Text style={styles.value}>
                  {office.name}
                  {result.distance_in_meters != null ? ` · ${result.distance_in_meters} m dari titik kantor` : ''}
                </Text>
              </View>
            )}
          </Card>
        </ScrollView>
        <View style={[styles.footer, { paddingBottom: Math.max(32, bottom + 12) }]}>
          <Button fullWidth label="Kembali ke Beranda" onPress={() => navigation.goBack()} />
        </View>
      </View>
    );
  }

  const needsPermission = !locationGranted || (selfieRequired && !cameraGranted);
  const step = !gps.position || !geofence?.inside ? 1 : selfieRequired && !selfieUri ? 2 : 3;

  return (
    <View style={styles.screen}>
      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        <Text style={styles.steps}>Langkah {step} dari 3 · Lokasi · Selfie · Kirim</Text>

        {needsPermission && (
          <Alert
            tone="warning"
            title="Izin belum lengkap"
            action={{ label: 'Atur izin', onPress: () => navigation.navigate('Permissions', { from: 'profile' }) }}
          >
            {!locationGranted
              ? 'Clock in butuh izin lokasi untuk memastikan kamu berada di radius kantor.'
              : 'Lokasi ini mewajibkan selfie, jadi clock in butuh izin kamera.'}
          </Alert>
        )}

        {failure && (
          <Alert tone="danger" title={failure.title} action={failure.action} onDismiss={() => setFailure(null)}>
            {failure.body}
          </Alert>
        )}

        <Card title="1 · Lokasi">
          <LocationStatus
            granted={locationGranted}
            loading={gps.locating || locations === null}
            error={gps.error}
            position={gps.position}
            check={geofence}
            emptyText={
              locations?.length === 0
                ? 'Lokasi kantor belum diatur atau belum bisa dimuat. Server tetap memeriksa saat clock in.'
                : 'Belum ada lokasi kantor aktif. Hubungi HRD.'
            }
            onRefresh={() => void gps.read()}
          />
        </Card>

        {activeSchedules.length > 1 && (
          <Select
            label="Jadwal kerja hari ini"
            placeholder="Pilih jadwal"
            options={activeSchedules.map((schedule) => ({
              value: schedule.id,
              label: `${schedule.name} (${schedule.start_time.slice(0, 5)}–${schedule.end_time.slice(0, 5)})`,
            }))}
            value={scheduleId}
            onChange={setScheduleId}
          />
        )}

        <Card title={`2 · Selfie${selfieRequired ? ' (wajib)' : ' (opsional)'}`}>
          <SelfieCapture
            uri={selfieUri}
            onChange={setSelfieUri}
            open={cameraOpen}
            onOpenChange={setCameraOpen}
            cameraGranted={cameraGranted}
            onError={(body) => setFailure({ title: 'Foto belum terambil', body })}
          />
        </Card>

        <Text style={styles.hint}>
          {formatLongDate(new Date())} · Waktu clock in dicatat server ({timeZoneLabel()}).
        </Text>
      </ScrollView>

      <View style={[styles.footer, { paddingBottom: Math.max(32, bottom + 12) }]}>
        <Button
          fullWidth
          label={submitting ? 'Mengirim…' : 'Clock In sekarang'}
          loading={submitting}
          disabled={!gps.position || !locationGranted || cameraOpen || (geofence !== null && !geofence.inside)}
          onPress={() => void submit()}
        />
      </View>
    </View>
  );
}

const makeStyles = ({ color, space, text }: Theme) =>
  StyleSheet.create({
    screen: { flex: 1, backgroundColor: color.bg.canvas },
    content: { gap: space[16], padding: space[20], paddingBottom: space[24] },
    steps: { ...text.labelS, color: color.text.secondary, textTransform: 'uppercase' },
    title: { ...text.headingH1, color: color.text.primary },
    value: { ...text.bodyM, color: color.text.primary, flexShrink: 1 },
    hint: { ...text.bodyS, color: color.text.secondary },
    footer: { gap: space[8], paddingTop: space[12], paddingHorizontal: space[20], backgroundColor: color.bg.canvas },
    resultHeader: { gap: space[12], alignItems: 'flex-start' },
    resultRow: { gap: space[2], paddingVertical: space[4] },
    label: { ...text.caption, color: color.text.secondary },
    time: { ...text.numericKpi, color: color.text.primary },
  });
