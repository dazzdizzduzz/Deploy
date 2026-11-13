import { apiRequest } from '@/lib/api/client';

/** Sesuai draf api/drafts/attendance-recap.openapi.yaml (belum ada di backend). */
export type TodayStatus = 'hadir' | 'terlambat' | 'izin' | 'alpha' | 'belum';

export type RecapRow = {
  user_id: string;
  name: string;
  department_name?: string | null;
  hadir: number;
  terlambat: number;
  izin: number;
  cuti: number;
  sakit: number;
  alpha: number;
  total_work_minutes: number;
  overtime_minutes: number;
  today_status?: TodayStatus | null;
};

export type RecapResponse = {
  month: string;
  data_until?: string | null;
  stats: { hadir: number; terlambat: number; izin: number; alpha: number; employees: number };
  data: RecapRow[];
  meta: { page: number; per_page: number; total: number; total_pages: number };
};

export type RecapExport = {
  id: string;
  status: 'processing' | 'ready' | 'failed';
  file_name?: string | null;
  download_url?: string | null;
  expires_at?: string | null;
  error?: string | null;
};

export type RecapQuery = {
  month: string;
  department_id?: string;
  today_status?: TodayStatus;
  page?: number;
  per_page?: number;
};

export type ExportInput = {
  month: string;
  department_id?: string | null;
  include_daily: boolean;
  include_selfie: boolean;
};

export const recapApi = {
  get: (query: RecapQuery) => apiRequest<RecapResponse>('attendance/recap', { query }),
  createExport: (body: ExportInput) =>
    apiRequest<RecapExport>('attendance/recap/exports', { method: 'POST', body }),
  getExport: (id: string) => apiRequest<RecapExport>(`attendance/recap/exports/${id}`),
};
