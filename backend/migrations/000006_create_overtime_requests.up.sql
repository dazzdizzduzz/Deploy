-- 000006_create_overtime_requests.up.sql
-- Migrasi: Tabel pengajuan & persetujuan lembur (overtime_requests) - ISSUE-66/67 (FR-07, FR-09)
-- Acuan: api/openapi-phase1.yaml bagian "Lembur & Persetujuan (F04)".

CREATE TABLE overtime_requests (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    attendance_id     UUID REFERENCES attendances(id) ON DELETE SET NULL,
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date              DATE NOT NULL,
    start_time        TIME NOT NULL,
    end_time          TIME NOT NULL,
    duration_minutes  INT NOT NULL CHECK (duration_minutes > 0),
    reason            TEXT NOT NULL,
    status            VARCHAR(20) NOT NULL DEFAULT 'menunggu'
                      CHECK (status IN ('menunggu', 'disetujui', 'ditolak')),
    reviewed_by       UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at       TIMESTAMPTZ,
    rejection_reason  TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Row Level Security (RLS) untuk isolasi multi-tenant
ALTER TABLE overtime_requests ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_overtime_requests ON overtime_requests
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::UUID);

-- Indeks performa pencarian & filter
CREATE INDEX idx_overtime_requests_tenant_status ON overtime_requests(tenant_id, status);
CREATE INDEX idx_overtime_requests_user ON overtime_requests(tenant_id, user_id);
CREATE INDEX idx_overtime_requests_date ON overtime_requests(tenant_id, date);
CREATE INDEX idx_overtime_requests_attendance ON overtime_requests(attendance_id);

