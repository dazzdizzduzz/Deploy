import { NextResponse, type NextRequest } from 'next/server';
import type { components } from '@/lib/api/schema';
import {
  NO_STORE,
  postUpstream,
  problem,
  problemFromUpstream,
  readJsonBody,
  unreachable,
} from '@/lib/api/upstream';

type ResetPasswordRequest = components['schemas']['ResetPasswordRequest'];
type MessageResponse = components['schemas']['MessageResponse'];

const titles: Record<number, string> = { 400: 'Tautan tidak dapat dipakai' };

export async function POST(request: NextRequest) {
  const parsed = await readJsonBody(request);
  if ('error' in parsed) return parsed.error;

  const { token, new_password: newPassword } = parsed.body;
  if (typeof token !== 'string' || typeof newPassword !== 'string' || !token || !newPassword) {
    return problem(400, 'Permintaan tidak valid', 'Token dan kata sandi baru wajib diisi.');
  }

  const upstream = await postUpstream('/auth/password/reset', {
    token,
    new_password: newPassword,
  } satisfies ResetPasswordRequest);
  if (!upstream) return unreachable();
  if (!upstream.ok) return problemFromUpstream(upstream, titles);

  const data = (await upstream.json().catch(() => null)) as Partial<MessageResponse> | null;
  return NextResponse.json({ message: data?.message ?? 'Kata sandi diperbarui.' }, { headers: NO_STORE });
}
