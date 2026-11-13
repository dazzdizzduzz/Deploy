package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func strPtr(s string) *string { return &s }

func TestEmployeeRepository_RequiresTenant(t *testing.T) {
	repo := NewEmployeeRepository(nil)
	ctx := context.Background()
	if _, _, err := repo.List(ctx, "", EmployeeFilter{}); !errors.Is(err, ErrTenantRequired) {
		t.Fatalf("List err = %v, want ErrTenantRequired", err)
	}
	if err := repo.Create(ctx, &Employee{}, "!"); !errors.Is(err, ErrTenantRequired) {
		t.Fatalf("Create err = %v, want ErrTenantRequired", err)
	}
}

func TestEscapeLike(t *testing.T) {
	if got := escapeLike(`50%_a\b`); got != `50\%\_a\\b` {
		t.Fatalf("escapeLike = %q", got)
	}
}

func TestPostgresEmployeeRepository_Flow(t *testing.T) {
	db := getTestDB(t)
	defer func() {
		_ = db.Close()
	}()

	ctx := context.Background()
	repo := NewEmployeeRepository(db)
	suffix := uuid.NewString()[:8]
	email := "ayu-" + suffix + "@ontrack.id"
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM users WHERE email LIKE $1", "%-"+suffix+"@ontrack.id")
	})

	joined := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	e := &Employee{
		TenantID:          seedTenantID,
		FullName:          "Putu Ayu " + suffix,
		Email:             email,
		Phone:             strPtr("0812 3456 7890"),
		DepartmentID:      strPtr("b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22"),
		PositionID:        strPtr("c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33"),
		ManagerID:         strPtr("e4eebc99-9c0b-4ef8-bb6d-6bb9bd380a55"),
		Role:              "employee",
		EmploymentType:    "kontrak",
		Status:            EmployeeStatusInvited,
		JoinedAt:          &joined,
		BaseSalary:        strPtr("6500000"),
		BankName:          strPtr("BCA"),
		BankAccountNumber: strPtr("0987654321"),
	}

	// 1. Create: nama relasi dan format uang terisi dari database.
	if err := repo.Create(ctx, e, "!"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if e.ID == "" || e.DepartmentName == nil || *e.DepartmentName != "Human Resources" ||
		e.PositionName == nil || *e.PositionName != "Staff" || e.ManagerName == nil || *e.ManagerName != "HRD OnTrack" {
		t.Fatalf("relasi tidak terisi: %+v", e)
	}
	if e.BaseSalary == nil || *e.BaseSalary != "6500000.00" {
		t.Fatalf("base_salary = %v, want 6500000.00", e.BaseSalary)
	}
	if e.JoinedAt == nil || e.JoinedAt.Format(time.DateOnly) != "2026-11-01" {
		t.Fatalf("joined_at = %v", e.JoinedAt)
	}

	// 2. Email sama di tenant yang sama ditolak.
	dup := &Employee{TenantID: seedTenantID, FullName: "Duplikat", Email: email, Role: "employee",
		EmploymentType: "tetap", Status: EmployeeStatusInvited}
	if err := repo.Create(ctx, dup, "!"); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("email ganda err = %v, want ErrEmailTaken", err)
	}

	// 3. List dengan filter dan pencarian.
	list, total, err := repo.List(ctx, seedTenantID, EmployeeFilter{Search: suffix, Status: EmployeeStatusInvited})
	if err != nil || total != 1 || len(list) != 1 || list[0].ID != e.ID {
		t.Fatalf("List search = %d/%d %v", len(list), total, err)
	}
	// "%" harus dicari sebagai karakter biasa, bukan wildcard.
	if _, total, _ := repo.List(ctx, seedTenantID, EmployeeFilter{Search: "%"}); total != 0 {
		t.Fatalf("pencarian %% seharusnya 0 hasil, dapat %d", total)
	}
	all, total, err := repo.List(ctx, seedTenantID, EmployeeFilter{PerPage: 2, Page: 2})
	if err != nil || total < 4 || len(all) != 2 {
		t.Fatalf("pagination: len=%d total=%d err=%v", len(all), total, err)
	}

	// 4. Tenant lain tidak melihat karyawan ini.
	other, err := repo.FindByID(ctx, uuid.NewString(), e.ID)
	if err != nil || other != nil {
		t.Fatalf("tenant lain = %+v, %v", other, err)
	}

	// 5. Update: nonaktifkan dan kosongkan relasi.
	today := time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)
	e.Status = EmployeeStatusInactive
	e.InactiveSince = &today
	e.DeactivationReason = strPtr("Kontrak selesai")
	e.DepartmentID = nil
	if err := repo.Update(ctx, e); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if e.Status != EmployeeStatusInactive || e.DepartmentName != nil || e.InactiveSince == nil {
		t.Fatalf("hasil update tidak sesuai: %+v", e)
	}
	var isActive bool
	_ = db.QueryRowContext(ctx, "SELECT is_active FROM users WHERE id = $1", e.ID).Scan(&isActive)
	if isActive {
		t.Fatal("is_active harus ikut false saat status nonaktif")
	}

	// 6. Update id yang tidak ada.
	ghost := *e
	ghost.ID = uuid.NewString()
	if err := repo.Update(ctx, &ghost); !errors.Is(err, ErrEmployeeNotFound) {
		t.Fatalf("Update id tidak ada err = %v, want ErrEmployeeNotFound", err)
	}

	// 7. Relasi milik tenant lain / tidak ada dilaporkan.
	missing, err := repo.MissingRefs(ctx, seedTenantID, EmployeeRefs{
		DepartmentID: strPtr("b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22"),
		PositionID:   strPtr(uuid.NewString()),
		ScheduleID:   strPtr("bukan-uuid"),
	})
	if err != nil || len(missing) != 2 || missing[0] != "position_id" || missing[1] != "schedule_id" {
		t.Fatalf("MissingRefs = %v, %v", missing, err)
	}
}
