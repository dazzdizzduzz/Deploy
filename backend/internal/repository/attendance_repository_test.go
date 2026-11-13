package repository

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Data seed dari migrasi 000002.
const (
	seedTenantID   = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	seedEmployeeID = "f5eebc99-9c0b-4ef8-bb6d-6bb9bd380a66"
)

// Test ini tidak butuh database: tenant kosong harus ditolak sebelum query dijalankan.
func TestAttendanceRepository_RequiresTenant(t *testing.T) {
	repo := NewAttendanceRepository(nil)
	ctx := context.Background()

	if err := repo.CreateClockIn(ctx, &Attendance{UserID: seedEmployeeID}); !errors.Is(err, ErrTenantRequired) {
		t.Fatalf("CreateClockIn err = %v, want ErrTenantRequired", err)
	}
	if _, err := repo.FindByUserAndDate(ctx, "", seedEmployeeID, time.Now()); !errors.Is(err, ErrTenantRequired) {
		t.Fatalf("FindByUserAndDate err = %v, want ErrTenantRequired", err)
	}
	if _, err := repo.UpdateClockOut(ctx, ClockOutUpdate{AttendanceID: "x"}); !errors.Is(err, ErrTenantRequired) {
		t.Fatalf("UpdateClockOut err = %v, want ErrTenantRequired", err)
	}
}

func TestPostgresAttendanceRepository_ClockInOutFlow(t *testing.T) {
	db := getTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	ctx := context.Background()
	repo := NewAttendanceRepository(db)

	// Tanggal jauh di masa depan agar tidak bentrok dengan data lain di database uji.
	workDate := time.Date(2099, time.January, 5, 0, 0, 0, 0, time.UTC)
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM attendances WHERE user_id = $1 AND work_date >= '2099-01-01'", seedEmployeeID)
	})

	dist := 42
	selfie := seedTenantID + "/selfie/clock-in-test.jpg"
	clockIn := time.Date(2099, time.January, 5, 0, 20, 0, 0, time.UTC)

	a := &Attendance{
		TenantID:         seedTenantID,
		UserID:           seedEmployeeID,
		WorkDate:         workDate,
		WorkMode:         "office",
		ClockIn:          clockIn,
		LatitudeIn:       -8.68500123,
		LongitudeIn:      115.26200456,
		DistanceInMeters: &dist,
		SelfieInKey:      &selfie,
		Status:           "late",
		LateMinutes:      20,
	}

	// 1. Clock-in pertama berhasil dan kolom default terisi.
	if err := repo.CreateClockIn(ctx, a); err != nil {
		t.Fatalf("CreateClockIn: %v", err)
	}
	if a.ID == "" || a.CreatedAt.IsZero() {
		t.Fatalf("ID/CreatedAt tidak terisi dari RETURNING: %+v", a)
	}
	if a.ClockOut != nil || a.OvertimeMinutes != 0 {
		t.Fatalf("presensi baru seharusnya belum clock-out: %+v", a)
	}
	// Presisi DECIMAL(10,8)/(11,8) harus utuh setelah round-trip.
	if a.LatitudeIn != -8.68500123 || a.LongitudeIn != 115.26200456 {
		t.Fatalf("koordinat berubah: %v, %v", a.LatitudeIn, a.LongitudeIn)
	}

	// 2. Clock-in kedua di tanggal yang sama ditolak.
	dup := *a
	dup.ID = ""
	if err := repo.CreateClockIn(ctx, &dup); !errors.Is(err, ErrAlreadyClockedIn) {
		t.Fatalf("clock-in ganda err = %v, want ErrAlreadyClockedIn", err)
	}

	// 3. Pencarian per tanggal.
	got, err := repo.FindByUserAndDate(ctx, seedTenantID, seedEmployeeID, workDate)
	if err != nil || got == nil || got.ID != a.ID {
		t.Fatalf("FindByUserAndDate = %+v, %v; want ID %s", got, err, a.ID)
	}
	none, err := repo.FindByUserAndDate(ctx, seedTenantID, seedEmployeeID, workDate.AddDate(0, 0, 1))
	if err != nil || none != nil {
		t.Fatalf("tanggal tanpa presensi = %+v, %v; want nil, nil", none, err)
	}

	// 4. Tenant lain tidak bisa melihat presensi ini (filter tenant_id + RLS).
	other, err := repo.FindByUserAndDate(ctx, "11111111-1111-1111-1111-111111111111", seedEmployeeID, workDate)
	if err != nil || other != nil {
		t.Fatalf("tenant lain = %+v, %v; want nil, nil", other, err)
	}

	// 5. Clock-out pertama berhasil.
	distOut := 15
	out := ClockOutUpdate{
		TenantID:          seedTenantID,
		AttendanceID:      a.ID,
		ClockOut:          clockIn.Add(10 * time.Hour),
		LatitudeOut:       -8.685,
		LongitudeOut:      115.262,
		DistanceOutMeters: &distOut,
		OvertimeMinutes:   60,
	}
	updated, err := repo.UpdateClockOut(ctx, out)
	if err != nil {
		t.Fatalf("UpdateClockOut: %v", err)
	}
	if updated.ClockOut == nil || !updated.ClockOut.Equal(out.ClockOut) || updated.OvertimeMinutes != 60 {
		t.Fatalf("hasil clock-out tidak sesuai: %+v", updated)
	}
	if updated.LatitudeOut == nil || *updated.DistanceOutMeters != 15 {
		t.Fatalf("koordinat/jarak clock-out tidak tersimpan: %+v", updated)
	}

	// 6. Clock-out kedua ditolak.
	if _, err := repo.UpdateClockOut(ctx, out); !errors.Is(err, ErrAttendanceNotOpen) {
		t.Fatalf("clock-out ganda err = %v, want ErrAttendanceNotOpen", err)
	}
}
