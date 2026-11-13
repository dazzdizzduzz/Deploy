import type { ReactNode } from 'react';
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useStyles, type Theme } from '../../theme/ThemeProvider';

/** Kerangka layar lupa/reset kata sandi: isi yang bisa di-scroll dan tombol tetap di bawah. */
export function AuthLayout({ footer, children }: { footer: ReactNode; children: ReactNode }) {
  const styles = useStyles(makeStyles);
  const { bottom } = useSafeAreaInsets();
  return (
    <KeyboardAvoidingView style={styles.screen} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
      <ScrollView style={styles.scroll} contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        {children}
      </ScrollView>
      <View style={[styles.footer, { paddingBottom: Math.max(32, bottom + 12) }]}>{footer}</View>
    </KeyboardAvoidingView>
  );
}

export function AuthHeading({ title, children }: { title: string; children: ReactNode }) {
  const styles = useStyles(makeStyles);
  return (
    <View style={styles.heading}>
      <Text accessibilityRole="header" style={styles.title}>
        {title}
      </Text>
      <Text style={styles.body}>{children}</Text>
    </View>
  );
}

const makeStyles = ({ color, space, text }: Theme) =>
  StyleSheet.create({
    screen: { flex: 1, backgroundColor: color.bg.surface },
    scroll: { flex: 1 },
    content: { flexGrow: 1, gap: space[20], paddingHorizontal: space[20], paddingVertical: space[24] },
    footer: { gap: space[8], paddingTop: space[12], paddingHorizontal: space[20], backgroundColor: color.bg.surface },
    heading: { gap: space[8] },
    title: { ...text.headingH1, color: color.text.primary },
    body: { ...text.bodyM, color: color.text.secondary },
  });
