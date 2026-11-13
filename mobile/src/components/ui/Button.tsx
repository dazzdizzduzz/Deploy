import { ActivityIndicator, Pressable, Text } from 'react-native';
import { useStyles, useTheme, type Theme } from '../../theme/ThemeProvider';

type Variant = 'primary' | 'secondary' | 'ghost' | 'accent' | 'danger';

type ButtonProps = {
  label: string;
  onPress?: () => void;
  variant?: Variant;
  disabled?: boolean;
  /** Menampilkan indikator proses dan menonaktifkan tombol. */
  loading?: boolean;
  /** Lebar penuh dan tinggi 52px; tanpa ini setinggi 44px. */
  fullWidth?: boolean;
};

export function Button({
  label,
  onPress,
  variant = 'primary',
  disabled = false,
  loading = false,
  fullWidth = false,
}: ButtonProps) {
  const styles = useStyles(makeStyles);
  const { color } = useTheme();
  const inactive = disabled || loading;
  const spinnerColor =
    variant === 'primary' || variant === 'danger' ? color.text.inverse : color.text.primary;

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: inactive, busy: loading }}
      disabled={inactive}
      onPress={onPress}
      style={({ pressed }) => [
        styles.base,
        fullWidth ? styles.fullWidth : styles.compact,
        styles[variant],
        pressed && styles[`${variant}Pressed`],
        disabled && styles.disabled,
      ]}
    >
      {loading && <ActivityIndicator color={spinnerColor} />}
      <Text style={[styles.label, styles[`${variant}Label`], disabled && styles.labelDisabled]}>{label}</Text>
    </Pressable>
  );
}

const makeStyles = ({ color, radius, size, space, text }: Theme) => ({
  base: {
    flexDirection: 'row' as const,
    alignItems: 'center' as const,
    justifyContent: 'center' as const,
    gap: space[8],
    borderRadius: radius.md,
  },
  fullWidth: {
    alignSelf: 'stretch' as const,
    minHeight: size.buttonL,
    paddingHorizontal: space[24],
    paddingVertical: space[16],
  },
  compact: {
    minHeight: size.iconTarget,
    paddingHorizontal: space[16],
    paddingVertical: space[8],
  },
  disabled: { backgroundColor: color.bg.disabled, borderWidth: 0 },
  label: { ...text.labelM },
  labelDisabled: { color: color.text.disabled },

  primary: { backgroundColor: color.bg.inverse },
  primaryPressed: { backgroundColor: color.bg.inverseHover },
  primaryLabel: { color: color.text.inverse },

  secondary: { borderWidth: 1, borderColor: color.border.strong, backgroundColor: color.bg.surface },
  secondaryPressed: { backgroundColor: color.bg.hover },
  secondaryLabel: { color: color.text.primary },

  ghost: { backgroundColor: 'transparent' },
  ghostPressed: { backgroundColor: color.bg.hover },
  ghostLabel: { color: color.text.primary },

  accent: { backgroundColor: color.accent.default },
  accentPressed: { backgroundColor: color.accent.hover },
  accentLabel: { color: color.text.onAccent },

  danger: { backgroundColor: color.status.dangerFg },
  dangerPressed: { backgroundColor: color.status.dangerHover },
  dangerLabel: { color: color.text.inverse },
});
