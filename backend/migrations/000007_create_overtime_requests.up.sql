-- 000007_create_overtime_requests.up.sql
-- Migrasi: Tabel overtime_requests + RLS berbasis tenant - ISSUE-67 (FR-09)
-- Acuan: api/openapi-phase1.yaml /overtime-requests (status menunggu/disetujui/ditolak,
-- tidak boleh menyetujui lembur sendiri, alasan penolakan maks 200 karakter, clock-out
-- > 30 menit setelah jam pulang otomatis membuat pengajuan) dan pola persetujuan di
-- api/drafts/leave-approval.openapi.yaml (decided_by, decided_at, decision_note).
-- Bergantung pada 000004 (attendances).

CREATE TABLE overtime_requests (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id         UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- Karyawan yang lembur (pengaju).
    user_id           UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- Presensi asal untuk pengajuan otomatis dari clock-out. NULL untuk pengajuan manual.
    attendance_id     UUID         REFERENCES attendances(id) ON DELETE CASCADE,

    work_date         DATE         NOT NULL,
    source            VARCHAR(10)  NOT NULL DEFAULT 'manual'
                      CHECK (source IN ('auto', 'manual')),
    -- Rentang lembur: dari jam pulang jadwal sampai clock-out (atau sesuai pengajuan manual).
    start_at          TIMESTAMPTZ  NOT NULL,
    end_at            TIMESTAMPTZ  NOT NULL,
    duration_minutes  INT          NOT NULL CHECK (duration_minutes BETWEEN 1 AND 1440),
    reason            TEXT         CHECK (reason IS NULL OR char_length(reason) <= 500),

    status            VARCHAR(20)  NOT NULL DEFAULT 'menunggu'
                      CHECK (status IN ('menunggu', 'disetujui', 'ditolak')),
    -- Penyetuju. SET NULL bila akunnya dihapus agar riwayat keputusan tetap ada.
    decided_by        UUID         REFERENCES users(id) ON DELETE SET NULL,
    decided_at        TIMESTAMPTZ,
    -- Catatan persetujuan (opsional) atau alasan penolakan (wajib saat ditolak).
    decision_note     VARCHAR(200),

    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    -- Satu presensi paling banyak menghasilkan satu pengajuan lembur otomatis
    -- (aman bila clock-out diproses ulang). NULL (manual) boleh lebih dari satu.
    CONSTRAINT uq_overtime_requests_attendance UNIQUE (attendance_id),
    CONSTRAINT chk_overtime_time_order CHECK (end_at > start_at),
    -- Pengajuan otomatis wajib menunjuk presensi asalnya.
    CONSTRAINT chk_overtime_auto_has_attendance CHECK (source = 'manual' OR attendance_id IS NOT NULL),
    -- Data keputusan konsisten dengan status.
    CONSTRAINT chk_overtime_decision_state CHECK (
        (status = 'menunggu' AND decided_by IS NULL AND decided_at IS NULL AND decision_note IS NULL)
        OR (status IN ('disetujui', 'ditolak') AND decided_at IS NOT NULL)
    ),
    -- Penolakan wajib menyertakan alasan (kontrak: reason wajib, maks 200).
    CONSTRAINT chk_overtime_reject_reason CHECK (
        status <> 'ditolak' OR (decision_note IS NOT NULL AND char_length(btrim(decision_note)) > 0)
    ),
    -- Karyawan tidak boleh menyetujui/menolak lemburnya sendiri.
    CONSTRAINT chk_overtime_not_self_decided CHECK (decided_by IS NULL OR decided_by <> user_id)
);

-- Tab daftar per status untuk approver (GET /overtime-requests?status=&date=).
CREATE INDEX idx_overtime_requests_tenant_status_date ON overtime_requests(tenant_id, status, work_date DESC);
-- Riwayat lembur per karyawan dan rekap bulanan.
CREATE INDEX idx_overtime_requests_tenant_user_date ON overtime_requests(tenant_id, user_id, work_date DESC);

-- =============================================================================
-- Row-Level Security (sama dengan migrasi lain: app.current_tenant)
-- =============================================================================
ALTER TABLE overtime_requests ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_overtime_requests ON overtime_requests
    AS PERMISSIVE
    FOR ALL
    TO PUBLIC
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::UUID);
