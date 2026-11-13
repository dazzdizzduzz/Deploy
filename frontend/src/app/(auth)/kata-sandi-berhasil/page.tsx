import type { Metadata } from 'next';
import { PasswordChanged } from '@/features/auth/PasswordChanged';

export const metadata: Metadata = { title: 'Kata Sandi Diubah · On-Track' };

export default function PasswordChangedPage() {
  return <PasswordChanged />;
}
