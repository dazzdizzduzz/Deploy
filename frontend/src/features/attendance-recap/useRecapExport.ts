'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { ApiProblem } from '@/lib/api/client';
import { recapApi, type ExportInput, type RecapExport } from './api';

const POLL_MS = 2000;
/** Berhenti mengecek setelah ±1 menit; pengguna bisa mengecek ulang secara manual. */
const MAX_POLLS = 30;

export type ExportState =
  | { kind: 'idle' }
  | { kind: 'processing'; id: string }
  | { kind: 'ready'; file: RecapExport }
  | { kind: 'slow'; id: string }
  | { kind: 'failed'; message: string };

/**
 * Ekspor PDF di server (FR-11): POST membuat tugas (202), lalu status dicek berkala sampai
 * tautan unduhan siap.
 */
export function useRecapExport() {
  const [state, setState] = useState<ExportState>({ kind: 'idle' });
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const polls = useRef(0);

  const stop = useCallback(() => {
    if (timer.current) clearTimeout(timer.current);
    timer.current = null;
  }, []);

  useEffect(() => stop, [stop]);

  const poll = useCallback(
    (id: string) => {
      stop();
      timer.current = setTimeout(async () => {
        polls.current += 1;
        try {
          const file = await recapApi.getExport(id);
          if (file.status === 'ready' && file.download_url) setState({ kind: 'ready', file });
          else if (file.status === 'failed') setState({ kind: 'failed', message: file.error || 'Server gagal membuat PDF.' });
          else if (polls.current >= MAX_POLLS) setState({ kind: 'slow', id });
          else poll(id);
        } catch (error) {
          setState({ kind: 'failed', message: error instanceof ApiProblem ? error.message : 'Status ekspor tidak dapat dicek.' });
        }
      }, POLL_MS);
    },
    [stop],
  );

  const start = useCallback(
    async (input: ExportInput) => {
      stop();
      polls.current = 0;
      try {
        const job = await recapApi.createExport(input);
        if (job.status === 'ready' && job.download_url) {
          setState({ kind: 'ready', file: job });
          return;
        }
        setState({ kind: 'processing', id: job.id });
        poll(job.id);
      } catch (error) {
        const notAvailable = error instanceof ApiProblem && (error.status === 404 || error.status === 405);
        setState({
          kind: 'failed',
          message: notAvailable
            ? 'Ekspor PDF belum tersedia di server. Gunakan "Cetak / simpan PDF" sebagai gantinya.'
            : error instanceof ApiProblem
              ? error.message
              : 'Ekspor belum bisa dimulai.',
        });
      }
    },
    [poll, stop],
  );

  const recheck = useCallback(
    (id: string) => {
      polls.current = 0;
      setState({ kind: 'processing', id });
      poll(id);
    },
    [poll],
  );

  const reset = useCallback(() => {
    stop();
    setState({ kind: 'idle' });
  }, [stop]);

  return { state, start, recheck, reset };
}
