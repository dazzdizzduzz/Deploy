import { ApiError, api } from './client';

/**
 * Respons GET /auth/me sesuai draf usulan api/drafts/auth-me.openapi.yaml.
 * Endpoint ini belum ada di backend; bila 404, data diambil dari GET /v1/me yang sudah ada.
 */
type NamedRef = { id: string; name: string };

type AuthMeResponse = {
  id: string;
  email: string;
  full_name?: string;
  /** Kontrak api/openapi-phase1.yaml memakai `name`. */
  name?: string;
  role: string;
  phone?: string | null;
  employment_type?: string | null;
  joined_at?: string | null;
  position?: NamedRef | string | null;
  department?: NamedRef | string | null;
  tenant?: { id: string; name: string; timezone?: string } | null;
};

/** GET /v1/me saat ini: hanya isi klaim token. */
type TokenMeResponse = { id: string; tenant_id: string; role: string };

export type Profile = {
  id: string;
  role: string;
  email: string | null;
  fullName: string | null;
  position: string | null;
  department: string | null;
  tenantName: string | null;
  phone: string | null;
  employmentType: string | null;
  joinedAt: string | null;
  /** `token`: profil lengkap belum tersedia, hanya data dari klaim token. */
  source: 'auth-me' | 'token';
};

const nameOf = (value: NamedRef | string | null | undefined) =>
  typeof value === 'string' ? value : (value?.name ?? null);

function fromAuthMe(data: AuthMeResponse): Profile {
  return {
    id: data.id,
    role: data.role,
    email: data.email ?? null,
    fullName: data.full_name ?? data.name ?? null,
    position: nameOf(data.position),
    department: nameOf(data.department),
    tenantName: data.tenant?.name ?? null,
    phone: data.phone ?? null,
    employmentType: data.employment_type ?? null,
    joinedAt: data.joined_at ?? null,
    source: 'auth-me',
  };
}

function fromToken(data: TokenMeResponse): Profile {
  return {
    id: data.id,
    role: data.role,
    email: null,
    fullName: null,
    position: null,
    department: null,
    tenantName: null,
    phone: null,
    employmentType: null,
    joinedAt: null,
    source: 'token',
  };
}

export async function getProfile(): Promise<Profile> {
  try {
    return fromAuthMe(await api.get<AuthMeResponse>('/auth/me'));
  } catch (error) {
    if (!(error instanceof ApiError) || (error.status !== 404 && error.status !== 405)) throw error;
    return fromToken(await api.get<TokenMeResponse>('/v1/me'));
  }
}
