'use client';

import Image from 'next/image';
import { useEffect, useState, type FormEvent } from 'react';
import { Alert } from '@/components/ui/Alert';
import { Button } from '@/components/ui/Button';
import { TextField } from '@/components/ui/Field';
import type { Problem } from '@/lib/api/problem';
import { postJson } from '@/lib/api/post-json';
import {
  RESEND_RESET_SECONDS,
  RESET_TOKEN_MINUTES,
  formatCountdown,
  maskEmail,
} from '@/lib/auth/password';
import { AuthHeading, AuthIconBadge, AuthNote, AuthStack, BackToLoginLink } from './AuthParts';

type State = { step: 'form' | 'sent'; email: string; problem?: Problem };

/** W19 · Lupa Kata Sandi, lalu W20 · Cek Email setelah permintaan diterima. */
export function ForgotPasswordForm() {
  const [state, setState] = useState<State>({ step: 'form', email: '' });
  const [pending, setPending] = useState(false);
  const [seconds, setSeconds] = useState(0);
  const counting = seconds > 0;

  useEffect(() => {
    if (!counting) return;
    const timer = setInterval(() => setSeconds((value) => Math.max(0, value - 1)), 1000);
    return () => clearInterval(timer);
  }, [counting]);

  async function request(email: string) {
    if (pending) return;
    setPending(true);
    const result = await postJson('/api/auth/password/forgot', { email });
    setPending(false);
    if (result.ok) {
      setSeconds(RESEND_RESET_SECONDS);
      setState({ step: 'sent', email });
    } else {
      setState((current) => ({ ...current, email, problem: result.problem }));
    }
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const email = String(new FormData(event.currentTarget).get('email') ?? '').trim();
    void request(email);
  }

  const { email, problem } = state;

  if (state.step === 'sent') {
    return (
      <AuthStack>
        <BackToLoginLink />
        <AuthIconBadge icon="mail" />
        <AuthHeading title="Cek email kamu">
          Jika {maskEmail(email)} terdaftar, tautan untuk membuat kata sandi baru sudah dikirim.
        </AuthHeading>
        {problem && (
          <Alert tone="danger" title={problem.title}>
            {problem.detail ?? 'Coba lagi beberapa saat lagi.'}
          </Alert>
        )}
        <Alert tone="info" title="Belum menerima email?">
          Cek folder Spam/Promosi. Tautan kedaluwarsa dalam {RESET_TOKEN_MINUTES} menit.
        </Alert>
        {counting ? (
          <Button fullWidth disabled>
            Kirim ulang dalam {formatCountdown(seconds)}
          </Button>
        ) : (
          <Button
            fullWidth
            variant="secondary"
            disabled={pending}
            onClick={() => void request(email)}
          >
            {pending ? 'Memproses…' : 'Kirim ulang tautan'}
          </Button>
        )}
      </AuthStack>
    );
  }

  const submitting = pending;

  return (
    <form onSubmit={handleSubmit}>
      <AuthStack>
        <BackToLoginLink />
        <AuthIconBadge icon="key" />
        <AuthHeading title="Lupa kata sandi?">
          Masukkan email kantor yang terdaftar. Kami akan mengirim tautan untuk membuat kata sandi
          baru.
        </AuthHeading>
        {problem && (
          <Alert tone="danger" title={problem.title}>
            {problem.detail ?? 'Coba lagi beberapa saat lagi.'}
          </Alert>
        )}
        <TextField
          label="Email kantor"
          helper={`Tautan berlaku ${RESET_TOKEN_MINUTES} menit`}
          name="email"
          type="email"
          autoComplete="username"
          required
        />
        <Button type="submit" fullWidth disabled={submitting}>
          {submitting ? 'Memproses…' : 'Kirim tautan reset'}
          {!submitting && <Image src="/icons/send.svg" alt="" width={20} height={20} unoptimized />}
        </Button>
        <AuthNote>Tidak punya akses ke email kantor? Hubungi HRD atau Owner tenant.</AuthNote>
      </AuthStack>
    </form>
  );
}
