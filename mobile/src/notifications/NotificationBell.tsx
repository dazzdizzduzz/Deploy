import { useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { Pressable, Text, View } from 'react-native';
import type { RootStackParamList } from '../navigation/types';
import { useStyles, type Theme } from '../theme/ThemeProvider';
import { useNotifications } from './NotificationsProvider';

/** Tombol Notifikasi di header Beranda dengan jumlah belum dibaca. */
export function NotificationBell() {
  const styles = useStyles(makeStyles);
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const { unreadCount } = useNotifications();
  const count = unreadCount ?? 0;

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={count > 0 ? `Notifikasi, ${count} belum dibaca` : 'Notifikasi'}
      onPress={() => navigation.navigate('Notifications')}
      hitSlop={8}
      style={({ pressed }) => [styles.button, pressed && styles.pressed]}
    >
      <Text style={styles.label}>Notifikasi</Text>
      {count > 0 && (
        <View style={styles.badge} importantForAccessibility="no-hide-descendants">
          <Text style={styles.badgeText}>{count > 99 ? '99+' : count}</Text>
        </View>
      )}
    </Pressable>
  );
}

const makeStyles = ({ color, radius, size, space, text }: Theme) => ({
  button: {
    flexDirection: 'row' as const,
    alignItems: 'center' as const,
    gap: space[4],
    minHeight: size.iconTarget,
    paddingHorizontal: space[8],
    borderRadius: radius.md,
  },
  pressed: { backgroundColor: color.bg.hover },
  label: { ...text.labelM, color: color.text.primary },
  badge: {
    minWidth: 20,
    height: 20,
    paddingHorizontal: space[4],
    alignItems: 'center' as const,
    justifyContent: 'center' as const,
    borderRadius: radius.full,
    backgroundColor: color.bg.inverse,
  },
  badgeText: { ...text.caption, color: color.text.inverse, fontFamily: text.labelM.fontFamily },
});
