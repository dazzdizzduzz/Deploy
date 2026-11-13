-- 000005_extend_users_employee_profile.down.sql
-- Rollback: Hapus kolom profil karyawan dan kembalikan constraint peran lama.
-- Catatan: rollback gagal bila masih ada user dengan peran super_admin/team_lead/cashier;
-- ubah atau hapus user tersebut terlebih dahulu agar data tidak hilang diam-diam.

DROP INDEX IF EXISTS idx_users_manager_id;
DROP INDEX IF EXISTS idx_users_tenant_status;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS chk_users_inactive_fields,
    DROP CONSTRAINT IF EXISTS chk_users_manager_not_self,
    DROP CONSTRAINT IF EXISTS chk_users_status_active;

ALTER TABLE users
    DROP COLUMN IF EXISTS deactivation_reason,
    DROP COLUMN IF EXISTS inactive_since,
    DROP COLUMN IF EXISTS bank_account_holder,
    DROP COLUMN IF EXISTS bank_account_number,
    DROP COLUMN IF EXISTS bank_name,
    DROP COLUMN IF EXISTS base_salary,
    DROP COLUMN IF EXISTS allow_wfh_without_approval,
    DROP COLUMN IF EXISTS office_location_id,
    DROP COLUMN IF EXISTS work_schedule_id,
    DROP COLUMN IF EXISTS manager_id,
    DROP COLUMN IF EXISTS joined_at,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS employment_type,
    DROP COLUMN IF EXISTS phone;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('owner', 'hrd', 'employee'));
