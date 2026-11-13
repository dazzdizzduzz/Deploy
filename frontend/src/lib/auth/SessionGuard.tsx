'use client';

import { useEffect, useRef } from 'react';
import { IDLE_TIMEOUT_MS } from './constants';
import { endSession } from './logout';

const ACTIVITY_KEY = 'ontrack.last_activity';
const CHECK_EVERY_MS = 30_000;
const WRITE_THROTTLE_MS = 5_000;
const EVENTS = ['pointerdown', 'keydown', 'scroll', 'touchstart'] as const;

function readShared(): number | null {
  try {
    const value = Number(window.localStorage.getItem(ACTIVITY_KEY));
    return Number.isFinite(value) && value > 0 ? value : null;
  } catch {
    return null;
  }
}

function writeShared(value: number) {
  try {
    window.localStorage.setItem(ACTIVITY_KEY, String(value));
  } catch {
    // Penyimpanan diblokir: penghitung tetap jalan per tab.
  }
}

/**
 * Mengakhiri sesi web setelah 30 menit tanpa aktivitas, atau saat access token kedaluwarsa.
 * Aktivitas dibagi antar-tab lewat localStorage, jadi tab yang aktif menjaga tab lain tetap masuk.
 */
export function SessionGuard({ expiresAt }: { expiresAt: number }) {
  const lastActivity = useRef(Date.now());
  const lastWrite = useRef(0);
  const ended = useRef(false);

  useEffect(() => {
    lastActivity.current = Date.now();
    writeShared(lastActivity.current);

    function markActive() {
      const now = Date.now();
      lastActivity.current = now;
      if (now - lastWrite.current > WRITE_THROTTLE_MS) {
        lastWrite.current = now;
        writeShared(now);
      }
    }

    function check() {
      if (ended.current) return;
      const now = Date.now();
      if (now >= expiresAt) {
        ended.current = true;
        void endSession('expired');
        return;
      }
      const latest = Math.max(lastActivity.current, readShared() ?? 0);
      if (now - latest >= IDLE_TIMEOUT_MS) {
        ended.current = true;
        void endSession('idle');
      }
    }

    // Saat tab kembali dibuka setelah lama ditinggal, langsung periksa tanpa menunggu interval.
    function onVisible() {
      if (document.visibilityState === 'visible') check();
    }

    EVENTS.forEach((event) => window.addEventListener(event, markActive, { passive: true }));
    document.addEventListener('visibilitychange', onVisible);
    const interval = window.setInterval(check, CHECK_EVERY_MS);
    const expiry = window.setTimeout(check, Math.max(0, expiresAt - Date.now()) + 1_000);

    return () => {
      EVENTS.forEach((event) => window.removeEventListener(event, markActive));
      document.removeEventListener('visibilitychange', onVisible);
      window.clearInterval(interval);
      window.clearTimeout(expiry);
    };
  }, [expiresAt]);

  return null;
}
