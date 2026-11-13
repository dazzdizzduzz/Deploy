import type { AttendanceDay } from '../api/history';
import type { LeaveRequest } from '../api/leave';

export type RootStackParamList = {
  Login: undefined;
  ForgotPassword: { email?: string } | undefined;
  CheckEmail: { email: string };
  /** `token` dari tautan ontrack://kata-sandi-baru?token=... */
  NewPassword: { token?: string } | undefined;
  PasswordChanged: undefined;
  Home: undefined;
  Profile: undefined;
  ClockIn: undefined;
  ClockOut: undefined;
  AttendanceHistory: undefined;
  AttendanceDay: { day: AttendanceDay };
  LeaveRequests: undefined;
  Notifications: undefined;
  LeaveForm: undefined;
  /** `request` dari daftar dipakai selagi detail dimuat; `created` setelah pengajuan terkirim. */
  LeaveDetail: { id: string; request?: LeaveRequest; created?: boolean };
  /** Penjelasan & permintaan izin lokasi dan kamera. `from: 'profile'` saat dibuka dari Profil Saya. */
  Permissions: { from?: 'intro' | 'profile' } | undefined;
  ApiStatus: undefined;
  /** Hanya di mode development: etalase komponen dasar. */
  Components: undefined;
};
