'use client';

import { useEffect, useState } from 'react';
import { Alert } from '@/components/ui/Alert';
import { Avatar } from '@/components/ui/Avatar';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { SearchField, SelectField } from '@/components/ui/Field';
import { Toast } from '@/components/ui/Toast';
import hr from '@/features/hr/hr.module.css';
import { departmentsApi } from '@/features/hr/api';
import { initialsOf } from '@/features/hr/labels';
import { formatDateTime } from '@/lib/format';
import { useAsync } from '@/lib/hooks/useAsync';
import { leaveApprovalApi, type LeaveApprovalItem, type LeaveStatus, type LeaveType } from './api';
import { formatRange, leaveStatusBadges, leaveTypeLabels } from './labels';
import { LeaveDetailDrawer } from './LeaveDetailDrawer';
import styles from './leave.module.css';

const PER_PAGE = 20;

const tabs: { value: LeaveStatus | 'semua'; label: string }[] = [
  { value: 'menunggu', label: 'Menunggu' },
  { value: 'disetujui', label: 'Disetujui' },
  { value: 'ditolak', label: 'Ditolak' },
  { value: 'semua', label: 'Semua' },
];

function useDebounced(value: string, delay = 300) {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(timer);
  }, [value, delay]);
  return debounced;
}

/** FR-14 · Antrean pengajuan izin, sakit, dan cuti untuk diproses atasan atau HRD. */
export function LeaveApproval() {
  const [status, setStatus] = useState<LeaveStatus | 'semua'>('menunggu');
  const [type, setType] = useState<LeaveType | ''>('');
  const [departmentId, setDepartmentId] = useState('');
  const [query, setQuery] = useState('');
  const [page, setPage] = useState(1);
  const [selected, setSelected] = useState<LeaveApprovalItem | null>(null);
  const [toast, setToast] = useState<{ title: string; body: string } | null>(null);
  const search = useDebounced(query.trim());

  useEffect(() => setPage(1), [search]);
  const filterBy =
    <T,>(setter: (value: T) => void) =>
    (value: T) => {
      setter(value);
      setPage(1);
    };

  const departments = useAsync(() => departmentsApi.list({ per_page: 100 }), []);
  const list = useAsync(
    () =>
      leaveApprovalApi.list({
        status: status === 'semua' ? undefined : status,
        type: type || undefined,
        department_id: departmentId || undefined,
        q: search || undefined,
        page,
        per_page: PER_PAGE,
      }),
    [status, type, departmentId, search, page],
  );

  const unavailable = list.error && (list.error.status === 404 || list.error.status === 405);
  const rows = list.data?.data ?? [];
  const meta = list.data?.meta;
  const counts = list.data?.counts;
  const from = meta && meta.total > 0 ? (meta.page - 1) * meta.per_page + 1 : 0;
  const filtered = Boolean(type || departmentId || search);

  function handleDecided(item: LeaveApprovalItem) {
    setSelected(null);
    setToast(
      item.status === 'disetujui'
        ? { title: 'Pengajuan disetujui', body: `${leaveTypeLabels[item.type]} ${item.employee.name} sudah disetujui.` }
        : { title: 'Pengajuan ditolak', body: `${item.employee.name} akan menerima alasan penolakan.` },
    );
    void list.reload();
  }

  return (
    <main className={hr.main}>
      <header className={hr.header}>
        <div className={hr.titleBlock}>
          <h1 className={hr.title}>Persetujuan izin</h1>
          <p className={hr.subtitle}>
            {counts
              ? `${counts.menunggu} pengajuan menunggu keputusan`
              : 'Tinjau pengajuan izin, sakit, dan cuti dari karyawan.'}
          </p>
        </div>
      </header>

      {toast && (
        <Toast tone="success" title={toast.title} onDismiss={() => setToast(null)}>
          {toast.body}
        </Toast>
      )}

      <div className={styles.tabs} role="group" aria-label="Status pengajuan">
        {tabs.map((tab) => {
          const active = tab.value === status;
          const count = counts && tab.value !== 'semua' ? counts[tab.value] : undefined;
          return (
            <button
              key={tab.value}
              type="button"
              aria-pressed={active}
              className={`${styles.tab} ${active ? styles.tabActive : ''}`}
              onClick={() => filterBy(setStatus)(tab.value)}
            >
              {tab.label}
              {count !== undefined && <span className={styles.count}>{count}</span>}
            </button>
          );
        })}
      </div>

      <div className={hr.tools}>
        <div className={hr.search}>
          <SearchField
            aria-label="Cari nama karyawan"
            placeholder="Cari nama karyawan…"
            value={query}
            maxLength={100}
            onChange={(event) => setQuery(event.target.value)}
          />
        </div>
        <div className={hr.filter}>
          <SelectField
            label="Jenis"
            placeholder="Semua jenis"
            value={type}
            onChange={(event) => filterBy(setType)(event.target.value as LeaveType | '')}
            options={(['cuti', 'izin', 'sakit'] as const).map((value) => ({ value, label: leaveTypeLabels[value] }))}
          />
        </div>
        <div className={hr.filter}>
          <SelectField
            label="Departemen"
            placeholder="Semua departemen"
            value={departmentId}
            onChange={(event) => filterBy(setDepartmentId)(event.target.value)}
            options={(departments.data?.data ?? []).map(({ id, name }) => ({ value: id, label: name }))}
          />
        </div>
      </div>

      {unavailable && (
        <Alert tone="info" title="Persetujuan izin belum tersedia di server">
          Endpoint pengajuan izin belum aktif di backend. Halaman ini akan terisi otomatis setelah endpoint tersedia.
        </Alert>
      )}
      {list.error && !unavailable && (
        <Alert tone="danger" title={list.error.problem.title}>
          {list.error.problem.detail ?? 'Daftar pengajuan belum bisa dimuat.'}{' '}
          <button type="button" className={hr.inlineLink} onClick={() => void list.reload()}>
            Coba lagi
          </button>
        </Alert>
      )}

      {(list.data || list.loading) && !unavailable && (
        <div className={hr.tableWrap} aria-busy={list.loading}>
          <div className={hr.tableScroll}>
            <table className={`${hr.table} ${hr.tableWide}`}>
              <thead>
                <tr>
                  <th scope="col">Karyawan</th>
                  <th scope="col">Jenis</th>
                  <th scope="col">Tanggal</th>
                  <th scope="col">Lama</th>
                  <th scope="col">Diajukan</th>
                  <th scope="col">Status</th>
                  <th scope="col" className={hr.colActions}>
                    <span className={styles.srOnly}>Aksi</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {rows.map((row) => {
                  const badge = leaveStatusBadges[row.status];
                  return (
                    <tr key={row.id}>
                      <td>
                        <div className={hr.person}>
                          <Avatar initials={initialsOf(row.employee.name)} size="M" />
                          <div className={hr.personText}>
                            <span className={hr.personName}>{row.employee.name}</span>
                            <span className={hr.personEmail}>{row.employee.department_name ?? 'Tanpa departemen'}</span>
                          </div>
                        </div>
                      </td>
                      <td>
                        {leaveTypeLabels[row.type]}
                        {row.attachment && <span className={styles.attachmentMark}> · lampiran</span>}
                      </td>
                      <td>{formatRange(row.start_date, row.end_date)}</td>
                      <td>{row.working_days} hari</td>
                      <td className={hr.muted}>{formatDateTime(row.created_at)}</td>
                      <td>
                        <Badge tone={badge.tone}>{badge.label}</Badge>
                      </td>
                      <td className={hr.colActions}>
                        <Button
                          variant={row.status === 'menunggu' ? 'primary' : 'secondary'}
                          compact
                          aria-label={`${row.status === 'menunggu' ? 'Tinjau' : 'Lihat'} pengajuan ${row.employee.name}`}
                          onClick={() => setSelected(row)}
                        >
                          {row.status === 'menunggu' ? 'Tinjau' : 'Lihat'}
                        </Button>
                      </td>
                    </tr>
                  );
                })}
                {!list.loading && rows.length === 0 && (
                  <tr>
                    <td colSpan={7} className={hr.empty}>
                      {filtered
                        ? 'Tidak ada pengajuan yang cocok dengan filter ini.'
                        : status === 'menunggu'
                          ? 'Tidak ada pengajuan yang menunggu keputusan.'
                          : 'Belum ada pengajuan.'}
                    </td>
                  </tr>
                )}
                {list.loading && rows.length === 0 && (
                  <tr>
                    <td colSpan={7} className={hr.empty}>
                      Memuat pengajuan…
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
          {meta && meta.total > 0 && (
            <div className={hr.pagination}>
              <p className={hr.paginationText}>
                Menampilkan {from}–{from + rows.length - 1} dari {meta.total} pengajuan
              </p>
              <div className={hr.paginationNav}>
                <Button variant="secondary" compact disabled={page <= 1} onClick={() => setPage(page - 1)}>
                  Sebelumnya
                </Button>
                <Button variant="secondary" compact disabled={page >= meta.total_pages} onClick={() => setPage(page + 1)}>
                  Berikutnya
                </Button>
              </div>
            </div>
          )}
        </div>
      )}

      {selected && <LeaveDetailDrawer item={selected} onClose={() => setSelected(null)} onDecided={handleDecided} />}
    </main>
  );
}
