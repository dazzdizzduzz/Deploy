import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useState } from 'react';
import { Linking, ScrollView, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Alert } from '../components/ui/Alert';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import type { RootStackParamList } from '../navigation/types';
import { statusLabels, usePermissions, type Permission, type PermissionStatus } from '../permissions/PermissionsProvider';
import { useStyles, type Theme } from '../theme/ThemeProvider';

type Props = NativeStackScreenProps<RootStackParamList, 'Permissions'>;

const reasons: Record<Permission, string> = {
  lokasi: 'lokasi untuk memastikan kamu berada di radius kantor',
  kamera: 'kamera untuk selfie bukti kehadiran',
};

/** Langkah di Pengaturan untuk izin yang hanya bisa diubah dari sana (M56). */
const settingsSteps = (missing: Permission[]) =>
  `Buka Pengaturan › On-Track › ${missing
    .map((name) => (name === 'lokasi' ? 'Lokasi, pilih "Saat aplikasi dipakai"' : 'Kamera, aktifkan'))
    .join('; lalu ')}.`;

const badgeTone = (status: PermissionStatus) =>
  status === 'granted' ? 'success' : status === 'undetermined' ? 'neutral' : 'danger';

/**
 * M13 · Izin Akses Lokasi & Kamera (NFR-06, UU PDP No. 27/2022). Menjelaskan pemakaian data sebelum
 * dialog izin sistem muncul. M56 · arahan ke Pengaturan bila izin ditolak permanen.
 */
export function PermissionScreen({ navigation, route }: Props) {
  const styles = useStyles(makeStyles);
  const { bottom } = useSafeAreaInsets();
  const { location, camera, request, markIntroSeen } = usePermissions();
  const [asking, setAsking] = useState(false);
  const [attempted, setAttempted] = useState(false);
  const fromIntro = route.params?.from !== 'profile';

  const missing: Permission[] = [
    ...(location !== 'granted' ? (['lokasi'] as const) : []),
    ...(camera !== 'granted' ? (['kamera'] as const) : []),
  ];
  const allGranted = missing.length === 0;
  const blocked = location === 'blocked' || camera === 'blocked';
  // Status dibaca ulang dari sistem saat kembali dari Pengaturan, jadi tampilan selalu terkini.
  const showDenied = !allGranted && (attempted || blocked || location === 'denied' || camera === 'denied');

  function leave() {
    void markIntroSeen();
    if (fromIntro) navigation.replace('Home');
    else navigation.goBack();
  }

  async function allow() {
    if (blocked) {
      void Linking.openSettings();
      return;
    }
    setAsking(true);
    await markIntroSeen();
    const result = await request();
    setAsking(false);
    setAttempted(true);
    if (result.granted && fromIntro) navigation.replace('Home');
  }

  return (
    <View style={styles.screen}>
      <ScrollView contentContainerStyle={styles.content}>
        <View style={styles.heading}>
          <Text accessibilityRole="header" style={styles.title}>
            Izinkan lokasi & kamera
          </Text>
          <Text style={styles.subtitle}>On-Track butuh dua izin ini supaya absensi kamu sah dan cepat.</Text>
        </View>

        <Card>
          <View style={styles.item}>
            <View style={styles.itemHeader}>
              <Text style={styles.itemTitle}>Lokasi (GPS)</Text>
              <Badge tone={badgeTone(location)}>{statusLabels[location]}</Badge>
            </View>
            <Text style={styles.itemBody}>Memastikan kamu berada di radius kantor saat Clock In/Out.</Text>
          </View>
          <View style={[styles.item, styles.itemBorder]}>
            <View style={styles.itemHeader}>
              <Text style={styles.itemTitle}>Kamera</Text>
              <Badge tone={badgeTone(camera)}>{statusLabels[camera]}</Badge>
            </View>
            <Text style={styles.itemBody}>Mengambil selfie sebagai bukti kehadiran.</Text>
          </View>
        </Card>

        {allGranted && attempted && (
          <Alert tone="success" title="Izin sudah lengkap">
            Kamu bisa absen dengan lokasi dan selfie.
          </Alert>
        )}
        {showDenied && (
          <Alert
            tone="warning"
            title={`Izin ${missing.join(' dan ')} ${blocked ? 'ditolak' : 'belum diberikan'}`}
          >
            {blocked
              ? settingsSteps(missing)
              : `Clock In/Out butuh ${missing.map((name) => reasons[name]).join(' serta ')}. Tanpa izin ini kamu belum bisa absen dari aplikasi.`}
          </Alert>
        )}

        <Card title="Bagaimana data kamu dipakai" description="Sesuai UU Pelindungan Data Pribadi No. 27/2022.">
          <View style={styles.points}>
            <Text style={styles.point}>• Lokasi hanya dibaca saat kamu menekan Clock In atau Clock Out, tidak dilacak di latar belakang.</Text>
            <Text style={styles.point}>• Foto selfie hanya diambil saat Clock In/Out dan disimpan privat di server perusahaan, tidak dibagikan ke pihak lain.</Text>
            <Text style={styles.point}>• Data absensi hanya bisa dilihat kamu dan pengelola SDM di perusahaanmu (HRD/Owner).</Text>
            <Text style={styles.point}>• Kamu bisa mencabut izin kapan saja lewat Pengaturan perangkat, dan meminta salinan atau penghapusan data lewat HRD.</Text>
          </View>
        </Card>
      </ScrollView>

      <View style={[styles.footer, { paddingBottom: Math.max(32, bottom + 12) }]}>
        {allGranted ? (
          <Button fullWidth label={fromIntro ? 'Lanjut ke Beranda' : 'Selesai'} onPress={leave} />
        ) : (
          <>
            <Button
              fullWidth
              label={blocked ? 'Buka Pengaturan' : attempted ? 'Coba izinkan lagi' : 'Izinkan akses'}
              loading={asking}
              onPress={() => void allow()}
            />
            <Button fullWidth variant="ghost" label="Nanti saja" onPress={leave} />
          </>
        )}
      </View>
    </View>
  );
}

const makeStyles = ({ color, space, text }: Theme) =>
  StyleSheet.create({
    screen: { flex: 1, backgroundColor: color.bg.canvas },
    content: { gap: space[16], padding: space[20], paddingBottom: space[24] },
    heading: { gap: space[8], paddingTop: space[8] },
    title: { ...text.headingH1, color: color.text.primary },
    subtitle: { ...text.bodyM, color: color.text.secondary },
    item: { gap: space[4], paddingVertical: space[8] },
    itemBorder: { borderTopWidth: 1, borderTopColor: color.border.default, paddingTop: space[16] },
    itemHeader: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', gap: space[8] },
    itemTitle: { ...text.headingH3, color: color.text.primary, flexShrink: 1 },
    itemBody: { ...text.bodyS, color: color.text.secondary },
    points: { gap: space[8] },
    point: { ...text.bodyS, color: color.text.primary },
    footer: { gap: space[8], paddingTop: space[12], paddingHorizontal: space[20], backgroundColor: color.bg.canvas },
  });
