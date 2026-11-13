import { useState } from 'react';
import { ScrollView, StyleSheet, Text, View } from 'react-native';
import { Alert } from '../components/ui/Alert';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Dialog } from '../components/ui/Dialog';
import { TextField } from '../components/ui/Field';
import { Select } from '../components/ui/Select';
import { ThemeProvider, useStyles, type Theme } from '../theme/ThemeProvider';
import type { ColorScheme } from '../theme/tokens';

const leaveTypes = [
  { value: 'cuti', label: 'Cuti tahunan' },
  { value: 'izin', label: 'Izin' },
  { value: 'sakit', label: 'Sakit' },
];

/** Etalase komponen dasar (hanya development) untuk memeriksa token di mode terang dan gelap. */
export function ComponentsScreen() {
  const [scheme, setScheme] = useState<ColorScheme>('light');
  return (
    <ThemeProvider scheme={scheme}>
      <Showcase scheme={scheme} onToggle={() => setScheme(scheme === 'light' ? 'dark' : 'light')} />
    </ThemeProvider>
  );
}

function Showcase({ scheme, onToggle }: { scheme: ColorScheme; onToggle: () => void }) {
  const styles = useStyles(makeStyles);
  const [leave, setLeave] = useState<string | null>(null);
  const [name, setName] = useState('');
  const [dialog, setDialog] = useState(false);

  return (
    <ScrollView style={styles.screen} contentContainerStyle={styles.content}>
      <Button
        variant="secondary"
        label={scheme === 'light' ? 'Coba mode gelap' : 'Kembali ke mode terang'}
        onPress={onToggle}
      />

      <Card title="Button" description="Primary, secondary, ghost, accent, danger, nonaktif, dan proses.">
        <Button fullWidth label="Primary" />
        <Button fullWidth variant="secondary" label="Secondary" />
        <Button fullWidth variant="ghost" label="Ghost" />
        <Button fullWidth variant="accent" label="Accent" />
        <Button fullWidth variant="danger" label="Danger" />
        <Button fullWidth disabled label="Nonaktif" />
        <Button fullWidth loading label="Memproses…" />
      </Card>

      <Card title="Input & Select">
        <TextField label="Nama" helper="Sesuai KTP" value={name} onChangeText={setName} placeholder="Nama lengkap" />
        <TextField label="Email" error="Format email tidak valid" value="daze@" onChangeText={() => undefined} />
        <Select label="Jenis izin" placeholder="Pilih jenis" options={leaveTypes} value={leave} onChange={setLeave} />
      </Card>

      <Card title="Badge">
        <View style={styles.row}>
          <Badge tone="success">Hadir</Badge>
          <Badge tone="warning">Terlambat</Badge>
          <Badge tone="danger">Alpa</Badge>
          <Badge>Izin</Badge>
          <Badge tone="inverse">Live</Badge>
        </View>
      </Card>

      <Card title="Alert">
        <Alert tone="success" title="Clock-in berhasil">Tercatat 08:02 di Kantor Pusat.</Alert>
        <Alert tone="warning" title="Hampir terlambat">Toleransi tinggal 3 menit.</Alert>
        <Alert tone="danger" title="Di luar area kantor" action={{ label: 'Coba lagi', onPress: () => undefined }}>
          Kamu berada 230 m dari lokasi kantor.
        </Alert>
        <Alert tone="info" title="Info">Jadwal kerja berubah mulai Senin.</Alert>
      </Card>

      <Card title="Dialog">
        <Button fullWidth variant="secondary" label="Buka dialog" onPress={() => setDialog(true)} />
      </Card>

      <Text style={styles.note}>Mode: {scheme === 'light' ? 'terang' : 'gelap'}</Text>

      <Dialog
        visible={dialog}
        title="Batalkan pengajuan izin?"
        description="Pengajuan yang dibatalkan tidak bisa dikembalikan."
        onClose={() => setDialog(false)}
        actions={
          <>
            <Button variant="secondary" label="Tidak" onPress={() => setDialog(false)} />
            <Button variant="danger" label="Batalkan" onPress={() => setDialog(false)} />
          </>
        }
      />
    </ScrollView>
  );
}

const makeStyles = ({ color, space, text }: Theme) =>
  StyleSheet.create({
    screen: { flex: 1, backgroundColor: color.bg.canvas },
    content: { gap: space[16], padding: space[16], paddingBottom: space[48] },
    row: { flexDirection: 'row', flexWrap: 'wrap', gap: space[8] },
    note: { ...text.caption, color: color.text.secondary, textAlign: 'center' },
  });
