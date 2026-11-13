'use client';

import { useState, type FormEvent } from 'react';
import { Alert } from '@/components/ui/Alert';
import { Button } from '@/components/ui/Button';
import { Drawer } from '@/components/ui/Drawer';
import { TextField } from '@/components/ui/Field';
import { Toggle } from '@/components/ui/Toggle';
import { ApiProblem } from '@/lib/api/client';
import type { Problem } from '@/lib/api/problem';
import { locationsApi, type OfficeLocation } from './api';
import { LocationMap } from './LocationMap';
import { RadiusSlider } from './RadiusSlider';
import { RADIUS_MAX, RADIUS_MIN, isLatitude, isLongitude } from './rules';
import styles from './settings.module.css';

type Values = {
  name: string;
  address: string;
  latitude: string;
  longitude: string;
  radius: number;
  selfieRequired: boolean;
  isActive: boolean;
};

type Errors = Partial<Record<'name' | 'address' | 'latitude' | 'longitude' | 'radius', string>>;

const fromLocation = (location: OfficeLocation | null): Values => ({
  name: location?.name ?? '',
  address: location?.address ?? '',
  latitude: location ? String(location.latitude) : '',
  longitude: location ? String(location.longitude) : '',
  radius: location?.radius_meters ?? 100,
  selfieRequired: location?.selfie_required ?? true,
  isActive: location?.is_active ?? true,
});

function validate(values: Values): Errors {
  const errors: Errors = {};
  if (!values.name.trim()) errors.name = 'Nama lokasi wajib diisi';
  if (!values.address.trim()) errors.address = 'Alamat wajib diisi';
  if (!values.latitude.trim() || !isLatitude(Number(values.latitude))) errors.latitude = 'Lintang -90 sampai 90';
  if (!values.longitude.trim() || !isLongitude(Number(values.longitude))) errors.longitude = 'Bujur -180 sampai 180';
  if (values.radius < RADIUS_MIN || values.radius > RADIUS_MAX) errors.radius = `Radius ${RADIUS_MIN}–${RADIUS_MAX} m`;
  return errors;
}

type LocationDrawerProps = {
  /** `null` berarti menambah lokasi baru. */
  location: OfficeLocation | null;
  onClose: () => void;
  onSaved: (location: OfficeLocation) => void;
};

/** FR-05 · Tambah atau ubah lokasi kantor dan batas geofence absensi. */
export function LocationDrawer({ location, onClose, onSaved }: LocationDrawerProps) {
  const [values, setValues] = useState<Values>(() => fromLocation(location));
  const [errors, setErrors] = useState<Errors>({});
  const [problem, setProblem] = useState<Problem | null>(null);
  const [saving, setSaving] = useState(false);
  const [locating, setLocating] = useState(false);
  const [geoError, setGeoError] = useState<string | null>(null);

  function set<K extends keyof Values>(key: K, value: Values[K]) {
    setValues((current) => ({ ...current, [key]: value }));
    if (key in errors) setErrors((current) => ({ ...current, [key]: undefined }));
  }

  /** Isi koordinat dari GPS perangkat; berguna bila HRD sedang berada di kantor. */
  function fillCurrentPosition() {
    if (!('geolocation' in navigator)) {
      setGeoError('Browser ini tidak mendukung lokasi.');
      return;
    }
    setLocating(true);
    setGeoError(null);
    navigator.geolocation.getCurrentPosition(
      ({ coords }) => {
        setLocating(false);
        setValues((current) => ({
          ...current,
          latitude: coords.latitude.toFixed(6),
          longitude: coords.longitude.toFixed(6),
        }));
        setErrors((current) => ({ ...current, latitude: undefined, longitude: undefined }));
      },
      (error) => {
        setLocating(false);
        setGeoError(
          error.code === error.PERMISSION_DENIED
            ? 'Izin lokasi ditolak. Izinkan lokasi di browser atau isi koordinat manual.'
            : 'Lokasi belum bisa dibaca. Coba lagi atau isi koordinat manual.',
        );
      },
      { enableHighAccuracy: true, timeout: 10_000 },
    );
  }

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    if (saving) return;
    const nextErrors = validate(values);
    setErrors(nextErrors);
    if (Object.values(nextErrors).some(Boolean)) return;

    setSaving(true);
    setProblem(null);
    const body = {
      name: values.name.trim(),
      address: values.address.trim(),
      latitude: Number(values.latitude),
      longitude: Number(values.longitude),
      radius_meters: values.radius,
      selfie_required: values.selfieRequired,
    };
    try {
      const saved = location
        ? await locationsApi.update(location.id, { ...body, is_active: values.isActive })
        : await locationsApi.create(body);
      onSaved(saved);
    } catch (error) {
      setSaving(false);
      setProblem(
        error instanceof ApiProblem
          ? error.problem
          : { type: 'about:blank', title: 'Terjadi kesalahan', status: 0 },
      );
    }
  }

  return (
    <Drawer
      title={location ? 'Edit lokasi kantor' : 'Tambah lokasi kantor'}
      onClose={onClose}
      onSubmit={handleSubmit}
      actions={
        <>
          <Button variant="secondary" compact onClick={onClose}>
            Batal
          </Button>
          <Button compact type="submit" disabled={saving}>
            {saving ? 'Menyimpan…' : 'Simpan lokasi'}
          </Button>
        </>
      }
    >
      {problem && (
        <Alert tone="danger" title={problem.title}>
          {problem.detail ?? 'Lokasi belum tersimpan. Coba lagi.'}
        </Alert>
      )}
      <TextField
        label="Nama lokasi"
        helper="Contoh: Kantor Pusat Denpasar"
        error={errors.name}
        value={values.name}
        maxLength={100}
        autoComplete="off"
        onChange={(event) => set('name', event.target.value)}
      />
      <TextField
        label="Alamat"
        helper=" "
        error={errors.address}
        value={values.address}
        maxLength={255}
        autoComplete="off"
        onChange={(event) => set('address', event.target.value)}
      />
      <LocationMap latitude={Number(values.latitude)} longitude={Number(values.longitude)} radiusMeters={values.radius} />
      <div className={styles.fieldRow}>
        <TextField
          label="Lintang (latitude)"
          helper="Contoh: -8.670458"
          inputMode="decimal"
          error={errors.latitude}
          value={values.latitude}
          autoComplete="off"
          onChange={(event) => set('latitude', event.target.value.replace(',', '.'))}
        />
        <TextField
          label="Bujur (longitude)"
          helper="Contoh: 115.212629"
          inputMode="decimal"
          error={errors.longitude}
          value={values.longitude}
          autoComplete="off"
          onChange={(event) => set('longitude', event.target.value.replace(',', '.'))}
        />
      </div>
      <div className={styles.geoRow}>
        <Button variant="secondary" compact onClick={fillCurrentPosition} disabled={locating}>
          {locating ? 'Membaca lokasi…' : 'Gunakan lokasi saya'}
        </Button>
        <p className={styles.hint} aria-live="polite">
          {geoError ?? 'Pakai saat Anda berada di kantor, lalu periksa titiknya di peta.'}
        </p>
      </div>
      <RadiusSlider value={values.radius} onChange={(radius) => set('radius', radius)} />
      <Toggle
        label="Wajib selfie saat clock-in"
        checked={values.selfieRequired}
        onChange={(event) => set('selfieRequired', event.target.checked)}
      />
      {location && (
        <Toggle
          label="Lokasi aktif untuk absensi"
          checked={values.isActive}
          onChange={(event) => set('isActive', event.target.checked)}
        />
      )}
    </Drawer>
  );
}
