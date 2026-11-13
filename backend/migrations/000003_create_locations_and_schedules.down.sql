-- 000003_create_locations_and_schedules.down.sql
-- Rollback: Hapus kebijakan RLS dan tabel office_locations serta work_schedules

DROP POLICY IF EXISTS tenant_isolation_work_schedules ON work_schedules;
DROP POLICY IF EXISTS tenant_isolation_office_locations ON office_locations;

DROP TABLE IF EXISTS work_schedules CASCADE;
DROP TABLE IF EXISTS office_locations CASCADE;

