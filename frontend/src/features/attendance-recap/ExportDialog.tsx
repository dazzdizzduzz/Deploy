'use client';

import { useState, type FormEvent } from 'react';
import { Button } from '@/components/ui/Button';
import { Checkbox } from '@/components/ui/Checkbox';
import { Dialog } from '@/components/ui/Dialog';
import { SelectField, type SelectOption } from '@/components/ui/Field';
import type { ExportInput } from './api';
import styles from './recap.module.css';
import { monthLabel, recentMonths } from './period';

type ExportDialogProps = {
  initial: { month: string; departmentId: string };
  departments: SelectOption[];
  onClose: () => void;
  onExport: (input: ExportInput) => void;
};

/** W38 · Ekspor PDF: pilih periode, departemen, dan isi dokumen. */
export function ExportDialog({ initial, departments, onClose, onExport }: ExportDialogProps) {
  const [month, setMonth] = useState(initial.month);
  const [departmentId, setDepartmentId] = useState(initial.departmentId);
  const [includeDaily, setIncludeDaily] = useState(true);
  const [includeSelfie, setIncludeSelfie] = useState(false);

  function handleSubmit(event: FormEvent) {
    event.preventDefault();
    onExport({ month, department_id: departmentId || null, include_daily: includeDaily, include_selfie: includeSelfie });
  }

  return (
    <Dialog
      title="Ekspor rekap ke PDF"
      description="Dokumen dibuat di server, lalu tautan unduhan muncul di halaman ini."
      size="md"
      showClose
      onClose={onClose}
      onSubmit={handleSubmit}
      actions={
        <>
          <Button variant="secondary" compact onClick={onClose}>
            Batal
          </Button>
          <Button type="submit">Buat PDF</Button>
        </>
      }
    >
      <div className={styles.dialogFields}>
        <SelectField
          label="Periode"
          value={month}
          onChange={(event) => setMonth(event.target.value)}
          options={recentMonths().map((value) => ({ value, label: monthLabel(value) }))}
        />
        <SelectField
          label="Departemen"
          placeholder="Semua departemen"
          value={departmentId}
          onChange={(event) => setDepartmentId(event.target.value)}
          options={departments}
        />
        <fieldset className={styles.options}>
          <legend className={styles.legend}>Isi dokumen</legend>
          <Checkbox label="Ringkasan per karyawan" checked disabled />
          <Checkbox
            label="Rincian harian per karyawan"
            checked={includeDaily}
            onChange={(event) => setIncludeDaily(event.target.checked)}
          />
          <Checkbox
            label="Sertakan foto selfie (dokumen jauh lebih besar)"
            checked={includeSelfie}
            onChange={(event) => setIncludeSelfie(event.target.checked)}
          />
        </fieldset>
      </div>
    </Dialog>
  );
}
