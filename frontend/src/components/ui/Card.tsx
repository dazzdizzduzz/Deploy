import { useId, type ReactNode } from 'react';
import styles from './Card.module.css';

type CardProps = {
  /** Judul kartu (Heading/H3). Tanpa judul, kartu hanya menjadi wadah. */
  title?: string;
  /** Kalimat di bawah judul (Body/S, text/secondary). */
  description?: string;
  /** Tombol atau tautan di kanan judul. */
  actions?: ReactNode;
  children?: ReactNode;
  className?: string;
};

/** Wadah konten: bg/surface, garis border/default, radius lg, padding 20. */
export function Card({ title, description, actions, children, className }: CardProps) {
  const titleId = useId();
  const hasHeader = Boolean(title || actions);

  return (
    <section
      className={[styles.card, className].filter(Boolean).join(' ')}
      aria-labelledby={title ? titleId : undefined}
    >
      {hasHeader && (
        <header className={styles.header}>
          <div className={styles.heading}>
            {title && (
              <h2 id={titleId} className={styles.title}>
                {title}
              </h2>
            )}
            {description && <p className={styles.description}>{description}</p>}
          </div>
          {actions && <div className={styles.actions}>{actions}</div>}
        </header>
      )}
      {children}
    </section>
  );
}
