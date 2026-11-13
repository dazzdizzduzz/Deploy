'use client';

import Image from 'next/image';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { useState } from 'react';
import { Alert } from '@/components/ui/Alert';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { SelectField } from '@/components/ui/Field';
import { StatCard } from '@/components/ui/StatCard';
import hr from '@/features/hr/hr.module.css';
import { departmentsApi } from '@/features/hr/api';
import { ApiProblem } from '@/lib/api/client';
import { formatDate, formatDateTime } from '@/lib/format';
import { useAsync } from '@/lib/hooks/useAsync';
import { recapApi, type TodayStatus } from './api';
import { ExportDialog } from './ExportDialog';
import { currentMonth, isMonth, monthLabel, recentMonths, shortDuration } from './period';
import styles from './recap.module.css';
import { useRecapExport } from './useRecapExport';

const PER_PAGE = 20;

const todayStatuses: { value: TodayStatus; label: string; tone: 'success' | 'warning' | 'neutral' | 'danger' }[] = [
  { value: 'hadir', label: 'Hadir', tone: 'success' },
  { value: 'terlambat', label: 'Terlambat', tone: 'warning' },
  { value: 'izin', label: 'Izin/Cuti/Sakit', tone: 'neutral' },
  { value: 'alpha', label: 'Alpha', tone: 'danger' },
  { value: 'belum', label: 'Belum absen', tone: 'neutral' },
];

const isTodayStatus = (value: string | null): value is TodayStatus =>
  todayStatuses.some((status) => status.value === value);

/** FR-11 · Rekap kehadiran seluruh karyawan untuk HRD/Owner, dengan ekspor PDF. */
export function AttendanceRecap({ canExport }: { canExport: boolean }) {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const thisMonth = currentMonth();

  // Filter disimpan di URL agar halaman bisa dibagikan dan tetap sama saat dimuat ulang.
  const month = isMonth(params.get('bulan')) ? (params.get('bulan') as string) : thisMonth;
  const departmentId = params.get('departemen') ?? '';
  const status = month === thisMonth && isTodayStatus(params.get('status')) ? (params.get('status') as TodayStatus) : '';
  const page = Math.max(1, Number(params.get('halaman')) || 1);

  const [exportOpen, setExportOpen] = useState(false);
  const exporter = useRecapExport();

  function update(next: Record<string, string>) {
    const query = new URLSearchParams(params.toString());
    for (const [key, value] of Object.entries(next)) {
      if (value) query.set(key, value);
      else query.delete(key);
    }
    if (!('halaman' in next)) query.delete('halaman');
    router.replace(`${pathname}${query.size ? `?${query}` : ''}`, { scroll: false });
  }

  const departments = useAsync(() => departmentsApi.list({ per_page: 100 }), []);
  const recap = useAsync(
    () =>
      recapApi.get({
        month,
        department_id: departmentId || undefined,
        today_status: status || undefined,
        page,
        per_page: PER_PAGE,
      }),
    [month, departmentId, status, page],
  );

  const departmentOptions = (departments.data?.data ?? []).map(({ id, name }) => ({ value: id, label: name }));
  const departmentName = departmentOptions.find((option) => option.value === departmentId)?.label;
  const unavailable = recap.error instanceof ApiProblem && (recap.error.status === 404 || recap.error.status === 405);
  const data = recap.data;
  const rows = data?.data ?? [];
  const meta = data?.meta;
  const from = meta && meta.total > 0 ? (meta.page - 1) * meta.per_page + 1 : 0;
  const filterText = [monthLabel(month), departmentName ?? 'Semua departemen', status && todayStatuses.find((s) => s.value === status)?.label]
    .filter(Boolean)
    .join(' · ');

  return (
    <main className={hr.main}>
      <header className={hr.header}>
        <div className={hr.titleBlock}>
          <h1 className={hr.title}>Rekap kehadiran</h1>
          <p className={hr.subtitle}>
            {filterText}
            {data?.data_until ? ` · data hingga ${formatDate(data.data_until)}` : ''}
            {meta ? ` · ${meta.total} karyawan` : ''}
          </p>
        </div>
        <div className={`${styles.headerActions} ${styles.noPrint}`}>
          <Button variant="secondary" compact onClick={() => window.print()} disabled={!data}>
            Cetak / simpan PDF
          </Button>
          {canExport && (
            <Button compact onClick={() => setExportOpen(true)} disabled={exporter.state.kind === 'processing'}>
              {exporter.state.kind === 'processing' ? 'Membuat PDF…' : 'Ekspor PDF'}
              {exporter.state.kind !== 'processing' && (
                <Image src="/icons/download.svg" alt="" width={20} height={20} unoptimized className={styles.invert} />
              )}
            </Button>
          )}
        </div>
      </header>

      <div className={styles.noPrint} aria-live="polite">
        {exporter.state.kind === 'processing' && (
          <Alert tone="info" title="PDF sedang dibuat">
            Server sedang menyiapkan dokumen. Halaman ini memeriksa statusnya otomatis.
          </Alert>
        )}
        {exporter.state.kind === 'slow' && (
          <Alert tone="warning" title="PDF belum selesai">
            Pembuatan dokumen lebih lama dari biasanya.{' '}
            <button type="button" className={hr.inlineLink} onClick={() => exporter.recheck((exporter.state as { id: string }).id)}>
              Cek lagi
            </button>
          </Alert>
        )}
        {exporter.state.kind === 'failed' && (
          <Alert tone="danger" title="Ekspor PDF gagal">
            {exporter.state.message}{' '}
            <button type="button" className={hr.inlineLink} onClick={exporter.reset}>
              Tutup
            </button>
          </Alert>
        )}
        {exporter.state.kind === 'ready' && (
          <Alert tone="success" title="PDF siap diunduh">
            <a href={exporter.state.file.download_url ?? '#'} target="_blank" rel="noopener noreferrer" className={hr.inlineLink}>
              Unduh {exporter.state.file.file_name ?? 'PDF'}
            </a>
            {exporter.state.file.expires_at ? ` · tautan berlaku sampai ${formatDateTime(exporter.state.file.expires_at)}` : ''}
          </Alert>
        )}
      </div>

      <div className={`${styles.filters} ${styles.noPrint}`}>
        <SelectField
          label="Periode"
          value={month}
          onChange={(event) => update({ bulan: event.target.value === thisMonth ? '' : event.target.value, status: '' })}
          options={recentMonths().map((value) => ({ value, label: monthLabel(value) }))}
        />
        <SelectField
          label="Departemen"
          placeholder="Semua departemen"
          value={departmentId}
          onChange={(event) => update({ departemen: event.target.value })}
          options={departmentOptions}
        />
        <SelectField
          label="Status hari ini"
          placeholder="Semua status"
          value={status}
          disabled={month !== thisMonth}
          helper={month !== thisMonth ? 'Hanya untuk bulan berjalan' : ' '}
          onChange={(event) => update({ status: event.target.value })}
          options={todayStatuses.map(({ value, label }) => ({ value, label }))}
        />
      </div>

      {unavailable && (
        <Alert tone="info" title="Rekap belum tersedia di server">
          Endpoint rekap kehadiran belum aktif di backend. Halaman ini akan terisi otomatis setelah endpoint tersedia.
        </Alert>
      )}
      {recap.error && !unavailable && (
        <Alert tone="danger" title={recap.error.problem.title}>
          {recap.error.problem.detail ?? 'Rekap belum bisa dimuat.'}{' '}
          <button type="button" className={hr.inlineLink} onClick={() => void recap.reload()}>
            Coba lagi
          </button>
        </Alert>
      )}

      {data && (
        <div className={styles.stats}>
          <StatCard label="Hadir" value={String(data.stats.hadir)} meta="hari tepat waktu" />
          <StatCard label="Terlambat" value={String(data.stats.terlambat)} meta="hari" />
          <StatCard label="Izin/Cuti/Sakit" value={String(data.stats.izin)} meta="hari" />
          <StatCard label="Alpha" value={String(data.stats.alpha)} meta={`dari ${data.stats.employees} karyawan`} />
        </div>
      )}

      {(data || recap.loading) && (
        <div className={hr.tableWrap} aria-busy={recap.loading}>
          <div className={hr.tableScroll}>
            <table className={`${hr.table} ${hr.tableWide}`}>
              <thead>
                <tr>
                  <th scope="col">Karyawan</th>
                  <th scope="col">Departemen</th>
                  <th scope="col" className={styles.num}>Hadir</th>
                  <th scope="col" className={styles.num}>Terlambat</th>
                  <th scope="col" className={styles.num}>Izin</th>
                  <th scope="col" className={styles.num}>Cuti</th>
                  <th scope="col" className={styles.num}>Sakit</th>
                  <th scope="col" className={styles.num}>Alpha</th>
                  <th scope="col">Jam kerja</th>
                  <th scope="col">Lembur</th>
                  {month === thisMonth && <th scope="col">Hari ini</th>}
                </tr>
              </thead>
              <tbody>
                {rows.map((row) => {
                  const today = todayStatuses.find((item) => item.value === row.today_status);
                  return (
                    <tr key={row.user_id}>
                      <td className={hr.personName}>{row.name}</td>
                      <td>{row.department_name ?? <span className={hr.muted}>—</span>}</td>
                      <td className={styles.num}>{row.hadir}</td>
                      <td className={styles.num}>{row.terlambat}</td>
                      <td className={styles.num}>{row.izin}</td>
                      <td className={styles.num}>{row.cuti}</td>
                      <td className={styles.num}>{row.sakit}</td>
                      <td className={`${styles.num} ${row.alpha > 0 ? styles.alert : ''}`}>{row.alpha}</td>
                      <td>{shortDuration(row.total_work_minutes)}</td>
                      <td>{shortDuration(row.overtime_minutes)}</td>
                      {month === thisMonth && (
                        <td>{today ? <Badge tone={today.tone}>{today.label}</Badge> : <span className={hr.muted}>—</span>}</td>
                      )}
                    </tr>
                  );
                })}
                {!recap.loading && rows.length === 0 && (
                  <tr>
                    <td colSpan={month === thisMonth ? 11 : 10} className={hr.empty}>
                      Tidak ada data kehadiran untuk filter ini.
                    </td>
                  </tr>
                )}
                {recap.loading && rows.length === 0 && (
                  <tr>
                    <td colSpan={month === thisMonth ? 11 : 10} className={hr.empty}>
                      Memuat rekap…
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
          {meta && meta.total > 0 && (
            <div className={`${hr.pagination} ${styles.noPrint}`}>
              <p className={hr.paginationText}>
                Menampilkan {from}–{from + rows.length - 1} dari {meta.total} karyawan
              </p>
              <div className={hr.paginationNav}>
                <Button variant="secondary" compact disabled={page <= 1} onClick={() => update({ halaman: String(page - 1) })}>
                  Sebelumnya
                </Button>
                <Button
                  variant="secondary"
                  compact
                  disabled={page >= meta.total_pages}
                  onClick={() => update({ halaman: String(page + 1) })}
                >
                  Berikutnya
                </Button>
              </div>
            </div>
          )}
        </div>
      )}

      {exportOpen && (
        <ExportDialog
          initial={{ month, departmentId }}
          departments={departmentOptions}
          onClose={() => setExportOpen(false)}
          onExport={(input) => {
            setExportOpen(false);
            void exporter.start(input);
          }}
        />
      )}
    </main>
  );
}
