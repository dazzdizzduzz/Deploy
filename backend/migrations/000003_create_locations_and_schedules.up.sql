-- 000003_create_locations_and_schedules.up.sql
-- Migrasi: Tabel office_locations dan work_schedules
-- Dilengkapi kolom spesifik: latitude, longitude, radius_meters, start_time, end_time
-- Menggunakan UUID untuk id dan tenant_id serta menerapkan Row-Level Security (RLS)

-- =============================================================================
-- 1. Tabel office_locations (Lokasi Kantor & Geofence)
-- =============================================================================
CREATE TABLE office_locations (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID             NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(255)     NOT NULL,
    address         TEXT             NOT NULL DEFAULT '',
    latitude        DOUBLE PRECISION NOT NULL,
    longitude       DOUBLE PRECISION NOT NULL,
    radius_meters   INT              NOT NULL DEFAULT 100,
    selfie_required BOOLEAN          NOT NULL DEFAULT FALSE,
    is_active       BOOLEAN          NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_office_locations_tenant_id ON office_locations(tenant_id);

-- =============================================================================
-- 2. Tabel work_schedules (Jadwal Kerja Operasional)
-- =============================================================================
CREATE TABLE work_schedules (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id               UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name                    VARCHAR(255) NOT NULL,
    start_time              TIME         NOT NULL,
    end_time                TIME         NOT NULL,
    tolerance_minutes       INT          NOT NULL DEFAULT 15,
    overtime_after_minutes  INT          NOT NULL DEFAULT 60,
    max_overtime_minutes    INT          NOT NULL DEFAULT 180,
    working_days            VARCHAR(100) NOT NULL DEFAULT 'Sen,Sel,Rab,Kam,Jum',
    allow_remote            BOOLEAN      NOT NULL DEFAULT FALSE,
    is_active               BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_work_schedules_tenant_id ON work_schedules(tenant_id);

-- =============================================================================
-- 3. Row-Level Security (RLS) PostgreSQL
-- =============================================================================
ALTER TABLE office_locations ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_schedules ENABLE ROW LEVEL SECURITY;

-- Kebijakan RLS untuk office_locations
CREATE POLICY tenant_isolation_office_locations ON office_locations
    AS PERMISSIVE
    FOR ALL
    TO PUBLIC
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::UUID);

-- Kebijakan RLS untuk work_schedules
CREATE POLICY tenant_isolation_work_schedules ON work_schedules
    AS PERMISSIVE
    FOR ALL
    TO PUBLIC
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::UUID);

