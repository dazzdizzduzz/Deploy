'use client';

import Image from 'next/image';
import { useRouter } from 'next/navigation';
import { useEffect, useState, type FormEvent } from 'react';
import { Alert } from '@/components/ui/Alert';
import { Button, ButtonLink } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { SelectField, TextField } from '@/components/ui/Field';
import { ApiProblem } from '@/lib/api/client';
import type { Problem } from '@/lib/api/problem';
import type { Role } from '@/lib/auth/roles';
import { useAsync } from '@/lib/hooks/useAsync';
import { departmentsApi, positionsApi, usersApi } from './api';
import styles from './hr.module.css';
import { employmentLabels, roleLabels } from './labels';
import { EMPLOYEE_ROLES, EMPLOYMENT_TYPES, type EmployeeRole, type EmploymentType, type User } from './types';

type Values = {
  name: string;
  email: string;
  phone: string;
  role: EmployeeRole | '';
  employment_type: EmploymentType | '';
  department_id: string;
  position_id: string;
  joined_at: string;
  status: 'diundang' | 'aktif';
};

type Errors = Partial<Record<keyof Values, string>>;

const empty: Values = {
  name: '',
  email: '',
  phone: '',
  role: '',
  employment_type: '',
  department_id: '',
  position_id: '',
  joined_at: '',
  status: 'diundang',
};

const fromUser = (user: User): Values => ({
  name: user.name,
  email: user.email,
  phone: user.phone ?? '',
  role: user.role === 'super_admin' ? '' : user.role,
  employment_type: user.employment_type,
  department_id: user.department_id ?? '',
  position_id: user.position_id ?? '',
  joined_at: user.joined_at ?? '',
  status: user.status === 'aktif' ? 'aktif' : 'diundang',
});

function validate(values: Values, isCreate: boolean): Errors {
  const errors: Errors = {};
  const name = values.name.trim();
  if (name.length < 2 || name.length > 100) errors.name = 'Nama 2–100 karakter';
  if (isCreate && !/^\S+@\S+\.\S+$/.test(values.email.trim())) errors.email = 'Masukkan email yang valid';
  if (!values.role) errors.role = 'Pilih peran';
  if (!values.employment_type) errors.employment_type = 'Pilih status kerja';
  return errors;
}

type EmployeeFormProps = {
  /** Tanpa `userId` berarti menambah karyawan baru. */
  userId?: string;
  /** Peran pengguna yang sedang masuk. Hanya owner yang boleh menetapkan peran owner. */
  currentRole: Role;
};

/** FR-03 · Tambah karyawan dan tetapkan peran serta jabatan; juga dipakai untuk edit. */
export function EmployeeForm({ userId, currentRole }: EmployeeFormProps) {
  const router = useRouter();
  const isCreate = !userId;
  const [values, setValues] = useState<Values>(empty);
  const [errors, setErrors] = useState<Errors>({});
  const [problem, setProblem] = useState<Problem | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const user = useAsync(() => (userId ? usersApi.get(userId) : Promise.resolve(null)), [userId]);
  const departments = useAsync(() => departmentsApi.list({ per_page: 100 }), []);
  const positions = useAsync(
    () => positionsApi.list({ per_page: 100, department_id: values.department_id || undefined }),
    [values.department_id],
  );

  // Isi form sekali setelah data karyawan dimuat.
  useEffect(() => {
    if (user.data) setValues(fromUser(user.data));
  }, [user.data]);

  const canAssignOwner = currentRole === 'owner' || currentRole === 'super_admin';
  const roleOptions = EMPLOYEE_ROLES.filter((role) => role !== 'owner' || canAssignOwner || values.role === 'owner');

  function set<K extends keyof Values>(key: K, value: Values[K]) {
    setValues((current) => ({
      ...current,
      [key]: value,
      // Jabatan dipilih ulang bila departemen berganti.
      ...(key === 'department_id' && value !== current.department_id ? { position_id: '' } : null),
    }));
    if (errors[key]) setErrors((current) => ({ ...current, [key]: undefined }));
  }

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    if (submitting) return;
    const nextErrors = validate(values, isCreate);
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length) return;

    setSubmitting(true);
    setProblem(null);
    const role = values.role as EmployeeRole;
    const employmentType = values.employment_type as EmploymentType;
    try {
      if (isCreate) {
        const email = values.email.trim();
        await usersApi.create({
          name: values.name.trim(),
          email,
          role,
          employment_type: employmentType,
          status: values.status,
          ...(values.phone.trim() && { phone: values.phone.trim() }),
          ...(values.department_id && { department_id: values.department_id }),
          ...(values.position_id && { position_id: values.position_id }),
          ...(values.joined_at && { joined_at: values.joined_at }),
        });
        router.push(`/dashboard/karyawan?ditambahkan=${encodeURIComponent(email)}`);
      } else {
        const updated = await usersApi.update(userId, {
          name: values.name.trim(),
          role,
          employment_type: employmentType,
          phone: values.phone.trim() || null,
          department_id: values.department_id || null,
          position_id: values.position_id || null,
          joined_at: values.joined_at || null,
        });
        router.push(`/dashboard/karyawan?diperbarui=${encodeURIComponent(updated?.name ?? values.name.trim())}`);
      }
    } catch (error) {
      setSubmitting(false);
      if (!(error instanceof ApiProblem)) {
        setProblem({ type: 'about:blank', title: 'Terjadi kesalahan', status: 0 });
        return;
      }
      if (error.status === 409) {
        setErrors({ email: 'Email sudah terdaftar di perusahaan ini' });
        return;
      }
      const fieldErrors = error.fieldErrors;
      if (error.status === 422 && Object.keys(fieldErrors).length) setErrors(fieldErrors as Errors);
      setProblem(error.problem);
    }
  }

  if (user.error) {
    return (
      <main className={styles.main}>
        <Alert tone="danger" title={user.error.problem.title}>
          {user.error.problem.detail ?? 'Data karyawan tidak dapat dimuat.'}
        </Alert>
        <ButtonLink variant="secondary" compact href="/dashboard/karyawan" className={styles.back}>
          Kembali ke daftar karyawan
        </ButtonLink>
      </main>
    );
  }

  const loadingUser = !isCreate && !user.data;

  return (
    <main className={styles.main}>
      <form className={styles.form} onSubmit={handleSubmit} noValidate>
        <header className={styles.header}>
          <div className={styles.titleBlock}>
            <h1 className={styles.title}>{isCreate ? 'Tambah karyawan' : 'Edit karyawan'}</h1>
            <p className={styles.subtitle}>
              {isCreate
                ? 'Buat akun karyawan, lalu tetapkan peran dan jabatannya.'
                : loadingUser
                  ? 'Memuat data karyawan…'
                  : values.email}
            </p>
          </div>
        </header>

        {problem && (
          <Alert tone="danger" title={problem.title}>
            {problem.detail ?? 'Periksa kembali data lalu coba lagi.'}
          </Alert>
        )}

        <Card title="Data diri">
          <div className={styles.fields}>
            <TextField
              label="Nama lengkap"
              name="name"
              autoComplete="off"
              maxLength={100}
              value={values.name}
              error={errors.name}
              helper=" "
              onChange={(event) => set('name', event.target.value)}
              disabled={loadingUser}
            />
            {isCreate && (
              <TextField
                label="Email kantor"
                name="email"
                type="email"
                autoComplete="off"
                value={values.email}
                error={errors.email}
                helper="Dipakai untuk masuk ke On-Track"
                onChange={(event) => set('email', event.target.value)}
              />
            )}
            <TextField
              label="Nomor HP (opsional)"
              name="phone"
              type="tel"
              autoComplete="off"
              maxLength={30}
              value={values.phone}
              error={errors.phone}
              helper=" "
              onChange={(event) => set('phone', event.target.value)}
              disabled={loadingUser}
            />
          </div>
        </Card>

        <Card title="Peran & jabatan">
          <div className={styles.fields}>
            <SelectField
              label="Peran (akses aplikasi)"
              placeholder="Pilih peran"
              value={values.role}
              error={errors.role}
              helper=" "
              onChange={(event) => set('role', event.target.value as EmployeeRole)}
              options={roleOptions.map((role) => ({ value: role, label: roleLabels[role] }))}
              disabled={loadingUser}
            />
            <SelectField
              label="Departemen (opsional)"
              placeholder="Tanpa departemen"
              value={values.department_id}
              error={errors.department_id}
              helper={departments.error ? 'Daftar departemen belum bisa dimuat' : ' '}
              onChange={(event) => set('department_id', event.target.value)}
              options={(departments.data?.data ?? []).map(({ id, name }) => ({ value: id, label: name }))}
              disabled={loadingUser}
            />
            <SelectField
              label="Jabatan (opsional)"
              placeholder={positions.loading ? 'Memuat jabatan…' : 'Tanpa jabatan'}
              value={values.position_id}
              error={errors.position_id}
              helper={positions.error ? 'Daftar jabatan belum bisa dimuat' : ' '}
              onChange={(event) => set('position_id', event.target.value)}
              options={(positions.data?.data ?? []).map(({ id, name }) => ({ value: id, label: name }))}
              disabled={loadingUser || positions.loading}
            />
          </div>
        </Card>

        <Card title="Status kerja">
          <div className={styles.fields}>
            <SelectField
              label="Status kerja"
              placeholder="Pilih status kerja"
              value={values.employment_type}
              error={errors.employment_type}
              helper=" "
              onChange={(event) => set('employment_type', event.target.value as EmploymentType)}
              options={EMPLOYMENT_TYPES.map((type) => ({ value: type, label: employmentLabels[type] }))}
              disabled={loadingUser}
            />
            <TextField
              label="Tanggal bergabung (opsional)"
              name="joined_at"
              type="date"
              value={values.joined_at}
              error={errors.joined_at}
              helper=" "
              onChange={(event) => set('joined_at', event.target.value)}
              disabled={loadingUser}
            />
            {isCreate && (
              <SelectField
                label="Status akun"
                value={values.status}
                helper={values.status === 'diundang' ? 'Karyawan mengaktifkan akun lewat undangan' : ' '}
                onChange={(event) => set('status', event.target.value as Values['status'])}
                options={[
                  { value: 'diundang', label: 'Diundang' },
                  { value: 'aktif', label: 'Aktif' },
                ]}
              />
            )}
          </div>
        </Card>

        <div className={styles.formActions}>
          <ButtonLink variant="secondary" href="/dashboard/karyawan">
            Batal
          </ButtonLink>
          <Button type="submit" disabled={submitting || loadingUser}>
            {submitting ? 'Menyimpan…' : isCreate ? 'Tambah karyawan' : 'Simpan perubahan'}
            {!submitting && isCreate && (
              <Image src="/icons/plus-inverse.svg" alt="" width={20} height={20} unoptimized />
            )}
          </Button>
        </div>
      </form>
    </main>
  );
}
