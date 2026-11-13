import { useEffect, type ReactNode } from 'react';
import { AccessibilityInfo, Text, View } from 'react-native';
import { useStyles, useTheme, type Theme } from '../../theme/ThemeProvider';
import { Button } from './Button';

type Tone = 'success' | 'warning' | 'danger' | 'info';

type AlertProps = {
  tone: Tone;
  title: string;
  children: ReactNode;
  /** Tombol aksi di dalam alert, misalnya "Coba lagi". */
  action?: { label: string; onPress: () => void };
  /** Menampilkan tombol "Tutup". */
  onDismiss?: () => void;
};

export function Alert({ tone, title, children, action, onDismiss }: AlertProps) {
  const styles = useStyles(makeStyles);
  const { color } = useTheme();
  const tones: Record<Tone, { background: string; accent: string }> = {
    success: { background: color.status.successBg, accent: color.status.successFg },
    warning: { background: color.status.warningBg, accent: color.status.warningFg },
    danger: { background: color.status.dangerBg, accent: color.status.dangerFg },
    info: { background: color.bg.canvas, accent: color.text.primary },
  };
  const { background, accent } = tones[tone];

  // Bacakan lewat pembaca layar setiap kali alert baru muncul.
  useEffect(() => {
    AccessibilityInfo.announceForAccessibility(title);
  }, [title]);

  return (
    <View
      accessibilityRole="alert"
      accessibilityLiveRegion="assertive"
      style={[styles.alert, { backgroundColor: background, borderLeftColor: accent }]}
    >
      <Text style={[styles.title, { color: accent }]}>{title}</Text>
      <Text style={styles.body}>{children}</Text>
      {(action || onDismiss) && (
        <View style={styles.actions}>
          {onDismiss && <Button variant="ghost" label="Tutup" onPress={onDismiss} />}
          {action && <Button variant="secondary" label={action.label} onPress={action.onPress} />}
        </View>
      )}
    </View>
  );
}

const makeStyles = ({ color, radius, space, text }: Theme) => ({
  alert: {
    alignSelf: 'stretch' as const,
    gap: space[4],
    padding: space[16],
    borderLeftWidth: 4,
    borderRadius: radius.md,
  },
  title: { ...text.labelM },
  body: { ...text.bodyS, color: color.text.primary },
  actions: {
    flexDirection: 'row' as const,
    justifyContent: 'flex-end' as const,
    gap: space[8],
    marginTop: space[8],
  },
});
