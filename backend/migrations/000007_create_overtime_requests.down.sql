-- 000007_create_overtime_requests.down.sql
-- Rollback: Hapus kebijakan RLS dan tabel overtime_requests.

DROP POLICY IF EXISTS tenant_isolation_overtime_requests ON overtime_requests;
DROP TABLE IF EXISTS overtime_requests CASCADE;
