-- 000005_extend_users_employee_profile.up.sql
-- Migrasi: Lengkapi tabel users dengan data karyawan - ISSUE-26 (API Karyawan, FR-03)
-- Acuan: api/openapi-phase1.yaml skema UserResponse, Skema Basis Data SRS (users, positions).
-- Bergantung pada 000003 (work_schedules, office_locations).

-- =============================================================================
-- 1. Peran: ikuti 6 peran kontrak API (models.ValidRoles)
--    Constraint lama dibuat inline di 000001 sehingga namanya users_role_check.
-- =============================================================================
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('super_admin', 'owner', 'hrd', 'team_lead', 'employee', 'cashier'));

-- =============================================================================
-- 2. Kolom profil karyawan
-- =============================================================================
ALTER TABLE users
    ADD COLUMN phone                      VARCHAR(30),
    ADD COLUMN employment_type            VARCHAR(20)  NOT NULL DEFAULT 'tetap'
                                          CHECK (employment_type IN ('tetap', 'kontrak', 'magang')),
    ADD COLUMN status                     VARCHAR(20)  NOT NULL DEFAULT 'aktif'
                                          CHECK (status IN ('aktif', 'nonaktif', 'diundang')),
    ADD COLUMN joined_at                  DATE,
    ADD COLUMN manager_id                 UUID REFERENCES users(id) ON DELETE SET NULL,
    -- Penugasan jadwal & lokasi per karyawan (dipakai clock-in, FR-05/FR-06).
    ADD COLUMN work_schedule_id           UUID REFERENCES work_schedules(id) ON DELETE SET NULL,
    ADD COLUMN office_location_id         UUID REFERENCES office_locations(id) ON DELETE SET NULL,
    ADD COLUMN allow_wfh_without_approval BOOLEAN      NOT NULL DEFAULT FALSE,
    -- Data sensitif: hanya ditampilkan API untuk owner & hrd. Uang DECIMAL(12,2) sesuai konvensi.
    ADD COLUMN base_salary                DECIMAL(12,2) CHECK (base_salary >= 0),
    ADD COLUMN bank_name                  VARCHAR(100),
    ADD COLUMN bank_account_number        VARCHAR(50),
    ADD COLUMN bank_account_holder        VARCHAR(100),
    -- Data penonaktifan.
    ADD COLUMN inactive_since             DATE,
    ADD COLUMN deactivation_reason        VARCHAR(200);

-- Data lama: status mengikuti is_active.
UPDATE users SET status = CASE WHEN is_active THEN 'aktif' ELSE 'nonaktif' END;

-- =============================================================================
-- 3. Konsistensi data
-- =============================================================================
-- is_active tetap dipakai login (handler auth). Constraint ini menjamin keduanya tidak
-- pernah berbeda: hanya status 'aktif' yang boleh login.
ALTER TABLE users ADD CONSTRAINT chk_users_status_active
    CHECK ((status = 'aktif') = is_active);

-- Karyawan tidak boleh menjadi atasan dirinya sendiri.
ALTER TABLE users ADD CONSTRAINT chk_users_manager_not_self
    CHECK (manager_id IS NULL OR manager_id <> id);

-- Data penonaktifan hanya terisi saat status nonaktif.
ALTER TABLE users ADD CONSTRAINT chk_users_inactive_fields
    CHECK (status = 'nonaktif' OR (inactive_since IS NULL AND deactivation_reason IS NULL));

-- =============================================================================
-- 4. Indeks untuk filter daftar karyawan (GET /users)
-- =============================================================================
CREATE INDEX idx_users_tenant_status ON users(tenant_id, status);
CREATE INDEX idx_users_manager_id ON users(manager_id);
