-- 000006_create_overtime_requests.down.sql
DROP POLICY IF EXISTS tenant_isolation_overtime_requests ON overtime_requests;
DROP TABLE IF EXISTS overtime_requests CASCADE;

