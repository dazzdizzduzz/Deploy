'use client';

import { useState, type FormEvent } from 'react';
import { Alert } from '@/components/ui/Alert';
import { Button } from '@/components/ui/Button';
import { ChipButton } from '@/components/ui/Chip';
import { Drawer } from '@/components/ui/Drawer';
import { TextField } from '@/components/ui/Field';
import { Toggle } from '@/components/ui/Toggle';
import { ApiProblem } from '@/lib/api/client';
import type { Problem } from '@/lib/api/problem';
import { schedulesApi, type WorkSchedule } from './api';
import { DAYS, joinDays, parseDays, toMinutes, toTimeInput } from './rules';
import styles from './settings.module.css';

type Values = {
  name: string;
  start: string;
  end: string;
  tolerance: string;
  overtimeAfter: string;
  maxOvertime: string;
  days: string[];
  allowRemote: boolean;
  isActive: boolean;
};

type Errors = Partial<Record<'name' | 'start' | 'end' | 'tolerance' | 'overtimeAfter' | 'maxOvertime' | 'days', string>>;

/** Nilai bawaan mengikuti default di kontrak `CreateScheduleRequest`. */
const fromSchedule = (schedule: WorkSchedule | null): Values => ({
  name: schedule?.name ?? '',
  start: schedule ? toTimeInput(schedule.start_time) : '08:00',
  end: schedule ? toTimeInput(schedule.end_time) : '17:00',
  tolerance: String(schedule?.tolerance_minutes ?? 15),
  overtimeAfter: String(schedule?.overtime_after_minutes ?? 60),
  maxOvertime: String(schedule?.max_overtime_minutes ?? 180),
  days: schedule ? parseDays(schedule.working_days) : ['Sen', 'Sel', 'Rab', 'Kam', 'Jum'],
  allowRemote: schedule?.allow_remote ?? false,
  isActive: schedule?.is_active ?? true,
});

const isMinutes = (value: string, max: number) => /^\d+$/.test(value) && Number(value) <= max;

function validate(values: Values): Errors {
  const errors: Errors = {};
  if (!values.name.trim()) errors.name = 'Nama jadwal wajib diisi';
  const start = toMinutes(values.start);
  const end = toMinutes(values.end);
  if (start === null) errors.start = 'Isi jam masuk';
  if (end === null) errors.end = 'Isi jam pulang';
  else if (start !== null && end <= start) errors.end = 'Harus setelah jam masuk';
  if (!isMinutes(values.tolerance, 240)) errors.tolerance = 'Angka 0–240 menit';
  if (!isMinutes(values.overtimeAfter, 720)) errors.overtimeAfter = 'Angka 0–720 menit';
  if (!isMinutes(values.maxOvertime, 720)) errors.maxOvertime = 'Angka 0–720 menit';
  if (values.days.length === 0) errors.days = 'Pilih minimal satu hari kerja';
  return errors;
}

type ScheduleDrawerProps = {
  /** `null` berarti menambah jadwal baru. */
  schedule: WorkSchedule | null;
  onClose: () => void;
  onSaved: (schedule: WorkSchedule) => void;
};

/** FR-05 · Tambah atau ubah jadwal kerja: jam, toleransi terlambat, hari kerja, dan lembur. */
export function ScheduleDrawer({ schedule, onClose, onSaved }: ScheduleDrawerProps) {
  const [values, setValues] = useState<Values>(() => fromSchedule(schedule));
  const [errors, setErrors] = useState<Errors>({});
  const [problem, setProblem] = useState<Problem | null>(null);
  const [saving, setSaving] = useState(false);

  function set<K extends keyof Values>(key: K, value: Values[K]) {
    setValues((current) => ({ ...current, [key]: value }));
    if (key in errors) setErrors((current) => ({ ...current, [key]: undefined }));
  }

  const toggleDay = (day: string) =>
    set('days', values.days.includes(day) ? values.days.filter((item) => item !== day) : [...values.days, day]);

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
      start_time: values.start,
      end_time: values.end,
      tolerance_minutes: Number(values.tolerance),
      overtime_after_minutes: Number(values.overtimeAfter),
      max_overtime_minutes: Number(values.maxOvertime),
      working_days: joinDays(values.days),
      allow_remote: values.allowRemote,
    };
    try {
      const saved = schedule
        ? await schedulesApi.update(schedule.id, { ...body, is_active: values.isActive })
        : await schedulesApi.create(body);
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
      title={schedule ? 'Edit jadwal kerja' : 'Tambah jadwal kerja'}
      onClose={onClose}
      onSubmit={handleSubmit}
      actions={
        <>
          <Button variant="secondary" compact onClick={onClose}>
            Batal
          </Button>
          <Button compact type="submit" disabled={saving}>
            {saving ? 'Menyimpan…' : 'Simpan jadwal'}
          </Button>
        </>
      }
    >
      {problem && (
        <Alert tone="danger" title={problem.title}>
          {problem.detail ?? 'Jadwal belum tersimpan. Coba lagi.'}
        </Alert>
      )}
      <TextField
        label="Nama jadwal"
        helper="Contoh: Shift Pagi Reguler"
        error={errors.name}
        value={values.name}
        maxLength={100}
        autoComplete="off"
        onChange={(event) => set('name', event.target.value)}
      />
      <div className={styles.fieldRow}>
        <TextField
          label="Jam masuk"
          helper="Zona waktu tenant"
          type="time"
          error={errors.start}
          value={values.start}
          onChange={(event) => set('start', event.target.value)}
        />
        <TextField
          label="Jam pulang"
          helper=" "
          type="time"
          error={errors.end}
          value={values.end}
          onChange={(event) => set('end', event.target.value)}
        />
      </div>
      <TextField
        label="Toleransi terlambat (menit)"
        helper="Lewat dari ini dihitung terlambat"
        inputMode="numeric"
        error={errors.tolerance}
        value={values.tolerance}
        onChange={(event) => set('tolerance', event.target.value.replace(/\D/g, ''))}
      />
      <fieldset className={styles.fieldset}>
        <legend className={styles.groupLabel}>Hari kerja</legend>
        <div className={styles.days}>
          {DAYS.map((day) => (
            <ChipButton key={day} selected={values.days.includes(day)} onClick={() => toggleDay(day)}>
              {day}
            </ChipButton>
          ))}
        </div>
        {errors.days && (
          <p className={styles.fieldError} role="alert">
            {errors.days}
          </p>
        )}
      </fieldset>
      <div className={styles.fieldRow}>
        <TextField
          label="Lembur dihitung setelah (menit)"
          helper="Dari jam pulang"
          inputMode="numeric"
          error={errors.overtimeAfter}
          value={values.overtimeAfter}
          onChange={(event) => set('overtimeAfter', event.target.value.replace(/\D/g, ''))}
        />
        <TextField
          label="Batas lembur harian (menit)"
          helper="Butuh persetujuan atasan"
          inputMode="numeric"
          error={errors.maxOvertime}
          value={values.maxOvertime}
          onChange={(event) => set('maxOvertime', event.target.value.replace(/\D/g, ''))}
        />
      </div>
      <Toggle
        label="Izinkan WFH / dinas luar dengan persetujuan atasan"
        checked={values.allowRemote}
        onChange={(event) => set('allowRemote', event.target.checked)}
      />
      {schedule && (
        <Toggle
          label="Jadwal aktif"
          checked={values.isActive}
          onChange={(event) => set('isActive', event.target.checked)}
        />
      )}
    </Drawer>
  );
}
