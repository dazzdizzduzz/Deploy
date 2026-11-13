'use client';

import Image from 'next/image';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Avatar } from '@/components/ui/Avatar';
import { LogoutButton } from '@/features/auth/LogoutButton';
import styles from './AppSidebar.module.css';

export type SidebarItem = {
  href: string;
  label: string;
  icon: string;
  /** Ikon versi terang untuk latar gelap saat menu aktif. */
  activeIcon: string;
  /** Aktif hanya bila path sama persis (untuk menu beranda). */
  exact?: boolean;
};

type AppSidebarProps = { items: SidebarItem[]; roleLabel: string };

export function AppSidebar({ items, roleLabel }: AppSidebarProps) {
  const pathname = usePathname();

  return (
    <aside className={styles.sidebar}>
      <div className={styles.topRow}>
        <div className={styles.logo}>
          <Image src="/icons/logo-mark.svg" alt="" width={14} height={14} unoptimized />
          <span>On-Track</span>
        </div>
        <LogoutButton iconOnly className={styles.logoutCompact} />
      </div>
      <nav aria-label="Navigasi utama" className={styles.nav}>
        {items.map((item) => {
          const active = item.exact
            ? pathname === item.href
            : pathname === item.href || pathname.startsWith(`${item.href}/`);
          return (
            <Link
              key={item.href}
              href={item.href}
              aria-current={active ? 'page' : undefined}
              className={`${styles.item} ${active ? styles.active : ''}`}
            >
              <Image
                src={`/icons/${active ? item.activeIcon : item.icon}.svg`}
                alt=""
                width={20}
                height={20}
                unoptimized
              />
              {item.label}
            </Link>
          );
        })}
      </nav>
      <div className={styles.user}>
        <Avatar initials={roleLabel.slice(0, 2).toUpperCase()} size="M" />
        <span className={styles.userText}>
          <span className={styles.userName}>Masuk sebagai</span>
          <span className={styles.userRole}>{roleLabel}</span>
        </span>
      </div>
      <LogoutButton className={styles.logout} />
    </aside>
  );
}
