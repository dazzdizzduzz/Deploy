import type { Metadata } from 'next';
import { ForgotPasswordForm } from '@/features/auth/ForgotPasswordForm';

export const metadata: Metadata = { title: 'Lupa Kata Sandi · On-Track' };

export default function ForgotPasswordPage() {
  return <ForgotPasswordForm />;
}
