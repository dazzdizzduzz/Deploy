package repository

import (
	"context"
	"testing"
	"time"
)

func TestOvertimeRepository_RequiresTenant(t *testing.T) {
	repo := &postgresOvertimeRepository{db: nil}
	ctx := context.Background()

	if err := repo.Create(ctx, &OvertimeRequest{}); err != ErrTenantRequired {
		t.Errorf("Create without tenant: got %v, want %v", err, ErrTenantRequired)
	}

	if _, err := repo.FindByID(ctx, "", "some-id"); err != ErrTenantRequired {
		t.Errorf("FindByID without tenant: got %v, want %v", err, ErrTenantRequired)
	}

	if _, _, _, err := repo.List(ctx, "", OvertimeFilter{}); err != ErrTenantRequired {
		t.Errorf("List without tenant: got %v, want %v", err, ErrTenantRequired)
	}

	if _, err := repo.Approve(ctx, "", "id", "rev"); err != ErrTenantRequired {
		t.Errorf("Approve without tenant: got %v, want %v", err, ErrTenantRequired)
	}

	if _, err := repo.Reject(ctx, "", "id", "rev", "alasan"); err != ErrTenantRequired {
		t.Errorf("Reject without tenant: got %v, want %v", err, ErrTenantRequired)
	}
}

func TestOvertimeRepository_RejectEmptyReason(t *testing.T) {
	repo := &postgresOvertimeRepository{db: nil}
	ctx := context.Background()

	_, err := repo.Reject(ctx, "tenant-id", "id", "rev", "   ")
	if err == nil {
		t.Error("expected error for empty reject reason, got nil")
	}
}

func TestOvertimeFilter_PaginationDefaults(t *testing.T) {
	f := OvertimeFilter{
		Page:    -1,
		PerPage: 200,
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage > 100 {
		f.PerPage = 20
	}
	if f.Page != 1 || f.PerPage != 20 {
		t.Errorf("unexpected pagination normalized values: page=%d perPage=%d", f.Page, f.PerPage)
	}
}

func TestOvertimeRequest_ModelFields(t *testing.T) {
	now := time.Now()
	attID := "att-1"
	dept := "IT"
	rev := "rev-1"
	revName := "Manager One"
	reason := "Tolak karena belum ada konfirmasi"

	req := OvertimeRequest{
		ID:              "ot-1",
		TenantID:        "t-1",
		AttendanceID:    &attID,
		UserID:          "u-1",
		UserName:        "User One",
		DepartmentName:  &dept,
		Date:            now,
		StartTime:       "17:30",
		EndTime:         "19:30",
		DurationMinutes: 120,
		Reason:          "Revisi proyek",
		Status:          OvertimeStatusPending,
		ReviewedBy:      &rev,
		ReviewerName:    &revName,
		ReviewedAt:      &now,
		RejectionReason: &reason,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if req.DurationMinutes != 120 {
		t.Errorf("got %d, want 120", req.DurationMinutes)
	}
	if req.Status != "menunggu" {
		t.Errorf("got %s, want menunggu", req.Status)
	}
	if *req.AttendanceID != "att-1" {
		t.Errorf("got %s, want att-1", *req.AttendanceID)
	}
}

