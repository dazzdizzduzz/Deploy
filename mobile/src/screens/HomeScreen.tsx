import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { ActivityIndicator, Pressable, RefreshControl, ScrollView, StyleSheet, Text, View } from 'react-native';
import { useAttendance } from '../attendance/AttendanceProvider';
import { statusLabel } from '../attendance/labels';
import { WorkSummary } from '../attendance/WorkSummary';
import { Alert } from '../components/ui/Alert';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Avatar } from '../components/ui/Avatar';
import { Card } from '../components/ui/Card';
import type { RootStackParamList } from '../navigation/types';
import { formatTime, timeZoneLabel } from '../lib/format';
import { firstName, formatToday, greeting, initialsOf, roleLabels } from '../profile/format';
import { useNotifications } from '../notifications/NotificationsProvider';
import { useProfile } from '../profile/ProfileProvider';
import { useStyles, useTheme, type Theme } from '../theme/ThemeProvider';

type Props = NativeStackScreenProps<RootStackParamList, 'Home'>;

/** Beranda karyawan setelah login: salam, ringkasan profil, dan menu. */
export function HomeScreen({ navigation }: Props) {
  const styles = useStyles(makeStyles);
  const { color } = useTheme();
  const { profile, loading, error, reload } = useProfile();
  const { today, serverState } = useAttendance();
  const { unreadCount } = useNotifications();
  const name = firstName(profile?.fullName ?? null);

  const menu = [
    { title: 'Riwayat absensi', description: 'Kehadiran dan jam kerja per bulan', onPress: () => navigation.navigate('AttendanceHistory') },
    { title: 'Izin & cuti', description: 'Ajukan izin, sakit, atau cuti dan pantau statusnya', onPress: () => navigation.navigate('LeaveRequests') },
    {
      title: 'Notifikasi',
      description: unreadCount ? `${unreadCount} belum dibaca` : 'Kabar status pengajuan izin',
      onPress: () => navigation.navigate('Notifications'),
    },
    { title: 'Profil Saya', description: 'Data diri, jabatan, dan perusahaan', onPress: () => navigation.navigate('Profile') },
    { title: 'Cek koneksi API', description: 'Pastikan aplikasi terhubung ke server', onPress: () => navigation.navigate('ApiStatus') },
    ...(__DEV__
      ? [{ title: 'Contoh komponen (dev)', description: 'Etalase komponen dasar', onPress: () => navigation.navigate('Components') }]
      : []),
  ];

  return (
    <ScrollView
      style={styles.screen}
      contentContainerStyle={styles.content}
      refreshControl={<RefreshControl refreshing={loading && Boolean(profile)} onRefresh={() => void reload()} />}
    >
      <View style={styles.header}>
        <Text style={styles.date}>{formatToday()}</Text>
        <Text accessibilityRole="header" style={styles.greeting}>
          {greeting()}
          {name ? `, ${name}` : ''}
        </Text>
      </View>

      {error && (
        <Alert tone="danger" title="Profil belum bisa dimuat" action={{ label: 'Coba lagi', onPress: () => void reload() }}>
          {error}
        </Alert>
      )}

      <Pressable
        accessibilityRole="button"
        accessibilityLabel={
          profile
            ? [profile.fullName ?? roleLabels[profile.role], profile.position, profile.tenantName, 'Buka Profil Saya']
                .filter(Boolean)
                .join(', ')
            : 'Buka Profil Saya'
        }
        onPress={() => navigation.navigate('Profile')}
        style={({ pressed }) => [styles.profileCard, pressed && styles.pressed]}
      >
        {loading && !profile ? (
          <ActivityIndicator color={color.text.primary} />
        ) : (
          <>
            <Avatar initials={initialsOf(profile?.fullName ?? null)} />
            <View style={styles.profileText}>
              <Text style={styles.profileName} numberOfLines={1}>
                {profile?.fullName ?? (profile ? roleLabels[profile.role] ?? profile.role : 'Profil')}
              </Text>
              <Text style={styles.profileMeta} numberOfLines={1}>
                {profile?.position ?? 'Jabatan belum tersedia'}
              </Text>
              <Text style={styles.profileTenant} numberOfLines={1}>
                {profile?.tenantName ?? 'Nama perusahaan belum tersedia'}
              </Text>
            </View>
            <Text style={styles.chevron} importantForAccessibility="no">
              ›
            </Text>
          </>
        )}
      </Pressable>

      {profile?.source === 'token' && (
        <Alert tone="info" title="Profil lengkap belum tersedia">
          Server baru mengirim peran akun ({roleLabels[profile.role] ?? profile.role}). Nama, jabatan, dan nama
          perusahaan tampil setelah endpoint profil aktif.
        </Alert>
      )}

      <Card title="Absensi hari ini">
        {today?.clock_out ? (
          <WorkSummary record={today} />
        ) : today ? (
          <>
            <View style={styles.attendance}>
              <Badge tone={statusLabel(today).tone}>{statusLabel(today).label}</Badge>
              <Text style={styles.attendanceTime}>
                Clock in {formatTime(today.clock_in)} {timeZoneLabel()}
              </Text>
            </View>
            <Button fullWidth label="Clock Out" onPress={() => navigation.navigate('ClockOut')} />
          </>
        ) : serverState === 'clocked_out' ? (
          <Text style={styles.rowMeta}>Kamu sudah clock in dan clock out hari ini (tercatat di server).</Text>
        ) : serverState === 'clocked_in' ? (
          <>
            <Text style={styles.rowMeta}>Kamu sudah clock in hari ini (tercatat di server).</Text>
            <Button fullWidth label="Clock Out" onPress={() => navigation.navigate('ClockOut')} />
          </>
        ) : (
          <>
            <Text style={styles.rowMeta}>Belum clock in. Pastikan kamu berada di area kantor.</Text>
            <Button fullWidth label="Clock In" onPress={() => navigation.navigate('ClockIn')} />
          </>
        )}
      </Card>

      <Card title="Menu">
        <View>
          {menu.map((item, index) => (
            <Pressable
              key={item.title}
              accessibilityRole="button"
              accessibilityLabel={`${item.title}, ${item.description}`}
              onPress={item.onPress}
              style={({ pressed }) => [styles.row, index > 0 && styles.rowBorder, pressed && styles.pressed]}
            >
              <View style={styles.rowText}>
                <Text style={styles.rowTitle}>{item.title}</Text>
                <Text style={styles.rowMeta}>{item.description}</Text>
              </View>
              <Text style={styles.chevron} importantForAccessibility="no">
                ›
              </Text>
            </Pressable>
          ))}
        </View>
      </Card>
    </ScrollView>
  );
}

const makeStyles = ({ color, radius, size, space, text }: Theme) =>
  StyleSheet.create({
    screen: { flex: 1, backgroundColor: color.bg.canvas },
    content: { gap: space[16], padding: space[20], paddingBottom: space[48] },
    header: { gap: space[4] },
    date: { ...text.bodyS, color: color.text.secondary },
    greeting: { ...text.headingH1, color: color.text.primary },
    profileCard: {
      flexDirection: 'row',
      alignItems: 'center',
      gap: space[12],
      minHeight: 88,
      padding: space[16],
      borderWidth: 1,
      borderColor: color.border.default,
      borderRadius: radius.lg,
      backgroundColor: color.bg.surface,
    },
    profileText: { flex: 1, gap: space[2] },
    profileName: { ...text.headingH3, color: color.text.primary },
    profileMeta: { ...text.bodyS, color: color.text.secondary },
    profileTenant: { ...text.caption, color: color.text.secondary },
    pressed: { backgroundColor: color.bg.hover },
    chevron: { ...text.headingH2, color: color.text.secondary },
    row: {
      flexDirection: 'row',
      alignItems: 'center',
      gap: space[12],
      minHeight: size.iconTarget + space[16],
      paddingVertical: space[8],
    },
    rowBorder: { borderTopWidth: 1, borderTopColor: color.border.default },
    rowText: { flex: 1, gap: space[2] },
    rowTitle: { ...text.labelM, color: color.text.primary },
    rowMeta: { ...text.caption, color: color.text.secondary },
    attendance: { gap: space[8], alignItems: 'flex-start' },
    attendanceTime: { ...text.headingH2, color: color.text.primary },
  });
