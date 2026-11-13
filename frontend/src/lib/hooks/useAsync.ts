'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { ApiProblem } from '@/lib/api/client';

type AsyncState<T> = { data: T | null; error: ApiProblem | null; loading: boolean };

/**
 * Menjalankan `load` saat `deps` berubah dan menyimpan hasilnya. Respons lama diabaikan bila
 * permintaan baru sudah dikirim, agar hasil pencarian tidak tertukar.
 */
export function useAsync<T>(load: () => Promise<T>, deps: readonly unknown[]) {
  const [state, setState] = useState<AsyncState<T>>({ data: null, error: null, loading: true });
  const requestId = useRef(0);

  // eslint-disable-next-line react-hooks/exhaustive-deps
  const run = useCallback(load, deps);

  const reload = useCallback(async () => {
    const id = ++requestId.current;
    setState((current) => ({ ...current, loading: true, error: null }));
    try {
      const data = await run();
      if (id === requestId.current) setState({ data, error: null, loading: false });
    } catch (error) {
      if (id !== requestId.current) return;
      const problem =
        error instanceof ApiProblem
          ? error
          : new ApiProblem({ type: 'about:blank', title: 'Terjadi kesalahan', status: 0 });
      setState((current) => ({ ...current, error: problem, loading: false }));
    }
  }, [run]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return { ...state, reload };
}
