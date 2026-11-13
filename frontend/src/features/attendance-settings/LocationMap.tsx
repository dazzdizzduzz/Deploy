import { isLatitude, isLongitude } from './rules';
import styles from './settings.module.css';

/** Kotak peta di sekitar titik; makin besar radius, makin luas area yang ditampilkan. */
function boundingBox(lat: number, lon: number, radiusMeters: number) {
  const span = Math.max(radiusMeters * 4, 400);
  const dLat = span / 111_320;
  const dLon = span / (111_320 * Math.max(Math.cos((lat * Math.PI) / 180), 0.01));
  return [lon - dLon, lat - dLat, lon + dLon, lat + dLat].map((value) => value.toFixed(6)).join(',');
}

/**
 * Pratinjau peta OpenStreetMap untuk memastikan titik koordinat kantor sudah benar.
 * Hanya menampilkan peta; titik diisi lewat kolom koordinat atau tombol "Gunakan lokasi saya".
 */
export function LocationMap({ latitude, longitude, radiusMeters }: { latitude: number; longitude: number; radiusMeters: number }) {
  if (!isLatitude(latitude) || !isLongitude(longitude)) {
    return (
      <div className={styles.mapEmpty}>
        Isi koordinat yang valid untuk melihat pratinjau peta.
      </div>
    );
  }

  const src = `https://www.openstreetmap.org/export/embed.html?bbox=${boundingBox(latitude, longitude, radiusMeters)}&layer=mapnik&marker=${latitude},${longitude}`;
  return (
    <iframe
      className={styles.map}
      title={`Peta lokasi kantor di ${latitude}, ${longitude}`}
      src={src}
      loading="lazy"
      referrerPolicy="no-referrer"
    />
  );
}
