import type { ReactNode } from 'react';
import { Modal, Pressable, Text, View } from 'react-native';
import { useStyles, type Theme } from '../../theme/ThemeProvider';

type DialogProps = {
  visible: boolean;
  title: string;
  description?: string;
  /** Tombol aksi, misalnya Batal dan Simpan. */
  actions: ReactNode;
  onClose: () => void;
  children?: ReactNode;
};

/** Dialog di tengah layar. Tombol kembali Android dan ketukan di luar dialog menutupnya. */
export function Dialog({ visible, title, description, actions, onClose, children }: DialogProps) {
  const styles = useStyles(makeStyles);
  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onClose}>
      <View style={styles.backdrop}>
        <Pressable style={styles.scrim} onPress={onClose} accessibilityLabel="Tutup dialog" />
        <View style={styles.dialog} accessibilityViewIsModal accessibilityRole="alert">
          <Text accessibilityRole="header" style={styles.title}>
            {title}
          </Text>
          {description && <Text style={styles.description}>{description}</Text>}
          {children}
          <View style={styles.actions}>{actions}</View>
        </View>
      </View>
    </Modal>
  );
}

const makeStyles = ({ color, radius, space, text }: Theme) => ({
  backdrop: { flex: 1, alignItems: 'center' as const, justifyContent: 'center' as const, padding: space[24] },
  scrim: { position: 'absolute' as const, top: 0, right: 0, bottom: 0, left: 0, backgroundColor: color.bg.scrim },
  dialog: {
    width: '100%' as const,
    maxWidth: 400,
    gap: space[16],
    padding: space[24],
    borderRadius: radius.lg,
    backgroundColor: color.bg.surface,
  },
  title: { ...text.headingH2, color: color.text.primary },
  description: { ...text.bodyS, color: color.text.secondary },
  actions: { flexDirection: 'row' as const, justifyContent: 'flex-end' as const, gap: space[8] },
});
