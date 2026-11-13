'use client';

import { useId, type CSSProperties } from 'react';
import { RADIUS_MAX, RADIUS_MIN, RADIUS_STEP } from './rules';
import styles from './settings.module.css';

export function RadiusSlider({ value, onChange }: { value: number; onChange: (meters: number) => void }) {
  const id = useId();
  const fill = ((value - RADIUS_MIN) / (RADIUS_MAX - RADIUS_MIN)) * 100;

  return (
    <div className={styles.radius}>
      <div className={styles.radiusLabel}>
        <label htmlFor={id}>Radius geofence</label>
        <span className={styles.radiusValue}>{value} m</span>
      </div>
      <input
        id={id}
        type="range"
        className={styles.slider}
        min={RADIUS_MIN}
        max={RADIUS_MAX}
        step={RADIUS_STEP}
        value={value}
        aria-valuetext={`${value} meter`}
        style={{ '--slider-fill': `${fill}%` } as CSSProperties}
        onChange={(event) => onChange(Number(event.target.value))}
      />
      <div className={styles.radiusScale}>
        <span>{RADIUS_MIN} m</span>
        <span>{RADIUS_MAX} m</span>
      </div>
    </div>
  );
}
