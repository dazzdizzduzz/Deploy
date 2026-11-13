import { api } from './client';

/**
 * Path dari api/openapi-phase1.yaml (Izin dan Cuti F06); bentuk respons dan endpoint unggah lampiran
 * mengikuti draf api/drafts/leave-requests.openapi.yaml.
 */
export type LeaveType = 'cuti' | 'izin' | 'sakit';
export type LeaveStatus = 'menunggu' | 'disetujui' | 'ditolak';

export type LeaveRequest = {
  id: string;
  type: LeaveType;
  status: LeaveStatus;
  /** Tanggal kalender "YYYY-MM-DD". */
  start_date: string;
  end_date: string;
  working_days: number;
  reason: string;
  attachment?: { file_name: string; download_url?: string | null } | null;
  approver_name?: string | null;
  decided_at?: string | null;
  decision_note?: string | null;
  created_at: string;
};

export type LeaveQuota = {
  year: number;
  total_days: number;
  used_days: number;
  pending_days: number;
  remaining_days: number;
};

export type LeavePreview = {
  working_days: number;
  deducts_quota: boolean;
  remaining_days_after?: number | null;
  quota_sufficient: boolean;
  doctor_note_required: boolean;
  overlapping?: boolean;
};

export type LeaveAttachment = { attachment_url: string; file_name: string; content_type: string; size_bytes: number };

type Page<T> = { data: T[]; meta: { page: number; per_page: number; total: number; total_pages: number } };

export const getLeaveQuota = () => api.get<LeaveQuota>('/me/leave-quota');

export function listLeaveRequests(status?: LeaveStatus, page = 1) {
  const query = new URLSearchParams({ page: String(page), per_page: '20' });
  if (status) query.set('status', status);
  return api.get<Page<LeaveRequest>>(`/me/leave-requests?${query.toString()}`);
}

export const getLeaveRequest = (id: string) => api.get<LeaveRequest>(`/me/leave-requests/${encodeURIComponent(id)}`);

type Range = { type: LeaveType; start_date: string; end_date: string };

export const previewLeaveRequest = (input: Range) => api.post<LeavePreview>('/me/leave-requests/preview', input);

export const createLeaveRequest = (input: Range & { reason: string; attachment_url?: string }) =>
  api.post<LeaveRequest>('/me/leave-requests', input);

export type PickedFile = { uri: string; name: string; mimeType: string; size?: number };

/** Unggah surat dokter (multipart) sebelum mengirim pengajuan. */
export function uploadLeaveAttachment(file: PickedFile) {
  const form = new FormData();
  // React Native mengirim berkas lewat objek { uri, name, type }.
  form.append('file', { uri: file.uri, name: file.name, type: file.mimeType } as unknown as Blob);
  return api.postForm<LeaveAttachment>('/me/leave-requests/attachments', form);
}
