import Image from 'next/image';
import styles from './AuthBrandPanel.module.css';
import { adminLogin, type BrandCopy } from './copy';

export function AuthBrandPanel({ brand = adminLogin.brand }: { brand?: BrandCopy }) {
  const { headline, subline, stats } = brand;

  return (
    <aside className={styles.panel}>
      <div className={styles.logo}>
        <Image src="/icons/logo-dot.svg" alt="" width={16} height={16} unoptimized />
        <span>On-Track</span>
      </div>
      <div className={styles.copy}>
        <p className={styles.headline}>{headline}</p>
        <p className={styles.subline}>{subline}</p>
      </div>
      <dl className={styles.stats}>
        {stats.map((stat) => (
          <div key={stat.label} className={styles.stat}>
            <dt className={styles.statLabel}>{stat.label}</dt>
            <dd className={styles.statValue}>{stat.value}</dd>
          </div>
        ))}
      </dl>
    </aside>
  );
}
