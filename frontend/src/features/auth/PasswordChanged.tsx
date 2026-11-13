import Image from 'next/image';
import { ButtonLink } from '@/components/ui/Button';
import { AuthHeading, AuthIconBadge, AuthStack } from './AuthParts';

/** W22 · Kata Sandi Berhasil Diubah */
export function PasswordChanged() {
  return (
    <AuthStack>
      <AuthIconBadge icon="check-badge" tone="accent" />
      <AuthHeading title="Kata sandi berhasil diubah">
        Silakan masuk dengan kata sandi baru.
      </AuthHeading>
      <ButtonLink fullWidth href="/login">
        Masuk ke dashboard
        <Image src="/icons/arrow-right-inverse.svg" alt="" width={20} height={20} unoptimized />
      </ButtonLink>
    </AuthStack>
  );
}
