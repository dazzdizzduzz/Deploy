import { apiRequest } from '@/lib/api/client';
import type { components } from '@/lib/api/schema';

type Schemas = components['schemas'];

export type OfficeLocation = Schemas['OfficeLocationResponse'];
export type CreateLocationInput = Schemas['CreateLocationRequest'];
export type UpdateLocationInput = Schemas['UpdateLocationRequest'];

export type WorkSchedule = Schemas['WorkScheduleResponse'];
export type CreateScheduleInput = Schemas['CreateScheduleRequest'];
export type UpdateScheduleInput = Schemas['UpdateScheduleRequest'];

export const locationsApi = {
  list: () => apiRequest<Schemas['OfficeLocationsListResponse']>('locations'),
  create: (body: CreateLocationInput) => apiRequest<OfficeLocation>('locations', { method: 'POST', body }),
  update: (id: string, body: UpdateLocationInput) =>
    apiRequest<OfficeLocation>(`locations/${id}`, { method: 'PATCH', body }),
  remove: (id: string) => apiRequest<Schemas['MessageResponse']>(`locations/${id}`, { method: 'DELETE' }),
};

export const schedulesApi = {
  list: () => apiRequest<Schemas['WorkSchedulesListResponse']>('schedules'),
  create: (body: CreateScheduleInput) => apiRequest<WorkSchedule>('schedules', { method: 'POST', body }),
  update: (id: string, body: UpdateScheduleInput) =>
    apiRequest<WorkSchedule>(`schedules/${id}`, { method: 'PATCH', body }),
  remove: (id: string) => apiRequest<Schemas['MessageResponse']>(`schedules/${id}`, { method: 'DELETE' }),
};
