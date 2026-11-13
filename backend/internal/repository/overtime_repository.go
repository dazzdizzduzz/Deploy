package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrOvertimeNotFound: pengajuan lembur tidak ditemukan di tenant ini (404).
	ErrOvertimeNotFound = errors.New("pengajuan lembur tidak ditemukan")
	// ErrOvertimeAlreadyProcessed: pengajuan sudah disetujui atau ditolak sebelumnya (409).
	ErrOvertimeAlreadyProcessed = errors.New("pengajuan lembur sudah diputuskan sebelumnya")
	// ErrOvertimeSelfApproval: atasan tidak boleh menyetujui/menolak lemburnya sendiri (403).
	ErrOvertimeSelfApproval = errors.New("karyawan tidak boleh menyetujui atau menolak lembur miliknya sendiri")
)

// Status pengajuan lembur
const (
	OvertimeStatusPending  = "menunggu"
	OvertimeStatusApproved = "disetujui"
	OvertimeStatusRejected = "ditolak"
)

// OvertimeRequest merepresentasikan baris di tabel overtime_requests beserta relasi user & reviewer.
type OvertimeRequest struct {
	ID              string
	TenantID        string
	AttendanceID    *string
	UserID          string
	UserName        string
	DepartmentName  *string
	Date            time.Time
	StartTime       string
	EndTime         string
	DurationMinutes int
	Reason          string
	Status          string
	ReviewedBy      *string
	ReviewerName    *string
	ReviewedAt      *time.Time
	RejectionReason *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// OvertimeFilter berisi parameter penyaringan pengajuan lembur (GET /overtime-requests).
type OvertimeFilter struct {
	Status       string
	DepartmentID string
	Date         *time.Time
	ManagerID    string // bila diisi, batasi ke bawahan manager ini (team_lead)
	Page         int
	PerPage      int
}

type OvertimeRepository interface {
	Create(ctx context.Context, req *OvertimeRequest) error
	FindByID(ctx context.Context, tenantID, id string) (*OvertimeRequest, error)
	List(ctx context.Context, tenantID string, f OvertimeFilter) ([]*OvertimeRequest, int, map[string]int, error)
	Approve(ctx context.Context, tenantID, id, reviewerID string) (*OvertimeRequest, error)
	Reject(ctx context.Context, tenantID, id, reviewerID, reason string) (*OvertimeRequest, error)
	FindByAttendanceID(ctx context.Context, tenantID, attendanceID string) (*OvertimeRequest, error)
}

type postgresOvertimeRepository struct {
	db *sql.DB
}

func NewOvertimeRepository(db *sql.DB) OvertimeRepository {
	return &postgresOvertimeRepository{db: db}
}

func (r *postgresOvertimeRepository) withTenantTx(ctx context.Context, tenantID string, fn func(tx *sql.Tx) error) error {
	if tenantID == "" {
		return ErrTenantRequired
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
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

func (r *postgresOvertimeRepository) Create(ctx context.Context, req *OvertimeRequest) error {
	if req.TenantID == "" {
		return ErrTenantRequired
	}
	if req.Status == "" {
		req.Status = OvertimeStatusPending
	}

	query := `
		INSERT INTO overtime_requests (
			tenant_id, attendance_id, user_id, date, start_time, end_time,
			duration_minutes, reason, status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		RETURNING id, created_at, updated_at;
	`

	return r.withTenantTx(ctx, req.TenantID, func(tx *sql.Tx) error {
		var attID sql.NullString
		if req.AttendanceID != nil && *req.AttendanceID != "" {
			attID = sql.NullString{String: *req.AttendanceID, Valid: true}
		}

		err := tx.QueryRowContext(ctx, query,
			req.TenantID,
			attID,
			req.UserID,
			req.Date,
			req.StartTime,
			req.EndTime,
			req.DurationMinutes,
			req.Reason,
			req.Status,
		).Scan(&req.ID, &req.CreatedAt, &req.UpdatedAt)

		if err != nil {
			return fmt.Errorf("failed to insert overtime_request: %w", err)
		}
		return nil
	})
}

func (r *postgresOvertimeRepository) FindByID(ctx context.Context, tenantID, id string) (*OvertimeRequest, error) {
	if tenantID == "" {
		return nil, ErrTenantRequired
	}

	query := `
		SELECT
			o.id, o.tenant_id, o.attendance_id, o.user_id,
			COALESCE(u.full_name, u.email) AS user_name,
			d.name AS department_name,
			o.date,
			to_char(o.start_time, 'HH24:MI') AS start_time,
			to_char(o.end_time, 'HH24:MI') AS end_time,
			o.duration_minutes, o.reason, o.status,
			o.reviewed_by,
			COALESCE(rev.full_name, rev.email) AS reviewer_name,
			o.reviewed_at, o.rejection_reason,
			o.created_at, o.updated_at
		FROM overtime_requests o
		JOIN users u ON u.id = o.user_id AND u.tenant_id = o.tenant_id
		LEFT JOIN departments d ON d.id = u.department_id AND d.tenant_id = u.tenant_id
		LEFT JOIN users rev ON rev.id = o.reviewed_by AND rev.tenant_id = o.tenant_id
		WHERE o.tenant_id = $1 AND o.id = $2
		LIMIT 1;
	`

	var o OvertimeRequest
	var attID, deptName, revID, revName, rejReason sql.NullString
	var revAt sql.NullTime

	err := r.withTenantTx(ctx, tenantID, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, query, tenantID, id).Scan(
			&o.ID, &o.TenantID, &attID, &o.UserID,
			&o.UserName, &deptName,
			&o.Date, &o.StartTime, &o.EndTime,
			&o.DurationMinutes, &o.Reason, &o.Status,
			&revID, &revName, &revAt, &rejReason,
			&o.CreatedAt, &o.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrOvertimeNotFound
			}
			return fmt.Errorf("failed to query overtime_request by id: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if attID.Valid {
		o.AttendanceID = &attID.String
	}
	if deptName.Valid {
		o.DepartmentName = &deptName.String
	}
	if revID.Valid {
		o.ReviewedBy = &revID.String
	}
	if revName.Valid {
		o.ReviewerName = &revName.String
	}
	if revAt.Valid {
		o.ReviewedAt = &revAt.Time
	}
	if rejReason.Valid {
		o.RejectionReason = &rejReason.String
	}

	return &o, nil
}

func (r *postgresOvertimeRepository) FindByAttendanceID(ctx context.Context, tenantID, attendanceID string) (*OvertimeRequest, error) {
	if tenantID == "" {
		return nil, ErrTenantRequired
	}

	query := `
		SELECT
			o.id, o.tenant_id, o.attendance_id, o.user_id,
			COALESCE(u.full_name, u.email) AS user_name,
			d.name AS department_name,
			o.date,
			to_char(o.start_time, 'HH24:MI') AS start_time,
			to_char(o.end_time, 'HH24:MI') AS end_time,
			o.duration_minutes, o.reason, o.status,
			o.reviewed_by,
			COALESCE(rev.full_name, rev.email) AS reviewer_name,
			o.reviewed_at, o.rejection_reason,
			o.created_at, o.updated_at
		FROM overtime_requests o
		JOIN users u ON u.id = o.user_id AND u.tenant_id = o.tenant_id
		LEFT JOIN departments d ON d.id = u.department_id AND d.tenant_id = u.tenant_id
		LEFT JOIN users rev ON rev.id = o.reviewed_by AND rev.tenant_id = o.tenant_id
		WHERE o.tenant_id = $1 AND o.attendance_id = $2
		LIMIT 1;
	`

	var o OvertimeRequest
	var attID, deptName, revID, revName, rejReason sql.NullString
	var revAt sql.NullTime

	err := r.withTenantTx(ctx, tenantID, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, query, tenantID, attendanceID).Scan(
			&o.ID, &o.TenantID, &attID, &o.UserID,
			&o.UserName, &deptName,
			&o.Date, &o.StartTime, &o.EndTime,
			&o.DurationMinutes, &o.Reason, &o.Status,
			&revID, &revName, &revAt, &rejReason,
			&o.CreatedAt, &o.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrOvertimeNotFound
			}
			return fmt.Errorf("failed to query overtime_request by attendance_id: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if attID.Valid {
		o.AttendanceID = &attID.String
	}
	if deptName.Valid {
		o.DepartmentName = &deptName.String
	}
	if revID.Valid {
		o.ReviewedBy = &revID.String
	}
	if revName.Valid {
		o.ReviewerName = &revName.String
	}
	if revAt.Valid {
		o.ReviewedAt = &revAt.Time
	}
	if rejReason.Valid {
		o.RejectionReason = &rejReason.String
	}

	return &o, nil
}

func (r *postgresOvertimeRepository) List(ctx context.Context, tenantID string, f OvertimeFilter) ([]*OvertimeRequest, int, map[string]int, error) {
	if tenantID == "" {
		return nil, 0, nil, ErrTenantRequired
	}

	page := f.Page
	if page < 1 {
		page = 1
	}
	perPage := f.PerPage
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	countConds := []string{"o.tenant_id = $1"}
	countArgs := []interface{}{tenantID}
	argIdx := 2

	if f.DepartmentID != "" {
		countConds = append(countConds, fmt.Sprintf("u.department_id = $%d", argIdx))
		countArgs = append(countArgs, f.DepartmentID)
		argIdx++
	}
	if f.Date != nil {
		countConds = append(countConds, fmt.Sprintf("o.date = $%d", argIdx))
		countArgs = append(countArgs, f.Date.Format(time.DateOnly))
		argIdx++
	}
	if f.ManagerID != "" {
		countConds = append(countConds, fmt.Sprintf("u.manager_id = $%d", argIdx))
		countArgs = append(countArgs, f.ManagerID)
	}

	counts := map[string]int{
		OvertimeStatusPending:  0,
		OvertimeStatusApproved: 0,
		OvertimeStatusRejected: 0,
	}
	var results []*OvertimeRequest
	var total int

	err := r.withTenantTx(ctx, tenantID, func(tx *sql.Tx) error {
		countQuery := fmt.Sprintf(`
			SELECT o.status, COUNT(*)
			FROM overtime_requests o
			JOIN users u ON u.id = o.user_id AND u.tenant_id = o.tenant_id
			WHERE %s
			GROUP BY o.status;
		`, strings.Join(countConds, " AND "))

		rows, err := tx.QueryContext(ctx, countQuery, countArgs...)
		if err != nil {
			return fmt.Errorf("failed to count overtime by status: %w", err)
		}
		defer func() {
			_ = rows.Close()
		}()

		for rows.Next() {
			var st string
			var c int
			if err := rows.Scan(&st, &c); err == nil {
				counts[st] = c
			}
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("error iterating overtime counts: %w", err)
		}

		// 2. Query daftar dengan pagination
		filterConds := append([]string{}, countConds...)
		filterArgs := append([]interface{}{}, countArgs...)
		fIdx := len(filterArgs) + 1

		if f.Status != "" {
			filterConds = append(filterConds, fmt.Sprintf("o.status = $%d", fIdx))
			filterArgs = append(filterArgs, f.Status)
			fIdx++
		}

		total = counts[OvertimeStatusPending] + counts[OvertimeStatusApproved] + counts[OvertimeStatusRejected]
		if f.Status != "" {
			total = counts[f.Status]
		}

		listQuery := fmt.Sprintf(`
			SELECT
				o.id, o.tenant_id, o.attendance_id, o.user_id,
				COALESCE(u.full_name, u.email) AS user_name,
				d.name AS department_name,
				o.date,
				to_char(o.start_time, 'HH24:MI') AS start_time,
				to_char(o.end_time, 'HH24:MI') AS end_time,
				o.duration_minutes, o.reason, o.status,
				o.reviewed_by,
				COALESCE(rev.full_name, rev.email) AS reviewer_name,
				o.reviewed_at, o.rejection_reason,
				o.created_at, o.updated_at
			FROM overtime_requests o
			JOIN users u ON u.id = o.user_id AND u.tenant_id = o.tenant_id
			LEFT JOIN departments d ON d.id = u.department_id AND d.tenant_id = u.tenant_id
			LEFT JOIN users rev ON rev.id = o.reviewed_by AND rev.tenant_id = o.tenant_id
			WHERE %s
			ORDER BY o.date DESC, o.created_at DESC
			LIMIT $%d OFFSET $%d;
		`, strings.Join(filterConds, " AND "), fIdx, fIdx+1)

		filterArgs = append(filterArgs, perPage, offset)

		dataRows, err := tx.QueryContext(ctx, listQuery, filterArgs...)
		if err != nil {
			return fmt.Errorf("failed to query overtime list: %w", err)
		}
		defer func() {
			_ = dataRows.Close()
		}()

		results = make([]*OvertimeRequest, 0, perPage)
		for dataRows.Next() {
			var o OvertimeRequest
			var attID, deptName, revID, revName, rejReason sql.NullString
			var revAt sql.NullTime

			err := dataRows.Scan(
				&o.ID, &o.TenantID, &attID, &o.UserID,
				&o.UserName, &deptName,
				&o.Date, &o.StartTime, &o.EndTime,
				&o.DurationMinutes, &o.Reason, &o.Status,
				&revID, &revName, &revAt, &rejReason,
				&o.CreatedAt, &o.UpdatedAt,
			)
			if err != nil {
				return fmt.Errorf("failed to scan overtime row: %w", err)
			}

			if attID.Valid {
				o.AttendanceID = &attID.String
			}
			if deptName.Valid {
				o.DepartmentName = &deptName.String
			}
			if revID.Valid {
				o.ReviewedBy = &revID.String
			}
			if revName.Valid {
				o.ReviewerName = &revName.String
			}
			if revAt.Valid {
				o.ReviewedAt = &revAt.Time
			}
			if rejReason.Valid {
				o.RejectionReason = &rejReason.String
			}

			results = append(results, &o)
		}
		if err := dataRows.Err(); err != nil {
			return fmt.Errorf("error iterating overtime list: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, 0, nil, err
	}

	return results, total, counts, nil
}

func (r *postgresOvertimeRepository) Approve(ctx context.Context, tenantID, id, reviewerID string) (*OvertimeRequest, error) {
	if tenantID == "" {
		return nil, ErrTenantRequired
	}

	err := r.withTenantTx(ctx, tenantID, func(tx *sql.Tx) error {
		var currentStatus string
		var applicantID string

		checkQuery := `
			SELECT status, user_id FROM overtime_requests
			WHERE tenant_id = $1 AND id = $2
			FOR UPDATE;
		`
		err := tx.QueryRowContext(ctx, checkQuery, tenantID, id).Scan(&currentStatus, &applicantID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrOvertimeNotFound
			}
			return fmt.Errorf("failed to check overtime status: %w", err)
		}

		// Validasi: tidak boleh approve lembur sendiri
		if applicantID == reviewerID {
			return ErrOvertimeSelfApproval
		}

		// Validasi: pengajuan sudah diputuskan sebelumnya
		if currentStatus != OvertimeStatusPending {
			return ErrOvertimeAlreadyProcessed
		}

		now := time.Now().UTC()
		updateQuery := `
			UPDATE overtime_requests
			SET status = $1, reviewed_by = $2, reviewed_at = $3, updated_at = $3
			WHERE tenant_id = $4 AND id = $5;
		`
		_, err = tx.ExecContext(ctx, updateQuery, OvertimeStatusApproved, reviewerID, now, tenantID, id)
		if err != nil {
			return fmt.Errorf("failed to update overtime approve: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return r.FindByID(ctx, tenantID, id)
}

func (r *postgresOvertimeRepository) Reject(ctx context.Context, tenantID, id, reviewerID, reason string) (*OvertimeRequest, error) {
	if tenantID == "" {
		return nil, ErrTenantRequired
	}
	if strings.TrimSpace(reason) == "" {
		return nil, errors.New("alasan penolakan wajib diisi")
	}

	err := r.withTenantTx(ctx, tenantID, func(tx *sql.Tx) error {
		var currentStatus string
		var applicantID string

		checkQuery := `
			SELECT status, user_id FROM overtime_requests
			WHERE tenant_id = $1 AND id = $2
			FOR UPDATE;
		`
		err := tx.QueryRowContext(ctx, checkQuery, tenantID, id).Scan(&currentStatus, &applicantID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrOvertimeNotFound
			}
			return fmt.Errorf("failed to check overtime status: %w", err)
		}

		// Validasi: tidak boleh reject lembur sendiri
		if applicantID == reviewerID {
			return ErrOvertimeSelfApproval
		}

		// Validasi: pengajuan sudah diputuskan sebelumnya
		if currentStatus != OvertimeStatusPending {
			return ErrOvertimeAlreadyProcessed
		}

		now := time.Now().UTC()
		updateQuery := `
			UPDATE overtime_requests
			SET status = $1, reviewed_by = $2, reviewed_at = $3, rejection_reason = $4, updated_at = $3
			WHERE tenant_id = $5 AND id = $6;
		`
		_, err = tx.ExecContext(ctx, updateQuery, OvertimeStatusRejected, reviewerID, now, strings.TrimSpace(reason), tenantID, id)
		if err != nil {
			return fmt.Errorf("failed to update overtime reject: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return r.FindByID(ctx, tenantID, id)
}
