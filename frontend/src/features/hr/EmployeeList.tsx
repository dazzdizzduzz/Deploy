'use client';

import Image from 'next/image';
import { useEffect, useState } from 'react';
import { Alert } from '@/components/ui/Alert';
import { Avatar } from '@/components/ui/Avatar';
import { Badge } from '@/components/ui/Badge';
import { Button, ButtonLink } from '@/components/ui/Button';
import { SearchField, SelectField } from '@/components/ui/Field';
import { Toast } from '@/components/ui/Toast';
import { useAsync } from '@/lib/hooks/useAsync';
import { departmentsApi, usersApi } from './api';
import styles from './hr.module.css';
import { employmentLabels, initialsOf, roleLabels, statusBadges } from './labels';
import { EMPLOYEE_ROLES, USER_STATUSES } from './types';

const PER_PAGE = 20;

type EmployeeListProps = {
  /** owner/hrd boleh menambah dan mengubah; team_lead hanya melihat. */
  canManage: boolean;
  /** Email karyawan yang baru ditambahkan, untuk toast. */
  addedEmail?: string;
  /** Nama karyawan yang baru diperbarui, untuk toast. */
  updatedName?: string;
};

function useDebounced(value: string, delay = 300) {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(timer);
  }, [value, delay]);
  return debounced;
}

/** FR-03 · Daftar karyawan tenant dengan pencarian, filter, dan pagination dari API. */
export function EmployeeList({ canManage, addedEmail, updatedName }: EmployeeListProps) {
  const [query, setQuery] = useState('');
  const [departmentId, setDepartmentId] = useState('');
  const [role, setRole] = useState('');
  const [status, setStatus] = useState('');
  const [page, setPage] = useState(1);
  const [toast, setToast] = useState(
    addedEmail
      ? { title: 'Karyawan berhasil ditambahkan', body: `Akun untuk ${addedEmail} sudah dibuat.` }
      : updatedName
        ? { title: 'Data karyawan diperbarui', body: `Perubahan untuk ${updatedName} sudah disimpan.` }
        : null,
  );
  const search = useDebounced(query.trim());

  // Kembali ke halaman 1 setiap kali kata kunci atau filter berubah.
  useEffect(() => setPage(1), [search]);
  const filterBy = (setter: (value: string) => void) => (value: string) => {
    setter(value);
    setPage(1);
  };

  const departments = useAsync(() => departmentsApi.list({ per_page: 100 }), []);
  const users = useAsync(
    () =>
      usersApi.list({
        search,
        department_id: departmentId,
        role,
        status,
        page,
        per_page: PER_PAGE,
      }),
    [search, departmentId, role, status, page],
  );

  const rows = users.data?.data ?? [];
  const meta = users.data?.meta;
  const from = meta && meta.total > 0 ? (meta.page - 1) * meta.per_page + 1 : 0;
  const to = meta ? from + rows.length - 1 : 0;

  return (
    <main className={styles.main}>
      <header className={styles.header}>
        <div className={styles.titleBlock}>
          <h1 className={styles.title}>Karyawan</h1>
          <p className={styles.subtitle}>
            {meta ? `${meta.total} karyawan terdaftar` : 'Data akun, peran, dan jabatan karyawan.'}
          </p>
        </div>
        {canManage && (
          <ButtonLink compact href="/dashboard/karyawan/tambah">
            Tambah karyawan
            <Image src="/icons/plus-inverse.svg" alt="" width={20} height={20} unoptimized />
          </ButtonLink>
        )}
      </header>

      {toast && (
        <Toast tone="success" title={toast.title} onDismiss={() => setToast(null)}>
          {toast.body}
        </Toast>
      )}

      <div className={styles.tools}>
        <div className={styles.search}>
          <SearchField
            aria-label="Cari karyawan"
            placeholder="Cari nama atau email…"
            value={query}
            maxLength={100}
            onChange={(event) => setQuery(event.target.value)}
          />
        </div>
        <div className={styles.filter}>
          <SelectField
            label="Departemen"
            placeholder="Semua departemen"
            value={departmentId}
            onChange={(event) => filterBy(setDepartmentId)(event.target.value)}
            options={(departments.data?.data ?? []).map(({ id, name }) => ({ value: id, label: name }))}
          />
        </div>
        <div className={styles.filter}>
          <SelectField
            label="Peran"
            placeholder="Semua peran"
            value={role}
            onChange={(event) => filterBy(setRole)(event.target.value)}
            options={EMPLOYEE_ROLES.map((value) => ({ value, label: roleLabels[value] }))}
          />
        </div>
        <div className={styles.filter}>
          <SelectField
            label="Status"
            placeholder="Semua status"
            value={status}
            onChange={(event) => filterBy(setStatus)(event.target.value)}
            options={USER_STATUSES.map((value) => ({ value, label: statusBadges[value].label }))}
          />
        </div>
      </div>

      {users.error && (
        <Alert tone="danger" title={users.error.problem.title}>
          {users.error.problem.detail ?? 'Data karyawan belum bisa dimuat.'}{' '}
          <button type="button" className={styles.inlineLink} onClick={() => void users.reload()}>
            Coba lagi
          </button>
        </Alert>
      )}

      <div className={styles.tableWrap} aria-busy={users.loading}>
        <div className={styles.tableScroll}>
          <table className={`${styles.table} ${styles.tableWide}`}>
            <thead>
              <tr>
                <th scope="col">Karyawan</th>
                <th scope="col">Jabatan</th>
                <th scope="col">Departemen</th>
                <th scope="col">Peran</th>
                <th scope="col">Status kerja</th>
                <th scope="col">Status</th>
                {canManage && (
                  <th scope="col" className={styles.colActions}>
                    Aksi
                  </th>
                )}
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => {
                const badge = statusBadges[row.status];
                return (
                  <tr key={row.id}>
                    <td>
                      <div className={styles.person}>
                        <Avatar initials={initialsOf(row.name)} size="M" muted={row.status === 'nonaktif'} />
                        <div className={styles.personText}>
                          <span className={styles.personName}>{row.name}</span>
                          <span className={styles.personEmail}>{row.email}</span>
                        </div>
                      </div>
                    </td>
                    <td>{row.position_name ?? <span className={styles.muted}>—</span>}</td>
                    <td>{row.department_name ?? <span className={styles.muted}>—</span>}</td>
                    <td>{roleLabels[row.role]}</td>
                    <td>{employmentLabels[row.employment_type]}</td>
                    <td>
                      <Badge tone={badge.tone}>{badge.label}</Badge>
                    </td>
                    {canManage && (
                      <td className={styles.colActions}>
                        <ButtonLink variant="secondary" compact href={`/dashboard/karyawan/${row.id}/edit`}>
                          Edit
                        </ButtonLink>
                      </td>
                    )}
                  </tr>
                );
              })}
              {!users.loading && !users.error && rows.length === 0 && (
                <tr>
                  <td colSpan={canManage ? 7 : 6} className={styles.empty}>
                    {search || departmentId || role || status
                      ? 'Tidak ada karyawan yang cocok dengan filter.'
                      : 'Belum ada karyawan. Tambahkan karyawan pertama.'}
                  </td>
                </tr>
              )}
              {users.loading && rows.length === 0 && (
                <tr>
                  <td colSpan={canManage ? 7 : 6} className={styles.empty}>
                    Memuat data karyawan…
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        {meta && meta.total > 0 && (
          <div className={styles.pagination}>
            <p className={styles.paginationText}>
              Menampilkan {from}–{to} dari {meta.total} karyawan
            </p>
            <div className={styles.paginationNav}>
              <Button
                variant="secondary"
                compact
                disabled={page <= 1 || users.loading}
                onClick={() => setPage((value) => value - 1)}
              >
                Sebelumnya
              </Button>
              <Button
                variant="secondary"
                compact
                disabled={page >= meta.total_pages || users.loading}
                onClick={() => setPage((value) => value + 1)}
              >
                Berikutnya
              </Button>
            </div>
          </div>
        )}
      </div>
    </main>
  );
}
