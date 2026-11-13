import * as SecureStore from 'expo-secure-store';
import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import type { Attendance } from '../api/attendance';
import { toLocalDateKey } from '../lib/format';
import { useProfile } from '../profile/ProfileProvider';

type AttendanceState = {
  /** Presensi hari ini milik pengguna yang login, atau `null` bila belum clock-in. */
  today: Attendance | null;
  /**
   * Status dari server saat rinciannya tidak tersimpan di perangkat ini (misalnya absen dari HP lain):
   * `clocked_in` bila server menolak clock-in ganda, `clocked_out` bila menolak clock-out ganda.
   */
  serverState: 'clocked_in' | 'clocked_out' | null;
  ready: boolean;
  saveToday: (record: Attendance) => Promise<void>;
  markServerState: (state: 'clocked_in' | 'clocked_out' | null) => void;
};

const AttendanceContext = createContext<AttendanceState | null>(null);

/**
 * Menyimpan presensi hari ini di perangkat. Backend belum punya GET /attendances/today
 * (lihat api/drafts/attendance-today.openapi.yaml), jadi hasil clock-in disimpan lokal per pengguna
 * dan dibuang otomatis saat tanggal kerja berganti.
 */
export function AttendanceProvider({ children }: { children: ReactNode }) {
  const { profile } = useProfile();
  const userId = profile?.id ?? null;
  const storageKey = userId ? `ontrack.attendance.${userId.replace(/[^A-Za-z0-9._-]/g, '')}` : null;
  const [today, setToday] = useState<Attendance | null>(null);
  const [serverState, setServerState] = useState<AttendanceState['serverState']>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    setToday(null);
    setServerState(null);
    if (!storageKey) return;
    setReady(false);
    SecureStore.getItemAsync(storageKey)
      .then((raw) => {
        if (!raw) return;
        const record = JSON.parse(raw) as Attendance;
        // Tanggal kerja dari server memakai zona waktu tenant; bandingkan dengan hari ini di zona yang sama.
        if (record.work_date === toLocalDateKey(new Date())) setToday(record);
        else void SecureStore.deleteItemAsync(storageKey);
      })
      .catch(() => undefined)
      .finally(() => setReady(true));
  }, [storageKey]);

  const saveToday = useCallback(
    async (record: Attendance) => {
      setToday(record);
      setServerState(null);
      if (storageKey) await SecureStore.setItemAsync(storageKey, JSON.stringify(record)).catch(() => undefined);
    },
    [storageKey],
  );

  const markServerState = useCallback((state: AttendanceState['serverState']) => setServerState(state), []);

  const value = useMemo(
    () => ({ today, serverState, ready: ready || !storageKey, saveToday, markServerState }),
    [today, serverState, ready, storageKey, saveToday, markServerState],
  );
  return <AttendanceContext.Provider value={value}>{children}</AttendanceContext.Provider>;
}

export function useAttendance() {
  const context = useContext(AttendanceContext);
  if (!context) throw new Error('useAttendance harus dipakai di dalam <AttendanceProvider>.');
  return context;
}
