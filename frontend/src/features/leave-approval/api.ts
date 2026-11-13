import { apiRequest } from '@/lib/api/client';

/** Mengikuti draf api/drafts/leave-approval.openapi.yaml (belum ada di kontrak utama). */
export type LeaveType = 'cuti' | 'izin' | 'sakit';
export type LeaveStatus = 'menunggu' | 'disetujui' | 'ditolak';

export type LeaveApprovalItem = {
  id: string;
  type: LeaveType;
  status: LeaveStatus;
  /** Tanggal kalender "YYYY-MM-DD". */
  start_date: string;
  end_date: string;
  working_days: number;
  reason: string;
  employee: { id: string; name: string; department_name?: string | null; position_name?: string | null };
  leave_quota_remaining?: number | null;
  attachment?: { file_name: string; content_type?: string; download_url?: string | null } | null;
  decided_by_name?: string | null;
  decided_at?: string | null;
  decision_note?: string | null;
  created_at: string;
};

export type LeaveApprovalList = {
  data: LeaveApprovalItem[];
  meta: { page: number; per_page: number; total: number; total_pages: number };
  counts: Record<LeaveStatus, number>;
};

export type LeaveApprovalQuery = {
  status?: LeaveStatus;
  type?: LeaveType;
  department_id?: string;
  q?: string;
  page?: number;
  per_page?: number;
};

export const leaveApprovalApi = {
  list: (query: LeaveApprovalQuery) => apiRequest<LeaveApprovalList>('leave-requests', { query }),
  get: (id: string) => apiRequest<LeaveApprovalItem>(`leave-requests/${id}`),
  approve: (id: string, note?: string) =>
    apiRequest<LeaveApprovalItem>(`leave-requests/${id}/approve`, { method: 'POST', body: note ? { note } : {} }),
  reject: (id: string, reason: string) =>
    apiRequest<LeaveApprovalItem>(`leave-requests/${id}/reject`, { method: 'POST', body: { reason } }),
};
