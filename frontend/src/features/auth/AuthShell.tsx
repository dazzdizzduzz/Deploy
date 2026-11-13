import type { ReactNode } from 'react';
import { AuthBrandPanel } from './AuthBrandPanel';
import type { BrandCopy } from './copy';
import styles from './AuthShell.module.css';

export function AuthShell({ brand, children }: { brand?: BrandCopy; children: ReactNode }) {
  return (
    <div className={styles.layout}>
      <AuthBrandPanel brand={brand} />
      <main className={styles.main}>{children}</main>
    </div>
  );
}
