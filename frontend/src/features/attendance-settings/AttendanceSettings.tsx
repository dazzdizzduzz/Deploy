'use client';

import Image from 'next/image';
import { useState, type FormEvent } from 'react';
import { Alert } from '@/components/ui/Alert';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Dialog } from '@/components/ui/Dialog';
import { Toast } from '@/components/ui/Toast';
import { ApiProblem } from '@/lib/api/client';
import type { Problem } from '@/lib/api/problem';
import { useAsync } from '@/lib/hooks/useAsync';
import hr from '@/features/hr/hr.module.css';
import { locationsApi, schedulesApi, type OfficeLocation, type WorkSchedule } from './api';
import { LocationDrawer } from './LocationDrawer';
import { formatMinutes, summarizeDays, toTimeInput } from './rules';
import { ScheduleDrawer } from './ScheduleDrawer';
import styles from './settings.module.css';

type Panel =
  | { kind: 'location'; item: OfficeLocation | null }
  | { kind: 'schedule'; item: WorkSchedule | null };

type Removing = { kind: 'location'; item: OfficeLocation } | { kind: 'schedule'; item: WorkSchedule };

const activeBadge = (active: boolean) =>
  active ? <Badge tone="success">Aktif</Badge> : <Badge tone="neutral">Nonaktif</Badge>;

/** FR-05 · Pengaturan batas absensi: lokasi kantor (geofence) dan jadwal kerja. */
export function AttendanceSettings({ canManage }: { canManage: boolean }) {
  const locations = useAsync(() => locationsApi.list(), []);
  const schedules = useAsync(() => schedulesApi.list(), []);
  const [panel, setPanel] = useState<Panel | null>(null);
  const [removing, setRemoving] = useState<Removing | null>(null);
  const [toast, setToast] = useState<string | null>(null);

  const locationRows = locations.data?.data ?? [];
  const scheduleRows = schedules.data?.data ?? [];

  function finish(message: string, reload: () => Promise<void>) {
    setPanel(null);
    setRemoving(null);
    setToast(message);
    void reload();
  }

  return (
    <main className={hr.main}>
      <header className={hr.header}>
        <div className={hr.titleBlock}>
          <h1 className={hr.title}>Jadwal &amp; lokasi absensi</h1>
          <p className={hr.subtitle}>
            Batas tempat dan waktu absensi. Perubahan berlaku mulai absensi berikutnya.
          </p>
        </div>
      </header>

      {toast && (
        <Toast tone="success" title="Pengaturan disimpan" onDismiss={() => setToast(null)}>
          {toast}
        </Toast>
      )}

      <section className={hr.card} aria-labelledby="lokasi-kantor">
        <div className={styles.cardHeader}>
          <div className={hr.titleBlock}>
            <h2 id="lokasi-kantor" className={hr.cardTitle}>
              Lokasi kantor &amp; geofence
            </h2>
            <p className={hr.subtitle}>Karyawan hanya bisa clock-in di dalam radius lokasi yang aktif.</p>
          </div>
          {canManage && (
            <Button variant="secondary" compact onClick={() => setPanel({ kind: 'location', item: null })}>
              Tambah lokasi
              <Image src="/icons/plus.svg" alt="" width={20} height={20} unoptimized />
            </Button>
          )}
        </div>
        <LoadError state={locations} label="lokasi kantor" />
        <div className={styles.tableWrap} aria-busy={locations.loading}>
          <div className={hr.tableScroll}>
            <table className={hr.table}>
              <thead>
                <tr>
                  <th scope="col">Lokasi</th>
                  <th scope="col">Koordinat</th>
                  <th scope="col">Radius</th>
                  <th scope="col">Selfie</th>
                  <th scope="col">Status</th>
                  {canManage && (
                    <th scope="col" className={hr.colActions}>
                      Aksi
                    </th>
                  )}
                </tr>
              </thead>
              <tbody>
                {locationRows.map((row) => (
                  <tr key={row.id}>
                    <td>
                      <div className={hr.personText}>
                        <span className={hr.personName}>{row.name}</span>
                        <span className={hr.personEmail}>{row.address}</span>
                      </div>
                    </td>
                    <td className={styles.mono}>
                      {row.latitude.toFixed(6)}, {row.longitude.toFixed(6)}
                    </td>
                    <td>{row.radius_meters} m</td>
                    <td>{row.selfie_required ? 'Wajib' : 'Tidak'}</td>
                    <td>{activeBadge(row.is_active)}</td>
                    {canManage && (
                      <td className={hr.colActions}>
                        <RowActions
                          name={row.name}
                          onEdit={() => setPanel({ kind: 'location', item: row })}
                          onDelete={() => setRemoving({ kind: 'location', item: row })}
                        />
                      </td>
                    )}
                  </tr>
                ))}
                {locationRows.length === 0 && (
                  <EmptyRow
                    colSpan={canManage ? 6 : 5}
                    loading={locations.loading}
                    text="Belum ada lokasi kantor. Tambahkan lokasi agar karyawan bisa clock-in."
                  />
                )}
              </tbody>
            </table>
          </div>
        </div>
      </section>

      <section className={hr.card} aria-labelledby="jadwal-kerja">
        <div className={styles.cardHeader}>
          <div className={hr.titleBlock}>
            <h2 id="jadwal-kerja" className={hr.cardTitle}>
              Jadwal kerja
            </h2>
            <p className={hr.subtitle}>Jam masuk, toleransi terlambat, hari kerja, dan aturan lembur.</p>
          </div>
          {canManage && (
            <Button variant="secondary" compact onClick={() => setPanel({ kind: 'schedule', item: null })}>
              Tambah jadwal
              <Image src="/icons/plus.svg" alt="" width={20} height={20} unoptimized />
            </Button>
          )}
        </div>
        <LoadError state={schedules} label="jadwal kerja" />
        <div className={styles.tableWrap} aria-busy={schedules.loading}>
          <div className={hr.tableScroll}>
            <table className={hr.table}>
              <thead>
                <tr>
                  <th scope="col">Jadwal</th>
                  <th scope="col">Jam kerja</th>
                  <th scope="col">Hari</th>
                  <th scope="col">Toleransi</th>
                  <th scope="col">Lembur</th>
                  <th scope="col">Status</th>
                  {canManage && (
                    <th scope="col" className={hr.colActions}>
                      Aksi
                    </th>
                  )}
                </tr>
              </thead>
              <tbody>
                {scheduleRows.map((row) => (
                  <tr key={row.id}>
                    <td>
                      <div className={hr.personText}>
                        <span className={hr.personName}>{row.name}</span>
                        {row.allow_remote && <span className={hr.personEmail}>WFH dengan persetujuan</span>}
                      </div>
                    </td>
                    <td>
                      {toTimeInput(row.start_time)}–{toTimeInput(row.end_time)}
                    </td>
                    <td>{summarizeDays(row.working_days)}</td>
                    <td>{formatMinutes(row.tolerance_minutes)}</td>
                    <td>
                      Setelah {formatMinutes(row.overtime_after_minutes)}, maks.{' '}
                      {formatMinutes(row.max_overtime_minutes)}
                    </td>
                    <td>{activeBadge(row.is_active)}</td>
                    {canManage && (
                      <td className={hr.colActions}>
                        <RowActions
                          name={row.name}
                          onEdit={() => setPanel({ kind: 'schedule', item: row })}
                          onDelete={() => setRemoving({ kind: 'schedule', item: row })}
                        />
                      </td>
                    )}
                  </tr>
                ))}
                {scheduleRows.length === 0 && (
                  <EmptyRow
                    colSpan={canManage ? 7 : 6}
                    loading={schedules.loading}
                    text="Belum ada jadwal kerja. Tambahkan jadwal untuk menghitung keterlambatan dan lembur."
                  />
                )}
              </tbody>
            </table>
          </div>
        </div>
      </section>

      {panel?.kind === 'location' && (
        <LocationDrawer
          location={panel.item}
          onClose={() => setPanel(null)}
          onSaved={(saved) =>
            finish(`Lokasi "${saved.name}" ${panel.item ? 'diperbarui' : 'ditambahkan'} dengan radius ${saved.radius_meters} m.`, locations.reload)
          }
        />
      )}
      {panel?.kind === 'schedule' && (
        <ScheduleDrawer
          schedule={panel.item}
          onClose={() => setPanel(null)}
          onSaved={(saved) =>
            finish(`Jadwal "${saved.name}" ${panel.item ? 'diperbarui' : 'ditambahkan'}.`, schedules.reload)
          }
        />
      )}
      {removing && (
        <DeleteDialog
          removing={removing}
          onClose={() => setRemoving(null)}
          onDeleted={() =>
            finish(
              `${removing.kind === 'location' ? 'Lokasi' : 'Jadwal'} "${removing.item.name}" dihapus.`,
              removing.kind === 'location' ? locations.reload : schedules.reload,
            )
          }
        />
      )}
    </main>
  );
}

function LoadError({ state, label }: { state: { error: ApiProblem | null; reload: () => Promise<void> }; label: string }) {
  if (!state.error) return null;
  return (
    <Alert tone="danger" title={state.error.problem.title}>
      {state.error.problem.detail ?? `Daftar ${label} belum bisa dimuat.`}{' '}
      <button type="button" className={hr.inlineLink} onClick={() => void state.reload()}>
        Coba lagi
      </button>
    </Alert>
  );
}

function RowActions({ name, onEdit, onDelete }: { name: string; onEdit: () => void; onDelete: () => void }) {
  return (
    <div className={hr.actions}>
      <Button variant="secondary" compact onClick={onEdit} aria-label={`Edit ${name}`}>
        Edit
      </Button>
      <Button variant="ghost" compact onClick={onDelete} aria-label={`Hapus ${name}`}>
        Hapus
      </Button>
    </div>
  );
}

function EmptyRow({ colSpan, loading, text }: { colSpan: number; loading: boolean; text: string }) {
  return (
    <tr>
      <td colSpan={colSpan} className={hr.empty}>
        {loading ? 'Memuat…' : text}
      </td>
    </tr>
  );
}

function DeleteDialog({ removing, onClose, onDeleted }: { removing: Removing; onClose: () => void; onDeleted: () => void }) {
  const [problem, setProblem] = useState<Problem | null>(null);
  const [busy, setBusy] = useState(false);
  const isLocation = removing.kind === 'location';

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    if (busy) return;
    setBusy(true);
    setProblem(null);
    try {
      await (isLocation ? locationsApi.remove(removing.item.id) : schedulesApi.remove(removing.item.id));
      onDeleted();
    } catch (error) {
      setBusy(false);
      setProblem(
        error instanceof ApiProblem ? error.problem : { type: 'about:blank', title: 'Terjadi kesalahan', status: 0 },
      );
    }
  }

  return (
    <Dialog
      title={`Hapus ${isLocation ? 'lokasi' : 'jadwal'} "${removing.item.name}"?`}
      description={
        isLocation
          ? 'Karyawan yang memakai lokasi ini tidak bisa clock-in di sana lagi. Untuk menghentikan sementara, nonaktifkan lewat Edit.'
          : 'Karyawan yang memakai jadwal ini perlu diberi jadwal lain. Untuk menghentikan sementara, nonaktifkan lewat Edit.'
      }
      onClose={onClose}
      onSubmit={handleSubmit}
      actions={
        <>
          <Button variant="secondary" compact onClick={onClose}>
            Batal
          </Button>
          <Button type="submit" variant="danger" disabled={busy}>
            {busy ? 'Menghapus…' : 'Hapus'}
          </Button>
        </>
      }
    >
      {problem && (
        <Alert tone="danger" title={problem.title}>
          {problem.detail ?? 'Coba lagi beberapa saat lagi.'}
        </Alert>
      )}
    </Dialog>
  );
}
