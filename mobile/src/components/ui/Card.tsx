import type { ReactNode } from 'react';
import { Text, View, type StyleProp, type ViewStyle } from 'react-native';
import { useStyles, type Theme } from '../../theme/ThemeProvider';

type CardProps = {
  title?: string;
  description?: string;
  children?: ReactNode;
  style?: StyleProp<ViewStyle>;
};

/** Wadah konten: bg/surface, garis border/default, radius lg, padding 20. */
export function Card({ title, description, children, style }: CardProps) {
  const styles = useStyles(makeStyles);
  return (
    <View style={[styles.card, style]}>
      {(title || description) && (
        <View style={styles.heading}>
          {title && (
            <Text accessibilityRole="header" style={styles.title}>
              {title}
            </Text>
          )}
          {description && <Text style={styles.description}>{description}</Text>}
        </View>
      )}
      {children}
    </View>
  );
}

const makeStyles = ({ color, radius, space, text }: Theme) => ({
  card: {
    alignSelf: 'stretch' as const,
    gap: space[16],
    padding: space[20],
    borderWidth: 1,
    borderColor: color.border.default,
    borderRadius: radius.lg,
    backgroundColor: color.bg.surface,
  },
  heading: { gap: space[4] },
  title: { ...text.headingH3, color: color.text.primary },
  description: { ...text.bodyS, color: color.text.secondary },
});
