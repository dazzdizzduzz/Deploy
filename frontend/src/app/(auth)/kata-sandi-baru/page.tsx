import type { Metadata } from 'next';
import { InvalidResetLink, NewPasswordForm } from '@/features/auth/NewPasswordForm';

// Token ada di URL; jangan bocorkan lewat header Referer ke situs lain.
export const metadata: Metadata = {
  title: 'Kata Sandi Baru · On-Track',
  referrer: 'no-referrer',
  robots: { index: false },
};

type PageProps = { searchParams: Promise<{ token?: string | string[] }> };

export default async function NewPasswordPage({ searchParams }: PageProps) {
  const { token } = await searchParams;
  return typeof token === 'string' && token ? <NewPasswordForm token={token} /> : <InvalidResetLink />;
}
