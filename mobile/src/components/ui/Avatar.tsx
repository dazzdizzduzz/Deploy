import { Text, View } from 'react-native';
import { useStyles, type Theme } from '../../theme/ThemeProvider';

const sizes = { M: 44, L: 64 } as const;

/** Lingkaran inisial nama. Dekoratif: nama lengkap selalu ditulis di sebelahnya. */
export function Avatar({ initials, size = 'M' }: { initials: string; size?: keyof typeof sizes }) {
  const styles = useStyles(makeStyles);
  const dimension = sizes[size];
  return (
    <View
      importantForAccessibility="no-hide-descendants"
      accessibilityElementsHidden
      style={[styles.avatar, { width: dimension, height: dimension, borderRadius: dimension / 2 }]}
    >
      <Text style={size === 'L' ? styles.textL : styles.textM}>{initials}</Text>
    </View>
  );
}

const makeStyles = ({ color, text }: Theme) => ({
  avatar: {
    alignItems: 'center' as const,
    justifyContent: 'center' as const,
    backgroundColor: color.accent.subtle,
  },
  textM: { ...text.labelM, color: color.text.accent },
  textL: { ...text.headingH2, color: color.text.accent },
});
