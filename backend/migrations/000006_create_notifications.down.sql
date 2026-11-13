-- 000006_create_notifications.down.sql
-- Rollback: Hapus kebijakan RLS dan tabel notifikasi.

DROP POLICY IF EXISTS tenant_isolation_notification_device_tokens ON notification_device_tokens;
DROP POLICY IF EXISTS tenant_isolation_notifications ON notifications;

DROP TABLE IF EXISTS notification_device_tokens CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;
