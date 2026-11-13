import Image from 'next/image';
import type { ReactNode } from 'react';
import { ButtonLink } from '@/components/ui/Button';
import styles from './AuthParts.module.css';

export function AuthStack({ children }: { children: ReactNode }) {
  return <div className={styles.stack}>{children}</div>;
}

type IconBadgeProps = {
  icon: string;
  /** subtle/neutral: 64px, accent: 80px; ukuran ikon mengikuti file SVG dari Figma. */
  tone?: 'subtle' | 'neutral' | 'accent';
};

const iconSizes = { subtle: 27, neutral: 27, accent: 34 };

export function AuthIconBadge({ icon, tone = 'subtle' }: IconBadgeProps) {
  const size = iconSizes[tone];
  return (
    <div className={`${styles.badge} ${styles[tone]}`}>
      <Image src={`/icons/${icon}.svg`} alt="" width={size} height={size} unoptimized />
    </div>
  );
}

export function AuthHeading({ title, children }: { title: string; children: ReactNode }) {
  return (
    <>
      <h1 className={styles.title}>{title}</h1>
      <p className={styles.body}>{children}</p>
    </>
  );
}

export function AuthNote({ children }: { children: ReactNode }) {
  return <p className={styles.note}>{children}</p>;
}

export function BackToLoginLink() {
  return (
    <div className={styles.start}>
      <ButtonLink variant="ghost" href="/login">
        Kembali ke login
        <Image src="/icons/back.svg" alt="" width={20} height={20} unoptimized />
      </ButtonLink>
    </div>
  );
}
