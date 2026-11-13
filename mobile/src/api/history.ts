import type { Attendance } from './attendance';
import { api } from './client';

/** Sesuai draf api/drafts/me-attendance.openapi.yaml (GET /me/attendance?month=YYYY-MM). */
export type DayStatus = 'on_time' | 'late' | 'izin' | 'cuti' | 'sakit' | 'alpha' | 'libur';

export type AttendanceDay = {
  date: string;
  status: DayStatus;
  attendance?: Attendance | null;
  office_location_name?: string | null;
  leave?: { type: 'izin' | 'cuti' | 'sakit'; reason?: string | null } | null;
};

export type MonthlySummary = {
  hadir: number;
  terlambat: number;
  izin: number;
  cuti: number;
  sakit: number;
  alpha: number;
  total_work_minutes: number;
  overtime_minutes: number;
};

export type MonthlyAttendance = { month: string; days: AttendanceDay[]; summary: MonthlySummary };

export const getMonthlyAttendance = (month: string) =>
  api.get<MonthlyAttendance>(`/me/attendance?month=${encodeURIComponent(month)}`);
