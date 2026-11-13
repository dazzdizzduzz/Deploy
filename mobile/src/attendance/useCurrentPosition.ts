import * as Location from 'expo-location';
import { useCallback, useEffect, useState } from 'react';
import type { Point } from './geo';

export type Position = Point & { accuracy: number | null };

/** Membaca GPS sekali saat izin lokasi ada, dan bisa dibaca ulang lewat `read()`. */
export function useCurrentPosition(enabled: boolean) {
  const [position, setPosition] = useState<Position | null>(null);
  const [locating, setLocating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const read = useCallback(async () => {
    setLocating(true);
    setError(null);
    try {
      const { coords } = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.High });
      setPosition({ latitude: coords.latitude, longitude: coords.longitude, accuracy: coords.accuracy ?? null });
    } catch {
      setError('Lokasi belum bisa dibaca. Pastikan GPS aktif, lalu coba lagi.');
    } finally {
      setLocating(false);
    }
  }, []);

  useEffect(() => {
    if (enabled && !position && !locating && !error) void read();
  }, [enabled, position, locating, error, read]);

  return { position, locating, error, read };
}
