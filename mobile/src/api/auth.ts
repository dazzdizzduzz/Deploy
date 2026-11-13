import { api } from './client';

/** Sesuai `LoginRequest` dan `LoginResponse` di backend/api/openapi.yaml. */
export type LoginRequest = { email: string; password: string };
export type LoginResponse = { access_token: string; token_type: string; expires_in: number };

export const login = (body: LoginRequest) => api.post<LoginResponse>('/auth/login', body);
