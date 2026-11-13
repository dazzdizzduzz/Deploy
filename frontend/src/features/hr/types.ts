/**
 * Tipe data SDM. Ditulis manual karena `api/openapi-phase1.yaml` belum bisa di-generate
 * (beberapa `$ref` belum terdefinisi).
 *
 * - Department, Position: `api/openapi-phase1.yaml`.
 * - User*: kontrak backend PR "API Karyawan (FR-03)" (backend/api/openapi.yaml).
 *
 * Ganti dengan tipe hasil `npm run generate:api` setelah kontrak tersebut masuk ke staging.
 */

export type PaginationMeta = { page: number; per_page: number; total: number; total_pages: number };

export type ListResponse<T> = { data: T[]; meta: PaginationMeta };

export type Department = {
  id: string;
  tenant_id: string;
  name: string;
  description?: string | null;
  employee_count?: number;
  created_at: string;
  updated_at?: string;
};

export type DepartmentInput = { name: string; description?: string };

export type Position = {
  id: string;
  tenant_id: string;
  department_id?: string | null;
  department_name?: string | null;
  name: string;
  description?: string | null;
  created_at: string;
  updated_at?: string;
};

export type PositionInput = { name: string; department_id?: string | null; description?: string };

export const EMPLOYEE_ROLES = ['owner', 'hrd', 'team_lead', 'employee', 'cashier'] as const;
export type EmployeeRole = (typeof EMPLOYEE_ROLES)[number];

export const EMPLOYMENT_TYPES = ['tetap', 'kontrak', 'magang'] as const;
export type EmploymentType = (typeof EMPLOYMENT_TYPES)[number];

export const USER_STATUSES = ['aktif', 'nonaktif', 'diundang'] as const;
export type UserStatus = (typeof USER_STATUSES)[number];

export type User = {
  id: string;
  tenant_id: string;
  name: string;
  email: string;
  phone?: string | null;
  department_id?: string | null;
  department_name?: string | null;
  position_id?: string | null;
  position_name?: string | null;
  role: EmployeeRole | 'super_admin';
  employment_type: EmploymentType;
  status: UserStatus;
  joined_at?: string | null;
  created_at: string;
  updated_at: string;
};

export type CreateUserInput = {
  name: string;
  email: string;
  phone?: string;
  department_id?: string;
  position_id?: string;
  role: EmployeeRole;
  employment_type: EmploymentType;
  status?: 'aktif' | 'diundang';
  joined_at?: string;
};

export type UpdateUserInput = {
  name?: string;
  phone?: string | null;
  department_id?: string | null;
  position_id?: string | null;
  role?: EmployeeRole;
  employment_type?: EmploymentType;
  joined_at?: string | null;
};
