import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { ActivityIndicator, StyleSheet, View } from 'react-native';
import { useAuth } from '../auth/AuthProvider';
import { ApiStatusScreen } from '../screens/ApiStatusScreen';
import { HomeScreen } from '../screens/HomeScreen';
import { ComponentsScreen } from '../screens/ComponentsScreen';
import { CheckEmailScreen } from '../screens/auth/CheckEmailScreen';
import { ForgotPasswordScreen } from '../screens/auth/ForgotPasswordScreen';
import { NewPasswordScreen } from '../screens/auth/NewPasswordScreen';
import { PasswordChangedScreen } from '../screens/auth/PasswordChangedScreen';
import { LoginScreen } from '../screens/LoginScreen';
import { AttendanceProvider } from '../attendance/AttendanceProvider';
import { ClockInScreen } from '../screens/ClockInScreen';
import { ClockOutScreen } from '../screens/ClockOutScreen';
import { AttendanceDayScreen } from '../screens/AttendanceDayScreen';
import { AttendanceHistoryScreen } from '../screens/AttendanceHistoryScreen';
import { LeaveDetailScreen } from '../screens/LeaveDetailScreen';
import { LeaveFormScreen } from '../screens/LeaveFormScreen';
import { LeaveListScreen } from '../screens/LeaveListScreen';
import { NotificationsScreen } from '../screens/NotificationsScreen';
import { NotificationBell } from '../notifications/NotificationBell';
import { NotificationsProvider } from '../notifications/NotificationsProvider';
import { PermissionScreen } from '../screens/PermissionScreen';
import { ProfileScreen } from '../screens/ProfileScreen';
import { PermissionsProvider, usePermissions } from '../permissions/PermissionsProvider';
import { ProfileProvider } from '../profile/ProfileProvider';
import { useStyles, useTheme, type Theme } from '../theme/ThemeProvider';
import type { RootStackParamList } from './types';

const Stack = createNativeStackNavigator<RootStackParamList>();

export function RootNavigator() {
  const { status } = useAuth();
  const styles = useStyles(makeStyles);
  const { color } = useTheme();

  // Membaca token dari SecureStore saat aplikasi dibuka.
  if (status === 'loading') {
    return (
      <View style={styles.loading}>
        <ActivityIndicator color={color.text.primary} />
      </View>
    );
  }

  if (status !== 'signedIn') {
    return (
      <Stack.Navigator>
        <Stack.Screen name="Login" component={LoginScreen} options={{ headerShown: false }} />
        <Stack.Screen name="ForgotPassword" component={ForgotPasswordScreen} options={{ title: 'Lupa Kata Sandi' }} />
        <Stack.Screen name="CheckEmail" component={CheckEmailScreen} options={{ title: 'Lupa Kata Sandi' }} />
        <Stack.Screen name="NewPassword" component={NewPasswordScreen} options={{ title: 'Kata Sandi Baru' }} />
        <Stack.Screen
          name="PasswordChanged"
          component={PasswordChangedScreen}
          options={{ headerShown: false, gestureEnabled: false }}
        />
      </Stack.Navigator>
    );
  }

  // Profil dan status izin dimuat sekali setelah login, dipakai bersama oleh layar-layar di bawah.
  return (
    <ProfileProvider>
      <PermissionsProvider>
        <AttendanceProvider>
          <NotificationsProvider>
            <SignedInStack />
          </NotificationsProvider>
        </AttendanceProvider>
      </PermissionsProvider>
    </ProfileProvider>
  );
}

function SignedInStack() {
  const styles = useStyles(makeStyles);
  const { color } = useTheme();
  const { ready, introSeen } = usePermissions();

  if (!ready) {
    return (
      <View style={styles.loading}>
        <ActivityIndicator color={color.text.primary} />
      </View>
    );
  }

  return (
    // Penjelasan izin (UU PDP) tampil sekali setelah login pertama di perangkat ini.
    <Stack.Navigator initialRouteName={introSeen ? 'Home' : 'Permissions'}>
      <Stack.Screen
        name="Home"
        component={HomeScreen}
        options={{ title: 'Beranda', headerRight: () => <NotificationBell /> }}
      />
      <Stack.Screen name="Profile" component={ProfileScreen} options={{ title: 'Profil Saya' }} />
      <Stack.Screen name="ClockIn" component={ClockInScreen} options={{ title: 'Clock In' }} />
      <Stack.Screen name="ClockOut" component={ClockOutScreen} options={{ title: 'Clock Out' }} />
      <Stack.Screen name="AttendanceHistory" component={AttendanceHistoryScreen} options={{ title: 'Riwayat Absensi' }} />
      <Stack.Screen name="AttendanceDay" component={AttendanceDayScreen} options={{ title: 'Detail Absensi' }} />
      <Stack.Screen name="LeaveRequests" component={LeaveListScreen} options={{ title: 'Izin & Cuti' }} />
      <Stack.Screen name="LeaveForm" component={LeaveFormScreen} options={{ title: 'Ajukan Izin' }} />
      <Stack.Screen name="LeaveDetail" component={LeaveDetailScreen} options={{ title: 'Detail Pengajuan' }} />
      <Stack.Screen name="Notifications" component={NotificationsScreen} options={{ title: 'Notifikasi' }} />
      <Stack.Screen name="Permissions" component={PermissionScreen} options={{ title: 'Izin Aplikasi' }} />
      <Stack.Screen name="ApiStatus" component={ApiStatusScreen} options={{ title: 'Koneksi API' }} />
      {__DEV__ && (
        <Stack.Screen name="Components" component={ComponentsScreen} options={{ title: 'Contoh komponen' }} />
      )}
    </Stack.Navigator>
  );
}

const makeStyles = ({ color }: Theme) =>
  StyleSheet.create({
    loading: {
      flex: 1,
      alignItems: 'center',
      justifyContent: 'center',
      backgroundColor: color.bg.surface,
    },
  });
