import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useEffect, useMemo, useState } from 'react';
import { ScrollView, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { clockOut, listLocations, type Attendance, type OfficeLocation } from '../api/attendance';
import { useAttendance } from '../attendance/AttendanceProvider';
import { attendanceFailure, type Failure } from '../attendance/errors';
import { checkGeofence } from '../attendance/geo';
import { formatDuration } from '../attendance/labels';
import { LocationStatus } from '../attendance/LocationStatus';
import { SelfieCapture } from '../attendance/SelfieCapture';
import { useCurrentPosition } from '../attendance/useCurrentPosition';
import { WorkSummary } from '../attendance/WorkSummary';
import { Alert } from '../components/ui/Alert';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Dialog } from '../components/ui/Dialog';
import { formatTime, timeZoneLabel } from '../lib/format';
import type { RootStackParamList } from '../navigation/types';
import { usePermissions } from '../permissions/PermissionsProvider';
import { useStyles, type Theme } from '../theme/ThemeProvider';

type Props = NativeStackScreenProps<RootStackParamList, 'ClockOut'>;

/**
 * FR-07 · Clock Out: lokasi harus di radius kantor yang sama dengan saat clock-in (aturan backend),
 * selfie bila diwajibkan lokasi, lalu tampilkan total jam kerja dan lembur hari itu.
 */
export function ClockOutScreen({ navigation }: Props) {
  const styles = useStyles(makeStyles);
  const { bottom } = useSafeAreaInsets();
  const permissions = usePermissions();
  const { today, saveToday, markServerState } = useAttendance();
  const locationGranted = permissions.location === 'granted';
  const cameraGranted = permissions.camera === 'granted';
  const gps = useCurrentPosition(locationGranted);

  const [locations, setLocations] = useState<OfficeLocation[] | null>(null);
  const [cameraOpen, setCameraOpen] = useState(false);
  const [selfieUri, setSelfieUri] = useState<string | null>(null);
  const [forceSelfie, setForceSelfie] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [failure, setFailure] = useState<Failure | null>(null);
  const [result, setResult] = useState<Attendance | null>(null);

  useEffect(() => {
    listLocations()
      .then(({ data }) => setLocations(data))
      .catch(() => setLocations([]));
  }, []);

  // Clock-out dibandingkan dengan kantor tempat clock-in, bukan kantor terdekat.
  const office = locations?.find((item) => item.id === today?.office_location_id) ?? null;
  const geofence = useMemo(
    () => (gps.position && office ? checkGeofence(gps.position, [{ ...office, is_active: true }]) : null),
    [gps.position, office],
  );
  const selfieRequired = forceSelfie || Boolean(office?.selfie_required);
  const sinceClockIn = today ? Math.max(0, Math.floor((Date.now() - Date.parse(today.clock_in)) / 60_000)) : 0;

  function requestSubmit() {
    if (!gps.position || submitting) return;
    if (selfieRequired && !selfieUri) {
      setCameraOpen(true);
      return;
    }
    // Clock-out hanya bisa sekali, jadi minta konfirmasi agar tidak tertekan tanpa sengaja.
    setConfirming(true);
  }

  async function submit() {
    const position = gps.position;
    setConfirming(false);
    if (!position || submitting) return;
    setSubmitting(true);
    setFailure(null);
    try {
      const record = await clockOut({
        latitude: position.latitude,
        longitude: position.longitude,
        selfieUri: selfieUri ?? undefined,
      });
      await saveToday(record);
      setResult(record);
    } catch (error) {
      setFailure(
        attendanceFailure(error, 'Clock out', {
          retry: () => void submit(),
          refreshLocation: () => void gps.read(),
          retakeSelfie: () => {
            setForceSelfie(true);
            setSelfieUri(null);
            setCameraOpen(true);
          },
          special: {
            ALREADY_CLOCKED_OUT: (title, body) => {
              markServerState('clocked_out');
              return { title, body, action: { label: 'Kembali ke Beranda', onPress: () => navigation.goBack() } };
            },
            NOT_CLOCKED_IN: (title, body) => {
              markServerState(null);
              return { title, body, action: { label: 'Kembali ke Beranda', onPress: () => navigation.goBack() } };
            },
          },
        }),
      );
    } finally {
      setSubmitting(false);
    }
  }

  if (result) {
    return (
      <View style={styles.screen}>
        <ScrollView contentContainerStyle={styles.content}>
          <Text accessibilityRole="header" style={styles.title}>
            Clock out berhasil
          </Text>
          <Card title="Ringkasan hari ini">
            <WorkSummary record={result} />
          </Card>
        </ScrollView>
        <View style={[styles.footer, { paddingBottom: Math.max(32, bottom + 12) }]}>
          <Button fullWidth label="Kembali ke Beranda" onPress={() => navigation.goBack()} />
        </View>
      </View>
    );
  }

  const needsPermission = !locationGranted || (selfieRequired && !cameraGranted);
  const outside = geofence !== null && !geofence.inside;

  return (
    <View style={styles.screen}>
      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        {today && (
          <Card>
            <Text style={styles.label}>Clock in hari ini</Text>
            <Text style={styles.time}>
              {formatTime(today.clock_in)} {timeZoneLabel()}
            </Text>
            <Text style={styles.hint}>Sudah bekerja {formatDuration(sinceClockIn)}.</Text>
          </Card>
        )}

        {needsPermission && (
          <Alert
            tone="warning"
            title="Izin belum lengkap"
            action={{ label: 'Atur izin', onPress: () => navigation.navigate('Permissions', { from: 'profile' }) }}
          >
            {!locationGranted
              ? 'Clock out butuh izin lokasi untuk memastikan kamu berada di radius kantor.'
              : 'Lokasi ini mewajibkan selfie, jadi clock out butuh izin kamera.'}
          </Alert>
        )}

        {failure && (
          <Alert tone="danger" title={failure.title} action={failure.action} onDismiss={() => setFailure(null)}>
            {failure.body}
          </Alert>
        )}

        <Card title="Lokasi">
          <LocationStatus
            granted={locationGranted}
            loading={gps.locating || locations === null}
            error={gps.error}
            position={gps.position}
            check={geofence}
            emptyText="Kantor tempat clock in tidak ditemukan. Server tetap memeriksa saat clock out."
            onRefresh={() => void gps.read()}
          />
        </Card>

        {(selfieRequired || selfieUri || cameraOpen) && (
          <Card title={`Selfie${selfieRequired ? ' (wajib)' : ''}`}>
            <SelfieCapture
              uri={selfieUri}
              onChange={setSelfieUri}
              open={cameraOpen}
              onOpenChange={setCameraOpen}
              cameraGranted={cameraGranted}
              onError={(body) => setFailure({ title: 'Foto belum terambil', body })}
            />
          </Card>
        )}

        <Text style={styles.hint}>
          Lembur dan pulang cepat dihitung server dari jadwal kerja kamu saat clock out.
        </Text>
      </ScrollView>

      <View style={[styles.footer, { paddingBottom: Math.max(32, bottom + 12) }]}>
        <Button
          fullWidth
          label={submitting ? 'Mengirim…' : 'Clock Out sekarang'}
          loading={submitting}
          disabled={!gps.position || !locationGranted || cameraOpen || outside}
          onPress={requestSubmit}
        />
      </View>

      <Dialog
        visible={confirming}
        title="Clock out sekarang?"
        description="Clock out hanya bisa sekali per hari kerja dan tidak bisa dibatalkan."
        onClose={() => setConfirming(false)}
        actions={
          <>
            <Button variant="secondary" label="Batal" onPress={() => setConfirming(false)} />
            <Button label="Clock out" onPress={() => void submit()} />
          </>
        }
      />
    </View>
  );
}

const makeStyles = ({ color, space, text }: Theme) =>
  StyleSheet.create({
    screen: { flex: 1, backgroundColor: color.bg.canvas },
    content: { gap: space[16], padding: space[20], paddingBottom: space[24] },
    title: { ...text.headingH1, color: color.text.primary },
    label: { ...text.caption, color: color.text.secondary },
    time: { ...text.numericKpi, color: color.text.primary },
    hint: { ...text.bodyS, color: color.text.secondary },
    footer: { gap: space[8], paddingTop: space[12], paddingHorizontal: space[20], backgroundColor: color.bg.canvas },
  });
