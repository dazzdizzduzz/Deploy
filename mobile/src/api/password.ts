import { api } from './client';

/** Sesuai ForgotPasswordRequest, ResetPasswordRequest, dan MessageResponse di backend/api/openapi.yaml. */
type MessageResponse = { message: string };

/** 202: backend tidak memberi tahu apakah email terdaftar. */
export const requestPasswordReset = (email: string) =>
  api.post<MessageResponse>('/auth/password/forgot', { email });

export const resetPassword = (token: string, newPassword: string) =>
  api.post<MessageResponse>('/auth/password/reset', { token, new_password: newPassword });
