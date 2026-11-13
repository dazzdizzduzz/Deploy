package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

var (
	// ErrAlreadyClockedIn: karyawan sudah clock-in pada tanggal kerja yang sama (FR-06).
	ErrAlreadyClockedIn = errors.New("karyawan sudah clock-in pada tanggal ini")
	// ErrAttendanceNotOpen: presensi tidak ditemukan atau sudah clock-out (FR-07).
	ErrAttendanceNotOpen = errors.New("presensi tidak ditemukan atau sudah clock-out")
	// ErrTenantRequired: tenant_id kosong; query tanpa tenant tidak boleh dijalankan.
	ErrTenantRequired = errors.New("tenant_id wajib diisi")
)

// pgUniqueViolation adalah kode SQLSTATE PostgreSQL untuk pelanggaran UNIQUE.
const pgUniqueViolation = "23505"

// Attendance memetakan satu baris tabel attendances (migrasi 000004).
// Kolom nullable memakai pointer agar NULL dan nilai nol bisa dibedakan.
type Attendance struct {
	ID               string
	TenantID         string
	UserID           string
	OfficeLocationID *string
	WorkScheduleID   *string
	WorkDate         time.Time // tanggal kerja menurut zona waktu tenant
	WorkMode         string    // office | wfh | business_trip

	ClockIn           time.Time
	ClockOut          *time.Time
	LatitudeIn        float64
	LongitudeIn       float64
	LatitudeOut       *float64
	LongitudeOut      *float64
	DistanceInMeters  *int
	DistanceOutMeters *int
	SelfieInKey       *string
	SelfieOutKey      *string

	Status            string // on_time | late | absent
	LateMinutes       int
	EarlyLeaveMinutes int
	OvertimeMinutes   int

	CreatedAt time.Time
	UpdatedAt time.Time
}

// ClockOutUpdate berisi data yang diisi saat clock-out.
type ClockOutUpdate struct {
	TenantID          string
	AttendanceID      string
	ClockOut          time.Time
	LatitudeOut       float64
	LongitudeOut      float64
	DistanceOutMeters *int
	SelfieOutKey      *string
	EarlyLeaveMinutes int
	OvertimeMinutes   int
}

type AttendanceRepository interface {
	CreateClockIn(ctx context.Context, a *Attendance) error
	FindByUserAndDate(ctx context.Context, tenantID, userID string, workDate time.Time) (*Attendance, error)
	UpdateClockOut(ctx context.Context, u ClockOutUpdate) (*Attendance, error)
}

type postgresAttendanceRepository struct {
	db *sql.DB
}

func NewAttendanceRepository(db *sql.DB) AttendanceRepository {
	return &postgresAttendanceRepository{db: db}
}

// withTenantTx menjalankan fn di dalam transaksi yang sudah menyetel app.current_tenant,
// sehingga kebijakan RLS ikut menyaring baris. set_config(..., true) hanya berlaku
// selama transaksi ini, jadi koneksi pool berikutnya tidak membawa tenant yang salah.
func (r *postgresAttendanceRepository) withTenantTx(ctx context.Context, tenantID string, fn func(tx *sql.Tx) error) error {
	if tenantID == "" {
		return ErrTenantRequired
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Rollback setelah Commit berhasil tidak berefek, jadi error-nya sengaja diabaikan.
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID); err != nil {
		return fmt.Errorf("failed to set app.current_tenant: %w", err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// attendanceColumns dipakai bersama oleh SELECT dan RETURNING agar urutan scan selalu sama.
const attendanceColumns = `
	id, tenant_id, user_id, office_location_id, work_schedule_id, work_date, work_mode,
	clock_in, clock_out, latitude_in, longitude_in, latitude_out, longitude_out,
	distance_in_meters, distance_out_meters, selfie_in_key, selfie_out_key,
	status, late_minutes, early_leave_minutes, overtime_minutes, created_at, updated_at`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAttendance(row rowScanner) (*Attendance, error) {
	var (
		a                         Attendance
		locationID, scheduleID    sql.NullString
		clockOut                  sql.NullTime
		latOut, lonOut            sql.NullFloat64
		distIn, distOut           sql.NullInt64
		selfieInKey, selfieOutKey sql.NullString
	)
	err := row.Scan(
		&a.ID, &a.TenantID, &a.UserID, &locationID, &scheduleID, &a.WorkDate, &a.WorkMode,
		&a.ClockIn, &clockOut, &a.LatitudeIn, &a.LongitudeIn, &latOut, &lonOut,
		&distIn, &distOut, &selfieInKey, &selfieOutKey,
		&a.Status, &a.LateMinutes, &a.EarlyLeaveMinutes, &a.OvertimeMinutes, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	a.OfficeLocationID = nullStringPtr(locationID)
	a.WorkScheduleID = nullStringPtr(scheduleID)
	a.SelfieInKey = nullStringPtr(selfieInKey)
	a.SelfieOutKey = nullStringPtr(selfieOutKey)
	if clockOut.Valid {
		a.ClockOut = &clockOut.Time
	}
	if latOut.Valid && lonOut.Valid {
		a.LatitudeOut, a.LongitudeOut = &latOut.Float64, &lonOut.Float64
	}
	a.DistanceInMeters = nullIntPtr(distIn)
	a.DistanceOutMeters = nullIntPtr(distOut)
	return &a, nil
}

// CreateClockIn menyimpan presensi baru. Bila karyawan sudah punya presensi di tanggal
// yang sama, constraint uq_attendances_tenant_user_date menolak dan kita kembalikan
// ErrAlreadyClockedIn. Pengecekan lewat constraint (bukan SELECT dulu) aman dari race
// condition saat dua permintaan clock-in datang bersamaan.
func (r *postgresAttendanceRepository) CreateClockIn(ctx context.Context, a *Attendance) error {
	return r.withTenantTx(ctx, a.TenantID, func(tx *sql.Tx) error {
		query := `
			INSERT INTO attendances (
				tenant_id, user_id, office_location_id, work_schedule_id, work_date, work_mode,
				clock_in, latitude_in, longitude_in, distance_in_meters, selfie_in_key,
				status, late_minutes
			) VALUES ($1, $2, $3, $4, $5::date, $6, $7, $8, $9, $10, $11, $12, $13)
			RETURNING ` + attendanceColumns
		row := tx.QueryRowContext(ctx, query,
			a.TenantID, a.UserID, a.OfficeLocationID, a.WorkScheduleID,
			a.WorkDate.Format(time.DateOnly), a.WorkMode,
			a.ClockIn, a.LatitudeIn, a.LongitudeIn, a.DistanceInMeters, a.SelfieInKey,
			a.Status, a.LateMinutes,
		)
		saved, err := scanAttendance(row)
		if err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == pgUniqueViolation {
				return ErrAlreadyClockedIn
			}
			return fmt.Errorf("failed to insert attendance: %w", err)
		}
		*a = *saved
		return nil
	})
}

// FindByUserAndDate mengambil presensi karyawan pada tanggal kerja tertentu.
// Mengembalikan (nil, nil) bila belum ada, sama seperti repository lain di paket ini.
func (r *postgresAttendanceRepository) FindByUserAndDate(ctx context.Context, tenantID, userID string, workDate time.Time) (*Attendance, error) {
	var found *Attendance
	err := r.withTenantTx(ctx, tenantID, func(tx *sql.Tx) error {
		query := `SELECT ` + attendanceColumns + `
			FROM attendances
			WHERE tenant_id = $1 AND user_id = $2 AND work_date = $3::date
			LIMIT 1`
		a, err := scanAttendance(tx.QueryRowContext(ctx, query, tenantID, userID, workDate.Format(time.DateOnly)))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return fmt.Errorf("failed to query attendance: %w", err)
		}
		found = a
		return nil
	})
	if err != nil {
		return nil, err
	}
	return found, nil
}

// UpdateClockOut mengisi data clock-out. Klausa "clock_out IS NULL" memastikan clock-out
// hanya bisa sekali; permintaan kedua tidak mengubah baris dan mendapat ErrAttendanceNotOpen.
func (r *postgresAttendanceRepository) UpdateClockOut(ctx context.Context, u ClockOutUpdate) (*Attendance, error) {
	var updated *Attendance
	err := r.withTenantTx(ctx, u.TenantID, func(tx *sql.Tx) error {
		query := `
			UPDATE attendances
			SET clock_out = $1,
			    latitude_out = $2,
			    longitude_out = $3,
			    distance_out_meters = $4,
			    selfie_out_key = $5,
			    early_leave_minutes = $6,
			    overtime_minutes = $7,
			    updated_at = NOW()
			WHERE id = $8 AND tenant_id = $9 AND clock_out IS NULL
			RETURNING ` + attendanceColumns
		a, err := scanAttendance(tx.QueryRowContext(ctx, query,
			u.ClockOut, u.LatitudeOut, u.LongitudeOut, u.DistanceOutMeters, u.SelfieOutKey,
			u.EarlyLeaveMinutes, u.OvertimeMinutes, u.AttendanceID, u.TenantID,
		))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrAttendanceNotOpen
			}
			return fmt.Errorf("failed to update clock-out: %w", err)
		}
		updated = a
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func nullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func nullIntPtr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int64)
	return &n
}
