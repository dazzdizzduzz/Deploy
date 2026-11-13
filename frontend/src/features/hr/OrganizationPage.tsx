'use client';

import Image from 'next/image';
import Link from 'next/link';
import { useState, type FormEvent } from 'react';
import { Alert } from '@/components/ui/Alert';
import { Button } from '@/components/ui/Button';
import { Dialog } from '@/components/ui/Dialog';
import { SelectField, TextareaField, TextField } from '@/components/ui/Field';
import { Toast } from '@/components/ui/Toast';
import { ApiProblem } from '@/lib/api/client';
import type { Problem } from '@/lib/api/problem';
import { useAsync } from '@/lib/hooks/useAsync';
import { departmentsApi, positionsApi } from './api';
import styles from './hr.module.css';
import type { Department, Position } from './types';

export type OrganizationTab = 'departemen' | 'jabatan';

type Kind = 'department' | 'position';

type Editing = { kind: Kind; item: Department | Position | null };
type Deleting = { kind: Kind; item: Department | Position };

const kindLabel: Record<Kind, string> = { department: 'departemen', position: 'jabatan' };

/** Struktur organisasi: kelola departemen dan jabatan tenant. */
export function OrganizationPage({ tab, canManage }: { tab: OrganizationTab; canManage: boolean }) {
  const departments = useAsync(() => departmentsApi.list({ per_page: 100 }), []);
  const positions = useAsync(() => positionsApi.list({ per_page: 100 }), []);
  const [editing, setEditing] = useState<Editing | null>(null);
  const [deleting, setDeleting] = useState<Deleting | null>(null);
  const [toast, setToast] = useState<string | null>(null);

  const isDepartments = tab === 'departemen';
  const active = isDepartments ? departments : positions;
  const kind: Kind = isDepartments ? 'department' : 'position';

  function done(message: string) {
    setEditing(null);
    setDeleting(null);
    setToast(message);
    void departments.reload();
    void positions.reload();
  }

  return (
    <main className={styles.main}>
      <header className={styles.header}>
        <div className={styles.titleBlock}>
          <h1 className={styles.title}>Struktur organisasi</h1>
          <p className={styles.subtitle}>Departemen dan jabatan yang dipakai saat menambah karyawan.</p>
        </div>
        {canManage && (
          <Button compact onClick={() => setEditing({ kind, item: null })}>
            Tambah {kindLabel[kind]}
            <Image src="/icons/plus-inverse.svg" alt="" width={20} height={20} unoptimized />
          </Button>
        )}
      </header>

      {toast && (
        <Toast tone="success" title="Berhasil" onDismiss={() => setToast(null)}>
          {toast}
        </Toast>
      )}

      <nav className={styles.tabs} aria-label="Bagian struktur organisasi">
        <Link
          href="/dashboard/struktur-organisasi"
          className={`${styles.tab} ${isDepartments ? styles.tabActive : ''}`}
          aria-current={isDepartments ? 'page' : undefined}
        >
          Departemen{departments.data ? ` (${departments.data.meta.total})` : ''}
        </Link>
        <Link
          href="/dashboard/struktur-organisasi?tab=jabatan"
          className={`${styles.tab} ${!isDepartments ? styles.tabActive : ''}`}
          aria-current={!isDepartments ? 'page' : undefined}
        >
          Jabatan{positions.data ? ` (${positions.data.meta.total})` : ''}
        </Link>
      </nav>

      {active.error && (
        <Alert tone="danger" title={active.error.problem.title}>
          {active.error.problem.detail ?? `Daftar ${kindLabel[kind]} belum bisa dimuat.`}{' '}
          <button type="button" className={styles.inlineLink} onClick={() => void active.reload()}>
            Coba lagi
          </button>
        </Alert>
      )}

      <div className={styles.tableWrap} aria-busy={active.loading}>
        <div className={styles.tableScroll}>
          {isDepartments ? (
            <DepartmentTable
              rows={departments.data?.data ?? []}
              loading={departments.loading}
              canManage={canManage}
              onEdit={(item) => setEditing({ kind: 'department', item })}
              onDelete={(item) => setDeleting({ kind: 'department', item })}
            />
          ) : (
            <PositionTable
              rows={positions.data?.data ?? []}
              loading={positions.loading}
              canManage={canManage}
              onEdit={(item) => setEditing({ kind: 'position', item })}
              onDelete={(item) => setDeleting({ kind: 'position', item })}
            />
          )}
        </div>
        {active.data && active.data.meta.total > active.data.data.length && (
          <div className={styles.pagination}>
            <p className={styles.paginationText}>
              Menampilkan {active.data.data.length} dari {active.data.meta.total} {kindLabel[kind]}
            </p>
          </div>
        )}
      </div>

      {editing && (
        <EditDialog
          editing={editing}
          departments={departments.data?.data ?? []}
          onClose={() => setEditing(null)}
          onSaved={(name) =>
            done(`${editing.kind === 'department' ? 'Departemen' : 'Jabatan'} "${name}" ${editing.item ? 'diperbarui' : 'ditambahkan'}.`)
          }
        />
      )}
      {deleting && (
        <DeleteDialog
          deleting={deleting}
          onClose={() => setDeleting(null)}
          onDeleted={() => done(`${deleting.kind === 'department' ? 'Departemen' : 'Jabatan'} "${deleting.item.name}" dihapus.`)}
        />
      )}
    </main>
  );
}

type TableProps<T> = {
  rows: T[];
  loading: boolean;
  canManage: boolean;
  onEdit: (item: T) => void;
  onDelete: (item: T) => void;
};

function RowActions<T extends { name: string }>({ item, onEdit, onDelete }: { item: T } & Pick<TableProps<T>, 'onEdit' | 'onDelete'>) {
  return (
    <div className={styles.actions}>
      <Button variant="secondary" compact onClick={() => onEdit(item)} aria-label={`Edit ${item.name}`}>
        Edit
      </Button>
      <Button variant="ghost" compact onClick={() => onDelete(item)} aria-label={`Hapus ${item.name}`}>
        Hapus
      </Button>
    </div>
  );
}

function EmptyRow({ colSpan, loading, label }: { colSpan: number; loading: boolean; label: string }) {
  return (
    <tr>
      <td colSpan={colSpan} className={styles.empty}>
        {loading ? `Memuat ${label}…` : `Belum ada ${label}.`}
      </td>
    </tr>
  );
}

function DepartmentTable({ rows, loading, canManage, onEdit, onDelete }: TableProps<Department>) {
  const columns = canManage ? 4 : 3;
  return (
    <table className={styles.table}>
      <thead>
        <tr>
          <th scope="col">Departemen</th>
          <th scope="col">Deskripsi</th>
          <th scope="col">Karyawan</th>
          {canManage && (
            <th scope="col" className={styles.colActions}>
              Aksi
            </th>
          )}
        </tr>
      </thead>
      <tbody>
        {rows.map((row) => (
          <tr key={row.id}>
            <td className={styles.personName}>{row.name}</td>
            <td>{row.description || <span className={styles.muted}>—</span>}</td>
            <td>{row.employee_count ?? <span className={styles.muted}>—</span>}</td>
            {canManage && (
              <td className={styles.colActions}>
                <RowActions item={row} onEdit={onEdit} onDelete={onDelete} />
              </td>
            )}
          </tr>
        ))}
        {rows.length === 0 && <EmptyRow colSpan={columns} loading={loading} label="departemen" />}
      </tbody>
    </table>
  );
}

function PositionTable({ rows, loading, canManage, onEdit, onDelete }: TableProps<Position>) {
  const columns = canManage ? 4 : 3;
  return (
    <table className={styles.table}>
      <thead>
        <tr>
          <th scope="col">Jabatan</th>
          <th scope="col">Departemen</th>
          <th scope="col">Deskripsi</th>
          {canManage && (
            <th scope="col" className={styles.colActions}>
              Aksi
            </th>
          )}
        </tr>
      </thead>
      <tbody>
        {rows.map((row) => (
          <tr key={row.id}>
            <td className={styles.personName}>{row.name}</td>
            <td>{row.department_name || <span className={styles.muted}>—</span>}</td>
            <td>{row.description || <span className={styles.muted}>—</span>}</td>
            {canManage && (
              <td className={styles.colActions}>
                <RowActions item={row} onEdit={onEdit} onDelete={onDelete} />
              </td>
            )}
          </tr>
        ))}
        {rows.length === 0 && <EmptyRow colSpan={columns} loading={loading} label="jabatan" />}
      </tbody>
    </table>
  );
}

function toProblem(error: unknown): ApiProblem {
  return error instanceof ApiProblem
    ? error
    : new ApiProblem({ type: 'about:blank', title: 'Terjadi kesalahan', status: 0 });
}

function EditDialog({
  editing,
  departments,
  onClose,
  onSaved,
}: {
  editing: Editing;
  departments: Department[];
  onClose: () => void;
  onSaved: (name: string) => void;
}) {
  const { kind, item } = editing;
  const [name, setName] = useState(item?.name ?? '');
  const [description, setDescription] = useState(item?.description ?? '');
  const [departmentId, setDepartmentId] = useState(
    kind === 'position' ? ((item as Position | null)?.department_id ?? '') : '',
  );
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [problem, setProblem] = useState<Problem | null>(null);
  const [saving, setSaving] = useState(false);
  const label = kindLabel[kind];

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    if (saving) return;
    const trimmed = name.trim();
    if (trimmed.length < 2 || trimmed.length > 100) {
      setErrors({ name: 'Nama 2–100 karakter' });
      return;
    }
    setSaving(true);
    setProblem(null);
    try {
      const desc = description.trim();
      if (kind === 'department') {
        const body = { name: trimmed, ...(desc && { description: desc }) };
        await (item ? departmentsApi.update(item.id, body) : departmentsApi.create(body));
      } else {
        const body = { name: trimmed, department_id: departmentId || null, ...(desc && { description: desc }) };
        await (item ? positionsApi.update(item.id, body) : positionsApi.create(body));
      }
      onSaved(trimmed);
    } catch (error) {
      const apiError = toProblem(error);
      setSaving(false);
      if (apiError.status === 409) {
        setErrors({ name: `Nama ${label} sudah dipakai` });
        return;
      }
      setErrors(apiError.fieldErrors);
      setProblem(apiError.problem);
    }
  }

  return (
    <Dialog
      title={`${item ? 'Edit' : 'Tambah'} ${label}`}
      size="md"
      showClose
      onClose={onClose}
      onSubmit={handleSubmit}
      actions={
        <>
          <Button variant="secondary" compact onClick={onClose}>
            Batal
          </Button>
          <Button type="submit" disabled={saving}>
            {saving ? 'Menyimpan…' : item ? 'Simpan perubahan' : `Tambah ${label}`}
          </Button>
        </>
      }
    >
      <div className={styles.dialogFields}>
        {problem && !Object.keys(errors).length && (
          <Alert tone="danger" title={problem.title}>
            {problem.detail ?? 'Coba lagi beberapa saat lagi.'}
          </Alert>
        )}
        <TextField
          label={`Nama ${label}`}
          name="name"
          autoComplete="off"
          autoFocus
          maxLength={100}
          value={name}
          error={errors.name}
          helper=" "
          onChange={(event) => {
            setName(event.target.value);
            setErrors({});
          }}
        />
        {kind === 'position' && (
          <SelectField
            label="Departemen (opsional)"
            placeholder="Tanpa departemen"
            value={departmentId}
            error={errors.department_id}
            helper=" "
            onChange={(event) => setDepartmentId(event.target.value)}
            options={departments.map(({ id, name: deptName }) => ({ value: id, label: deptName }))}
          />
        )}
        <TextareaField
          label="Deskripsi (opsional)"
          name="description"
          maxLength={255}
          value={description}
          error={errors.description}
          helper={`${description.length}/255`}
          onChange={(event) => setDescription(event.target.value)}
        />
      </div>
    </Dialog>
  );
}

function DeleteDialog({
  deleting,
  onClose,
  onDeleted,
}: {
  deleting: Deleting;
  onClose: () => void;
  onDeleted: () => void;
}) {
  const { kind, item } = deleting;
  const [problem, setProblem] = useState<Problem | null>(null);
  const [removing, setRemoving] = useState(false);

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    if (removing) return;
    setRemoving(true);
    setProblem(null);
    try {
      await (kind === 'department' ? departmentsApi.remove(item.id) : positionsApi.remove(item.id));
      onDeleted();
    } catch (error) {
      setRemoving(false);
      setProblem(toProblem(error).problem);
    }
  }

  return (
    <Dialog
      title={`Hapus ${kindLabel[kind]} "${item.name}"?`}
      description={
        kind === 'department'
          ? 'Departemen hanya bisa dihapus bila tidak ada karyawan atau jabatan yang terhubung.'
          : 'Jabatan yang masih dipakai karyawan tidak bisa dihapus.'
      }
      onClose={onClose}
      onSubmit={handleSubmit}
      actions={
        <>
          <Button variant="secondary" compact onClick={onClose}>
            Batal
          </Button>
          <Button type="submit" variant="danger" disabled={removing}>
            {removing ? 'Menghapus…' : `Hapus ${kindLabel[kind]}`}
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
