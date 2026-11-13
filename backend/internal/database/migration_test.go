package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrationFiles_000004_Attendances(t *testing.T) {
	// Temukan path file migrasi 000004
	candidates := []string{
		"../../migrations/000004_create_attendances.up.sql",
		"../migrations/000004_create_attendances.up.sql",
		"migrations/000004_create_attendances.up.sql",
		"backend/migrations/000004_create_attendances.up.sql",
	}

	var upPath string
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			upPath = p
			break
		}
	}

	if upPath == "" {
		t.Fatal("berkas migrasi 000004_create_attendances.up.sql tidak ditemukan")
	}

	downPath := filepath.Join(filepath.Dir(upPath), "000004_create_attendances.down.sql")
	if _, err := os.Stat(downPath); err != nil {
		t.Fatalf("berkas migrasi down tidak ditemukan di %s: %v", downPath, err)
	}

	// 1. Verifikasi isi berkas UP
	upContent, err := os.ReadFile(upPath)
	if err != nil {
		t.Fatalf("gagal membaca %s: %v", upPath, err)
	}
	upSQL := string(upContent)

	requiredUpSubstrings := []string{
		"CREATE TABLE attendances",
		"tenant_id",
		"user_id",
		"office_location_id",
		"work_schedule_id",
		"work_date",
		"work_mode",
		"clock_in",
		"clock_out",
		"latitude_in",
		"longitude_in",
		"status",
		"late_minutes",
		"early_leave_minutes",
		"overtime_minutes",
		"uq_attendances_tenant_user_date",
		"ENABLE ROW LEVEL SECURITY",
		"CREATE POLICY tenant_isolation_attendances ON attendances",
		"app.current_tenant",
	}

	for _, sub := range requiredUpSubstrings {
		if !strings.Contains(upSQL, sub) {
			t.Errorf("000004 UP migration harus mengandung klausa %q", sub)
		}
	}

	// 2. Verifikasi isi berkas DOWN
	downContent, err := os.ReadFile(downPath)
	if err != nil {
		t.Fatalf("gagal membaca %s: %v", downPath, err)
	}
	downSQL := string(downContent)

	requiredDownSubstrings := []string{
		"DROP POLICY IF EXISTS tenant_isolation_attendances ON attendances",
		"DROP TABLE IF EXISTS attendances",
	}

	for _, sub := range requiredDownSubstrings {
		if !strings.Contains(downSQL, sub) {
			t.Errorf("000004 DOWN migration harus mengandung klausa %q", sub)
		}
	}
}

func TestMigrationFiles_000006_OvertimeRequests(t *testing.T) {
	candidates := []string{
		"../../migrations/000006_create_overtime_requests.up.sql",
		"../migrations/000006_create_overtime_requests.up.sql",
		"migrations/000006_create_overtime_requests.up.sql",
		"backend/migrations/000006_create_overtime_requests.up.sql",
	}

	var upPath string
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			upPath = p
			break
		}
	}

	if upPath == "" {
		t.Fatal("berkas migrasi 000006_create_overtime_requests.up.sql tidak ditemukan")
	}

	downPath := filepath.Join(filepath.Dir(upPath), "000006_create_overtime_requests.down.sql")
	if _, err := os.Stat(downPath); err != nil {
		t.Fatalf("berkas migrasi down tidak ditemukan di %s: %v", downPath, err)
	}

	upContent, err := os.ReadFile(upPath)
	if err != nil {
		t.Fatalf("gagal membaca %s: %v", upPath, err)
	}
	upSQL := string(upContent)

	requiredUpSubstrings := []string{
		"CREATE TABLE overtime_requests",
		"tenant_id",
		"attendance_id",
		"user_id",
		"date",
		"start_time",
		"end_time",
		"duration_minutes",
		"reason",
		"status",
		"reviewed_by",
		"reviewed_at",
		"rejection_reason",
		"ENABLE ROW LEVEL SECURITY",
		"CREATE POLICY tenant_isolation_overtime_requests ON overtime_requests",
		"app.current_tenant",
	}

	for _, sub := range requiredUpSubstrings {
		if !strings.Contains(upSQL, sub) {
			t.Errorf("000006 UP migration harus mengandung klausa %q", sub)
		}
	}

	downContent, err := os.ReadFile(downPath)
	if err != nil {
		t.Fatalf("gagal membaca %s: %v", downPath, err)
	}
	downSQL := string(downContent)

	requiredDownSubstrings := []string{
		"DROP POLICY IF EXISTS tenant_isolation_overtime_requests ON overtime_requests",
		"DROP TABLE IF EXISTS overtime_requests",
	}

	for _, sub := range requiredDownSubstrings {
		if !strings.Contains(downSQL, sub) {
			t.Errorf("000006 DOWN migration harus mengandung klausa %q", sub)
		}
	}
}

