import { api } from './client';

/** Sesuai backend/api/openapi.yaml (AttendanceResponse, OfficeLocationResponse, WorkScheduleResponse). */
export type AttendanceStatus = 'on_time' | 'late' | 'absent';
export type WorkMode = 'office' | 'wfh' | 'business_trip';

export type Attendance = {
  id: string;
  office_location_id?: string | null;
  work_schedule_id?: string | null;
  /** Tanggal kerja di zona waktu tenant ("2026-11-12"). */
  work_date: string;
  work_mode: WorkMode;
  /** ISO 8601 UTC. */
  clock_in: string;
  clock_out?: string | null;
  latitude_in: number;
  longitude_in: number;
  distance_in_meters?: number | null;
  distance_out_meters?: number | null;
  status: AttendanceStatus;
  late_minutes: number;
  early_leave_minutes: number;
  overtime_minutes: number;
  has_selfie_in: boolean;
  has_selfie_out: boolean;
};

export type OfficeLocation = {
  id: string;
  name: string;
  address: string;
  latitude: number;
  longitude: number;
  radius_meters: number;
  selfie_required: boolean;
  is_active: boolean;
};

export type WorkSchedule = {
  id: string;
  name: string;
  start_time: string;
  end_time: string;
  is_active: boolean;
  allow_remote: boolean;
};

export const listLocations = () => api.get<{ data: OfficeLocation[] }>('/locations');
export const listSchedules = () => api.get<{ data: WorkSchedule[] }>('/schedules');

export type ClockInInput = {
  latitude: number;
  longitude: number;
  workScheduleId?: string;
  /** Berkas JPEG hasil kamera (uri lokal). */
  selfieUri?: string;
};

/** POST /attendances/clock-in (multipart/form-data). Mode kerja `office`; server memeriksa geofence. */
export function clockIn({ latitude, longitude, workScheduleId, selfieUri }: ClockInInput) {
  const form = new FormData();
  form.append('latitude', String(latitude));
  form.append('longitude', String(longitude));
  form.append('work_mode', 'office');
  if (workScheduleId) form.append('work_schedule_id', workScheduleId);
  if (selfieUri) {
    // React Native mengirim berkas lewat objek { uri, name, type }.
    form.append('selfie', { uri: selfieUri, name: 'selfie.jpg', type: 'image/jpeg' } as unknown as Blob);
  }
  return api.postForm<Attendance>('/attendances/clock-in', form);
}

export type ClockOutInput = { latitude: number; longitude: number; selfieUri?: string };

/** POST /attendances/clock-out (multipart/form-data). Server menghitung lembur dan pulang cepat. */
export function clockOut({ latitude, longitude, selfieUri }: ClockOutInput) {
  const form = new FormData();
  form.append('latitude', String(latitude));
  form.append('longitude', String(longitude));
  if (selfieUri) {
    form.append('selfie', { uri: selfieUri, name: 'selfie.jpg', type: 'image/jpeg' } as unknown as Blob);
  }
  return api.postForm<Attendance>('/attendances/clock-out', form);
}
