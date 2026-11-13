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

type ForgotPasswordRequest = components['schemas']['ForgotPasswordRequest'];
type MessageResponse = components['schemas']['MessageResponse'];

const titles: Record<number, string> = { 400: 'Email tidak valid' };

export async function POST(request: NextRequest) {
  const parsed = await readJsonBody(request);
  if ('error' in parsed) return parsed.error;

  const { email } = parsed.body;
  if (typeof email !== 'string' || !email.trim()) {
    return problem(400, titles[400], 'Email wajib diisi.');
  }

  const upstream = await postUpstream('/auth/password/forgot', {
    email: email.trim(),
  } satisfies ForgotPasswordRequest);
  if (!upstream) return unreachable();
  if (!upstream.ok) return problemFromUpstream(upstream, titles);

  // Backend membalas 202 tanpa memberi tahu apakah email terdaftar; jangan menebak di sini.
  const data = (await upstream.json().catch(() => null)) as Partial<MessageResponse> | null;
  return NextResponse.json({ message: data?.message ?? 'Permintaan diterima.' }, { status: 202, headers: NO_STORE });
}
