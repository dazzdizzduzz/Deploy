import { api } from './client';

/** Mengikuti draf api/drafts/notifications.openapi.yaml (belum ada di kontrak utama). */
export type NotificationType = 'leave_approved' | 'leave_rejected';

export type AppNotification = {
  id: string;
  /** Jenis yang belum dikenal aplikasi tetap ditampilkan, hanya tanpa tujuan saat di-tap. */
  type: NotificationType | (string & {});
  title: string;
  body: string;
  leave_request_id?: string | null;
  read_at?: string | null;
  created_at: string;
};

type Page<T> = { data: T[]; meta: { page: number; per_page: number; total: number; total_pages: number } };

export const listNotifications = (page = 1) =>
  api.get<Page<AppNotification> & { unread_count: number }>(`/me/notifications?page=${page}&per_page=20`);

export const getUnreadCount = () => api.get<{ unread_count: number }>('/me/notifications/unread-count');

export const markNotificationRead = (id: string) => api.post<void>(`/me/notifications/${encodeURIComponent(id)}/read`);

export const markAllNotificationsRead = () => api.post<void>('/me/notifications/read-all');

export const registerDevice = (input: { token: string; platform: 'android' | 'ios'; device_name?: string }) =>
  api.post<void>('/me/devices', input);

export const unregisterDevice = (token: string) => api.post<void>('/me/devices/unregister', { token });
