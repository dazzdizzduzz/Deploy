-- 000004_create_attendances.down.sql
-- Rollback: Hapus kebijakan RLS dan tabel attendances

DROP POLICY IF EXISTS tenant_isolation_attendances ON attendances;
DROP TABLE IF EXISTS attendances CASCADE;
