import { apiRequest } from '@/lib/api/client';
import type {
  CreateUserInput,
  Department,
  DepartmentInput,
  ListResponse,
  Position,
  PositionInput,
  UpdateUserInput,
  User,
} from './types';

type ListQuery = Record<string, string | number | undefined>;

export const departmentsApi = {
  list: (query?: ListQuery) => apiRequest<ListResponse<Department>>('departments', { query }),
  create: (body: DepartmentInput) => apiRequest<Department>('departments', { method: 'POST', body }),
  update: (id: string, body: DepartmentInput) =>
    apiRequest<Department>(`departments/${id}`, { method: 'PATCH', body }),
  remove: (id: string) => apiRequest<{ success: boolean }>(`departments/${id}`, { method: 'DELETE' }),
};

export const positionsApi = {
  list: (query?: ListQuery) => apiRequest<ListResponse<Position>>('positions', { query }),
  create: (body: PositionInput) => apiRequest<Position>('positions', { method: 'POST', body }),
  update: (id: string, body: PositionInput) =>
    apiRequest<Position>(`positions/${id}`, { method: 'PATCH', body }),
  remove: (id: string) => apiRequest<{ success: boolean }>(`positions/${id}`, { method: 'DELETE' }),
};

export const usersApi = {
  list: (query?: ListQuery) => apiRequest<ListResponse<User>>('users', { query }),
  get: (id: string) => apiRequest<User>(`users/${id}`),
  create: (body: CreateUserInput) => apiRequest<User>('users', { method: 'POST', body }),
  update: (id: string, body: UpdateUserInput) => apiRequest<User>(`users/${id}`, { method: 'PATCH', body }),
};
