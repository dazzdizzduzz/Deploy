import { CameraView } from 'expo-camera';
import { ImageManipulator, SaveFormat } from 'expo-image-manipulator';
import { useRef, useState } from 'react';
import { Image, StyleSheet, Text, View } from 'react-native';
import { Button } from '../components/ui/Button';
import { useStyles, type Theme } from '../theme/ThemeProvider';

/** Lebar foto setelah diperkecil: cukup untuk wajah, jauh di bawah batas 2 MB backend. */
const SELFIE_WIDTH = 720;

type SelfieCaptureProps = {
  uri: string | null;
  onChange: (uri: string | null) => void;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  cameraGranted: boolean;
  onError: (message: string) => void;
};

/** Kamera depan untuk selfie absensi; hasilnya JPEG 720 px. */
export function SelfieCapture({ uri, onChange, open, onOpenChange, cameraGranted, onError }: SelfieCaptureProps) {
  const styles = useStyles(makeStyles);
  const cameraRef = useRef<CameraView>(null);
  const [capturing, setCapturing] = useState(false);

  async function take() {
    if (!cameraRef.current || capturing) return;
    setCapturing(true);
    try {
      const photo = await cameraRef.current.takePictureAsync({ quality: 0.8 });
      const image = await ImageManipulator.manipulate(photo.uri).resize({ width: SELFIE_WIDTH }).renderAsync();
      const saved = await image.saveAsync({ compress: 0.7, format: SaveFormat.JPEG });
      onChange(saved.uri);
      onOpenChange(false);
    } catch {
      onError('Kamera tidak merespons. Coba ambil foto lagi.');
    } finally {
      setCapturing(false);
    }
  }

  if (open && cameraGranted) {
    return (
      <>
        <CameraView ref={cameraRef} facing="front" style={styles.camera} />
        <View style={styles.row}>
          <Button variant="ghost" label="Batal" onPress={() => onOpenChange(false)} />
          <Button label="Ambil foto" loading={capturing} onPress={() => void take()} />
        </View>
      </>
    );
  }

  if (uri) {
    return (
      <View style={styles.row}>
        <Image source={{ uri }} style={styles.preview} accessibilityLabel="Foto selfie yang akan dikirim" />
        <Button variant="secondary" label="Ulangi" onPress={() => onOpenChange(true)} />
      </View>
    );
  }

  return (
    <>
      <Text style={styles.hint}>Wajah terlihat jelas, tanpa masker atau kacamata hitam.</Text>
      <Button variant="secondary" label="Buka kamera depan" disabled={!cameraGranted} onPress={() => onOpenChange(true)} />
    </>
  );
}

const makeStyles = ({ color, radius, space, text }: Theme) =>
  StyleSheet.create({
    camera: { height: 360, borderRadius: radius.md, overflow: 'hidden' },
    row: { flexDirection: 'row', alignItems: 'center', gap: space[12] },
    preview: { width: 96, height: 128, borderRadius: radius.md, backgroundColor: color.bg.subtle },
    hint: { ...text.bodyS, color: color.text.secondary },
  });
