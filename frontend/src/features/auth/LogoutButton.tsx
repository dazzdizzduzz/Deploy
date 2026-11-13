'use client';

import Image from 'next/image';
import { useState } from 'react';
import { requestLogout } from '@/lib/auth/logout';

type Props = {
  className?: string;
  /** Hanya ikon (bar atas di tablet/ponsel); label tetap dibacakan screen reader. */
  iconOnly?: boolean;
};

/** Tombol Keluar: hapus sesi di server dan cookie, lalu kembali ke halaman login. */
export function LogoutButton({ className, iconOnly = false }: Props) {
  const [busy, setBusy] = useState(false);

  async function logout() {
    setBusy(true);
    await requestLogout();
    window.location.assign('/login');
  }

  return (
    <button
      type="button"
      className={className}
      onClick={() => void logout()}
      disabled={busy}
      aria-busy={busy}
      aria-label={iconOnly ? 'Keluar' : undefined}
    >
      <Image src="/icons/logout.svg" alt="" width={20} height={20} unoptimized />
      {!iconOnly && (busy ? 'Keluar…' : 'Keluar')}
    </button>
  );
}
