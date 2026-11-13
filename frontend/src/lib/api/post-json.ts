import { isProblem, type Problem } from '@/lib/api/problem';

export type PostResult = { ok: true } | { ok: false; problem: Problem };

const networkProblem: Problem = {
  type: 'about:blank',
  title: 'Tidak dapat terhubung',
  status: 0,
  detail: 'Periksa koneksi internet lalu coba lagi.',
};

/** POST JSON ke Route Handler milik aplikasi ini. Galat selalu berbentuk problem+json. */
export async function postJson(path: string, body: unknown): Promise<PostResult> {
  try {
    const response = await fetch(path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'same-origin',
      body: JSON.stringify(body),
    });
    if (response.ok) return { ok: true };

    const data: unknown = await response.json().catch(() => null);
    return {
      ok: false,
      problem: isProblem(data)
        ? data
        : { ...networkProblem, title: 'Permintaan gagal', status: response.status },
    };
  } catch {
    return { ok: false, problem: networkProblem };
  }
}
