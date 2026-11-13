-- 000004_create_attendances.up.sql
-- Migrasi: Tabel attendances (presensi harian karyawan) - ISSUE-63
-- Acuan: Skema Basis Data SRS bagian 3.2 (attendances), FR-06 s.d. FR-09, NFR-01.
-- Bergantung pada 000003 (office_locations, work_schedules).

-- =============================================================================
-- 1. Tabel attendances
-- =============================================================================
CREATE TABLE attendances (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id           UUID          NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id             UUID          NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Lokasi dan jadwal yang dipakai saat clock-in. NULL untuk mode WFH/dinas luar
    -- atau bila data acuan dihapus; riwayat absensi tetap utuh (ON DELETE SET NULL).
    office_location_id  UUID          REFERENCES office_locations(id) ON DELETE SET NULL,
    work_schedule_id    UUID          REFERENCES work_schedules(id) ON DELETE SET NULL,

    -- Tanggal kerja menurut zona waktu tenant, bukan UTC.
    -- Dipakai untuk aturan satu presensi per karyawan per hari.
    work_date           DATE          NOT NULL,
    work_mode           VARCHAR(20)   NOT NULL DEFAULT 'office'
                        CHECK (work_mode IN ('office', 'wfh', 'business_trip')),

    clock_in            TIMESTAMPTZ   NOT NULL,
    clock_out           TIMESTAMPTZ,

    -- Presisi koordinat mengikuti dokumen skema: 8 angka desimal (sekitar 1 mm).
    latitude_in         DECIMAL(10,8) NOT NULL CHECK (latitude_in BETWEEN -90 AND 90),
    longitude_in        DECIMAL(11,8) NOT NULL CHECK (longitude_in BETWEEN -180 AND 180),
    latitude_out        DECIMAL(10,8) CHECK (latitude_out BETWEEN -90 AND 90),
    longitude_out       DECIMAL(11,8) CHECK (longitude_out BETWEEN -180 AND 180),

    -- Jarak ke titik pusat kantor saat clock-in/out (meter), disimpan untuk audit geofence.
    distance_in_meters  INT           CHECK (distance_in_meters >= 0),
    distance_out_meters INT           CHECK (distance_out_meters >= 0),

    -- Object key di storage S3/MinIO: {tenant_id}/selfie/{nama}. Bukan URL publik.
    selfie_in_key       TEXT,
    selfie_out_key      TEXT,

    status              VARCHAR(20)   NOT NULL
                        CHECK (status IN ('on_time', 'late', 'absent')),
    late_minutes        INT           NOT NULL DEFAULT 0 CHECK (late_minutes >= 0),
    early_leave_minutes INT           NOT NULL DEFAULT 0 CHECK (early_leave_minutes >= 0),
    overtime_minutes    INT           NOT NULL DEFAULT 0 CHECK (overtime_minutes >= 0),

    created_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    -- Satu karyawan hanya boleh punya satu presensi per tanggal kerja.
    CONSTRAINT uq_attendances_tenant_user_date UNIQUE (tenant_id, user_id, work_date),

    -- Clock-out tidak boleh lebih awal dari clock-in.
    CONSTRAINT chk_attendances_clock_order CHECK (clock_out IS NULL OR clock_out >= clock_in),

    -- Koordinat clock-out diisi berpasangan.
    CONSTRAINT chk_attendances_out_coords CHECK (
        (latitude_out IS NULL AND longitude_out IS NULL)
        OR (latitude_out IS NOT NULL AND longitude_out IS NOT NULL)
    )
);

-- Rekap per tenant per tanggal (FR-11) dan riwayat per karyawan (FR-10).
-- Indeks unik di atas sudah mencakup pencarian (tenant_id, user_id, work_date).
CREATE INDEX idx_attendances_tenant_date ON attendances(tenant_id, work_date);
CREATE INDEX idx_attendances_user_id ON attendances(user_id);

-- =============================================================================
-- 2. Row-Level Security (RLS) PostgreSQL
--    Nama variabel sesi mengikuti migrasi 000001 dan 000003 (app.current_tenant).
-- =============================================================================
ALTER TABLE attendances ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_attendances ON attendances
    AS PERMISSIVE
    FOR ALL
    TO PUBLIC
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::UUID);
