import { Text, View } from 'react-native';
import { useStyles, useTheme, type Theme } from '../../theme/ThemeProvider';

type Tone = 'neutral' | 'success' | 'warning' | 'danger' | 'inverse';

/** Label status kecil. Warna selalu disertai teks (WCAG 1.4.1). */
export function Badge({ tone = 'neutral', children }: { tone?: Tone; children: string }) {
  const styles = useStyles(makeStyles);
  const { color } = useTheme();
  const tones: Record<Tone, { background: string; border: string; foreground: string }> = {
    neutral: { background: color.bg.canvas, border: color.badge.neutralBorder, foreground: color.text.primary },
    success: { background: color.status.successBg, border: color.badge.successBorder, foreground: color.status.successFg },
    warning: { background: color.status.warningBg, border: color.badge.warningBorder, foreground: color.status.warningFg },
    danger: { background: color.status.dangerBg, border: color.badge.dangerBorder, foreground: color.status.dangerFg },
    inverse: { background: color.bg.inverse, border: color.badge.liveBorder, foreground: color.text.inverse },
  };
  const { background, border, foreground } = tones[tone];

  return (
    <View style={[styles.badge, { backgroundColor: background, borderColor: border }]}>
      <Text style={[styles.label, { color: foreground }]}>{children.toUpperCase()}</Text>
    </View>
  );
}

const makeStyles = ({ radius, space, text }: Theme) => ({
  badge: {
    alignSelf: 'flex-start' as const,
    paddingHorizontal: space[8],
    paddingVertical: space[4],
    borderWidth: 1,
    borderRadius: radius.sm,
  },
  label: { ...text.labelS },
});
