-- 000006_create_notifications.up.sql
-- Migrasi: Tabel notifications & notification_device_tokens + RLS - ISSUE-82 (FR-14)
-- Dipakai notifikasi status izin (FR-15) dan task (FR-20): in-app, email (SMTP), push (FCM).

-- =============================================================================
-- 1. notifications: kotak masuk notifikasi in-app per karyawan
-- =============================================================================
CREATE TABLE notifications (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- Penerima notifikasi. Bila akun dihapus, notifikasinya ikut terhapus.
    user_id     UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- Jenis event, huruf kecil dipisah titik, contoh: leave.approved, task.overdue, announcement.
    type        VARCHAR(50)  NOT NULL
                CHECK (type ~ '^[a-z][a-z_]*(\.[a-z][a-z_]*)*$'),
    title       VARCHAR(150) NOT NULL CHECK (char_length(title) > 0),
    body        TEXT         NOT NULL CHECK (char_length(body) BETWEEN 1 AND 1000),
    -- Referensi tambahan untuk klien, contoh {"leave_request_id": "..."}. Jangan isi data sensitif.
    data        JSONB        NOT NULL DEFAULT '{}'::jsonb
                CHECK (jsonb_typeof(data) = 'object'),
    read_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Daftar notifikasi terbaru milik satu karyawan (GET /me/notifications).
CREATE INDEX idx_notifications_user_created ON notifications(tenant_id, user_id, created_at DESC);
-- Hitung belum dibaca dengan cepat; hanya baris belum dibaca yang masuk indeks.
CREATE INDEX idx_notifications_user_unread ON notifications(tenant_id, user_id) WHERE read_at IS NULL;

-- =============================================================================
-- 2. notification_device_tokens: token FCM per perangkat
-- =============================================================================
CREATE TABLE notification_device_tokens (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id     UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id       UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token         TEXT         NOT NULL CHECK (char_length(token) BETWEEN 10 AND 4096),
    platform      VARCHAR(10)  NOT NULL CHECK (platform IN ('android', 'ios', 'web')),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_seen_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    -- Satu token hanya milik satu perangkat. Bila perangkat ganti akun, baris diperbarui
    -- (upsert) sehingga push tidak terkirim ke akun lama.
    CONSTRAINT uq_notification_device_tokens_token UNIQUE (token)
);

CREATE INDEX idx_notification_device_tokens_user ON notification_device_tokens(tenant_id, user_id);

-- =============================================================================
-- 3. Row-Level Security (sama dengan migrasi lain: app.current_tenant)
-- =============================================================================
ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_device_tokens ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_notifications ON notifications
    AS PERMISSIVE
    FOR ALL
    TO PUBLIC
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::UUID);

CREATE POLICY tenant_isolation_notification_device_tokens ON notification_device_tokens
    AS PERMISSIVE
    FOR ALL
    TO PUBLIC
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::UUID);
