import type { OfficeLocation } from '../api/attendance';

export type Point = { latitude: number; longitude: number };

const EARTH_RADIUS_M = 6_371_000;
const toRad = (degree: number) => (degree * Math.PI) / 180;

/** Jarak dua titik (meter) dengan rumus haversine, sama seperti perhitungan geofence di backend. */
export function distanceMeters(a: Point, b: Point): number {
  const dLat = toRad(b.latitude - a.latitude);
  const dLon = toRad(b.longitude - a.longitude);
  const h =
    Math.sin(dLat / 2) ** 2 + Math.cos(toRad(a.latitude)) * Math.cos(toRad(b.latitude)) * Math.sin(dLon / 2) ** 2;
  return 2 * EARTH_RADIUS_M * Math.asin(Math.min(1, Math.sqrt(h)));
}

export type GeofenceCheck = {
  location: OfficeLocation;
  distance: number;
  inside: boolean;
};

/**
 * Lokasi kantor aktif terdekat dan apakah titik berada di dalam radiusnya. Hanya untuk tampilan;
 * keputusan akhir tetap di server saat clock-in.
 */
export function checkGeofence(point: Point, locations: OfficeLocation[]): GeofenceCheck | null {
  let best: GeofenceCheck | null = null;
  for (const location of locations) {
    if (!location.is_active) continue;
    // Sama dengan backend: bandingkan jarak sebelum dibulatkan; tepat di batas masih di dalam.
    const exact = distanceMeters(point, location);
    const distance = Math.round(exact);
    if (!best || distance < best.distance) {
      best = { location, distance, inside: exact <= location.radius_meters };
    }
  }
  return best;
}
