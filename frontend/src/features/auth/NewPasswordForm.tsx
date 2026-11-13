'use client';

import Image from 'next/image';
import { useRouter } from 'next/navigation';
import { useState, type FormEvent } from 'react';
import { Alert } from '@/components/ui/Alert';
import { Button, ButtonLink } from '@/components/ui/Button';
import { PasswordField } from '@/components/ui/Field';
import { postJson } from '@/lib/api/post-json';
import type { Problem } from '@/lib/api/problem';
import { checkPassword } from '@/lib/auth/password';
import { AuthHeading, AuthIconBadge, AuthStack } from './AuthParts';
import styles from './NewPasswordForm.module.css';

function Rule({ label, met }: { label: string; met: boolean }) {
  return (
    <li className={met ? styles.rule : `${styles.rule} ${styles.ruleUnmet}`}>
      <Image
        src={met ? '/icons/check-rule.svg' : '/icons/close.svg'}
        alt={met ? 'Terpenuhi:' : 'Belum terpenuhi:'}
        width={16}
        height={16}
        unoptimized
      />
      {label}
    </li>
  );
}

/** Tautan reset tidak ada, rusak, atau sudah ditolak backend. */
export function InvalidResetLink({ problem }: { problem?: Problem }) {
  return (
    <AuthStack>
      <AuthIconBadge icon="key" />
      <AuthHeading title="Tautan tidak dapat dipakai">
        Tautan reset kata sandi tidak valid, sudah kedaluwarsa, atau sudah pernah dipakai.
      </AuthHeading>
      {problem?.detail && (
        <Alert tone="danger" title={problem.title}>
          {problem.detail}
        </Alert>
      )}
      <ButtonLink fullWidth href="/lupa-kata-sandi">
        Minta tautan baru
        <Image src="/icons/arrow-right-inverse.svg" alt="" width={20} height={20} unoptimized />
      </ButtonLink>
    </AuthStack>
  );
}

/** W21 · Buat Kata Sandi Baru. `token` berasal dari tautan di email (`?token=`). */
export function NewPasswordForm({ token }: { token: string }) {
  const router = useRouter();
  const [password, setPassword] = useState('');
  const [confirmation, setConfirmation] = useState('');
  const [submitted, setSubmitted] = useState(false);
  const [pending, setPending] = useState(false);
  const [problem, setProblem] = useState<Problem | null>(null);

  const { minLength, lettersAndDigits, uppercase } = checkPassword(password);
  const passwordValid = minLength && lettersAndDigits && uppercase;
  const matches = password === confirmation;

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    if (pending) return;
    setSubmitted(true);
    if (!passwordValid || !matches) return;

    setPending(true);
    setProblem(null);
    const result = await postJson('/api/auth/password/reset', { token, new_password: password });
    if (result.ok) {
      router.replace('/kata-sandi-berhasil');
      return;
    }
    setPending(false);
    setProblem(result.problem);
  }

  // Token ditolak backend: form tidak bisa dipakai lagi, minta tautan baru.
  if (problem?.status === 400) return <InvalidResetLink problem={problem} />;

  return (
    <form onSubmit={handleSubmit} noValidate>
      <AuthStack>
        <AuthIconBadge icon="lock" />
        <AuthHeading title="Buat kata sandi baru">
          Pilih kata sandi baru untuk akun kamu.
        </AuthHeading>
        {problem && (
          <Alert tone="danger" title={problem.title}>
            {problem.detail ?? 'Coba lagi beberapa saat lagi.'}
          </Alert>
        )}
        <PasswordField
          label="Kata sandi baru"
          helper="Minimal 8 karakter"
          error={submitted && !passwordValid ? 'Kata sandi belum memenuhi syarat' : undefined}
          name="password"
          autoComplete="new-password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
        />
        <PasswordField
          label="Ulangi kata sandi"
          helper="Harus sama dengan kata sandi baru"
          error={submitted && !matches ? 'Kata sandi tidak sama' : undefined}
          name="confirmation"
          autoComplete="new-password"
          value={confirmation}
          onChange={(event) => setConfirmation(event.target.value)}
        />
        <ul className={styles.rules}>
          <Rule label="Minimal 8 karakter" met={minLength} />
          <Rule label="Mengandung huruf dan angka" met={lettersAndDigits} />
          <Rule label="Mengandung huruf besar" met={uppercase} />
        </ul>
        <Button type="submit" fullWidth disabled={pending}>
          {pending ? 'Menyimpan…' : 'Simpan kata sandi'}
        </Button>
      </AuthStack>
    </form>
  );
}
