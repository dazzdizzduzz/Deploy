import { api } from './client';

export type HealthResponse = { status: string };

/** GET /health (lihat backend/api/openapi.yaml). */
export const getHealth = (signal?: AbortSignal) => api.get<HealthResponse>('/health', signal);
