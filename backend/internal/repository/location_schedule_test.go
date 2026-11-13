package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func getTestDB(t *testing.T) *sql.DB {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres integration test")
		return nil
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping integration test: cannot connect to db: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("Skipping integration test: db ping failed: %v", err)
	}

	return db
}

func TestPostgresLocationRepository_CRUDAndTenantIsolation(t *testing.T) {
	db := getTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	ctx := context.Background()
	repo := NewLocationRepository(db)

	tenantA := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11" // seeded tenant
	// Buat dummy tenant B untuk menguji isolasi RLS
	tenantB := uuid.NewString()
	_, err := db.ExecContext(ctx, "INSERT INTO tenants (id, name, slug) VALUES ($1, 'Tenant B Test', 'tenant-b-test') ON CONFLICT DO NOTHING;", tenantB)
	if err != nil {
		t.Fatalf("failed to insert tenant B: %v", err)
	}
	defer func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM tenants WHERE id = $1;", tenantB)
	}()

	// 1. Create Location di Tenant A
	loc := &OfficeLocation{
		TenantID:       tenantA,
		Name:           "Kantor Cabang Denpasar",
		Address:        "Jl. Teuku Umar No. 100",
		Latitude:       -8.6705,
		Longitude:      115.2126,
		RadiusMeters:   150,
		SelfieRequired: true,
		IsActive:       true,
	}

	if err := repo.Create(ctx, loc); err != nil {
		t.Fatalf("failed to create location: %v", err)
	}
	if loc.ID == "" {
		t.Fatal("expected location ID to be generated")
	}
	defer func() {
		_ = repo.Delete(ctx, tenantA, loc.ID)
	}()

	// 2. FindByID di Tenant A -> Harus ketemu
	found, err := repo.FindByID(ctx, tenantA, loc.ID)
	if err != nil {
		t.Fatalf("failed to find location in tenant A: %v", err)
	}
	if found == nil || found.Name != loc.Name {
		t.Fatalf("expected location found, got: %+v", found)
	}

	// 3. FindByID di Tenant B (RLS test) -> Harus nil (tidak boleh melihat data tenant A)
	foundB, err := repo.FindByID(ctx, tenantB, loc.ID)
	if err != nil {
		t.Fatalf("unexpected error querying from tenant B: %v", err)
	}
	if foundB != nil {
		t.Fatalf("RLS isolation failure: tenant B was able to see tenant A's location!")
	}

	// 4. Update di Tenant A
	loc.Name = "Kantor Cabang Denpasar Baru"
	loc.RadiusMeters = 200
	if err := repo.Update(ctx, loc); err != nil {
		t.Fatalf("failed to update location: %v", err)
	}

	updated, err := repo.FindByID(ctx, tenantA, loc.ID)
	if err != nil || updated == nil || updated.Name != "Kantor Cabang Denpasar Baru" || updated.RadiusMeters != 200 {
		t.Fatalf("update was not saved properly: %+v", updated)
	}

	// 5. Delete di Tenant A
	if err := repo.Delete(ctx, tenantA, loc.ID); err != nil {
		t.Fatalf("failed to delete location: %v", err)
	}

	afterDel, err := repo.FindByID(ctx, tenantA, loc.ID)
	if err != nil || afterDel != nil {
		t.Fatalf("expected deleted location to not be found: %v, %+v", err, afterDel)
	}
}

func TestPostgresScheduleRepository_CRUDAndTenantIsolation(t *testing.T) {
	db := getTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	ctx := context.Background()
	repo := NewScheduleRepository(db)

	tenantA := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	tenantB := uuid.NewString()
	_, err := db.ExecContext(ctx, "INSERT INTO tenants (id, name, slug) VALUES ($1, 'Tenant B Sched Test', 'tenant-b-sched-test') ON CONFLICT DO NOTHING;", tenantB)
	if err != nil {
		t.Fatalf("failed to insert tenant B: %v", err)
	}
	defer func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM tenants WHERE id = $1;", tenantB)
	}()

	// 1. Create Schedule di Tenant A
	sched := &WorkSchedule{
		TenantID:             tenantA,
		Name:                 "Jadwal Operasional Khusus",
		StartTime:            "08:30",
		EndTime:              "17:30",
		ToleranceMinutes:     15,
		OvertimeAfterMinutes: 60,
		MaxOvertimeMinutes:   180,
		WorkingDays:          "Sen,Sel,Rab,Kam,Jum",
		AllowRemote:          true,
		IsActive:             true,
	}

	if err := repo.Create(ctx, sched); err != nil {
		t.Fatalf("failed to create work schedule: %v", err)
	}
	if sched.ID == "" {
		t.Fatal("expected schedule ID to be generated")
	}
	defer func() {
		_ = repo.Delete(ctx, tenantA, sched.ID)
	}()

	// 2. FindByID di Tenant A
	found, err := repo.FindByID(ctx, tenantA, sched.ID)
	if err != nil {
		t.Fatalf("failed to find schedule: %v", err)
	}
	if found == nil || found.StartTime != "08:30" || found.EndTime != "17:30" {
		t.Fatalf("expected schedule 08:30 - 17:30, got: %+v", found)
	}

	// 3. FindByID di Tenant B (RLS test) -> Harus nil
	foundB, err := repo.FindByID(ctx, tenantB, sched.ID)
	if err != nil {
		t.Fatalf("unexpected error querying from tenant B: %v", err)
	}
	if foundB != nil {
		t.Fatalf("RLS isolation failure: tenant B was able to see tenant A's schedule!")
	}

	// 4. Update di Tenant A
	sched.Name = "Jadwal Operasional Diperbarui"
	sched.StartTime = "09:00"
	sched.EndTime = "18:00"
	if err := repo.Update(ctx, sched); err != nil {
		t.Fatalf("failed to update schedule: %v", err)
	}

	updated, err := repo.FindByID(ctx, tenantA, sched.ID)
	if err != nil || updated == nil || updated.StartTime != "09:00" || updated.EndTime != "18:00" {
		t.Fatalf("expected updated schedule 09:00 - 18:00, got: %+v", updated)
	}

	// 5. Delete di Tenant A
	if err := repo.Delete(ctx, tenantA, sched.ID); err != nil {
		t.Fatalf("failed to delete schedule: %v", err)
	}

	afterDel, err := repo.FindByID(ctx, tenantA, sched.ID)
	if err != nil || afterDel != nil {
		t.Fatalf("expected deleted schedule to not be found: %v, %+v", err, afterDel)
	}
}
