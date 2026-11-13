package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

var (
	// ErrEmailTaken: email sudah dipakai karyawan lain di tenant yang sama (409).
	ErrEmailTaken = errors.New("email sudah terdaftar pada tenant ini")
	// ErrEmployeeNotFound: karyawan tidak ada di tenant ini (404, termasuk milik tenant lain).
	ErrEmployeeNotFound = errors.New("karyawan tidak ditemukan")
)

// Status karyawan sesuai kontrak (UserResponse.status).
const (
	EmployeeStatusActive   = "aktif"
	EmployeeStatusInactive = "nonaktif"
	EmployeeStatusInvited  = "diundang"
)

// Employee adalah satu baris users beserta nama relasinya (departemen, jabatan, atasan,
// jadwal, lokasi) untuk respons API Karyawan (FR-03).
type Employee struct {
	ID             string
	TenantID       string
	FullName       string
	Email          string
	Phone          *string
	DepartmentID   *string
	DepartmentName *string
	PositionID     *string
	PositionName   *string
	Role           string
	EmploymentType string
	Status         string
	JoinedAt       *time.Time
	ManagerID      *string
	ManagerName    *string
	ScheduleID     *string
	ScheduleName   *string
	LocationID     *string
	LocationName   *string
	AllowWFH       bool

	// Data sensitif. Repository selalu mengisi; handler yang menyaring per peran.
	BaseSalary        *string // string desimal "7250000.00"
	BankName          *string
	BankAccountNumber *string
	BankAccountHolder *string

	InactiveSince      *time.Time
	DeactivationReason *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// EmployeeFilter adalah filter GET /users.
type EmployeeFilter struct {
	Status       string
	DepartmentID string
	Role         string
	Search       string
	Page         int
	PerPage      int
}

// EmployeeRefs adalah id relasi yang harus milik tenant yang sama.
type EmployeeRefs struct {
	DepartmentID *string
	PositionID   *string
	ManagerID    *string
	ScheduleID   *string
	LocationID   *string
}

type EmployeeRepository interface {
	List(ctx context.Context, tenantID string, f EmployeeFilter) ([]*Employee, int, error)
	FindByID(ctx context.Context, tenantID, id string) (*Employee, error)
	Create(ctx context.Context, e *Employee, passwordHash string) error
	Update(ctx context.Context, e *Employee) error
	// MissingRefs mengembalikan nama field yang id-nya tidak ditemukan di tenant ini.
	MissingRefs(ctx context.Context, tenantID string, refs EmployeeRefs) ([]string, error)
}

type postgresEmployeeRepository struct {
	db *sql.DB
}

func NewEmployeeRepository(db *sql.DB) EmployeeRepository {
	return &postgresEmployeeRepository{db: db}
}

// withTenantTx sama dengan versi di attendance_repository: menyetel app.current_tenant
// agar RLS aktif, dan menolak tenant kosong.
func (r *postgresEmployeeRepository) withTenantTx(ctx context.Context, tenantID string, fn func(tx *sql.Tx) error) error {
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

// employeeSelect mengambil users beserta nama relasi. Setiap JOIN juga mencocokkan
// tenant_id agar relasi milik tenant lain tidak pernah ikut terbaca.
const employeeSelect = `
	SELECT u.id, u.tenant_id, u.full_name, u.email, u.phone,
	       u.department_id, d.name, u.position_id, p.name,
	       u.role, u.employment_type, u.status, u.joined_at,
	       u.manager_id, m.full_name, u.work_schedule_id, ws.name, u.office_location_id, ol.name,
	       u.allow_wfh_without_approval,
	       u.base_salary::text, u.bank_name, u.bank_account_number, u.bank_account_holder,
	       u.inactive_since, u.deactivation_reason, u.created_at, u.updated_at
	FROM users u
	LEFT JOIN departments d       ON d.id = u.department_id      AND d.tenant_id = u.tenant_id
	LEFT JOIN positions p         ON p.id = u.position_id        AND p.tenant_id = u.tenant_id
	LEFT JOIN users m             ON m.id = u.manager_id         AND m.tenant_id = u.tenant_id
	LEFT JOIN work_schedules ws   ON ws.id = u.work_schedule_id  AND ws.tenant_id = u.tenant_id
	LEFT JOIN office_locations ol ON ol.id = u.office_location_id AND ol.tenant_id = u.tenant_id`

func scanEmployee(row rowScanner) (*Employee, error) {
	var (
		e                                             Employee
		phone, deptID, deptName, posID, posName       sql.NullString
		mgrID, mgrName, schedID, schedName, locID     sql.NullString
		locName, salary, bankName, bankNo, bankHolder sql.NullString
		reason                                        sql.NullString
		joinedAt, inactiveSince                       sql.NullTime
	)
	err := row.Scan(
		&e.ID, &e.TenantID, &e.FullName, &e.Email, &phone,
		&deptID, &deptName, &posID, &posName,
		&e.Role, &e.EmploymentType, &e.Status, &joinedAt,
		&mgrID, &mgrName, &schedID, &schedName, &locID, &locName,
		&e.AllowWFH,
		&salary, &bankName, &bankNo, &bankHolder,
		&inactiveSince, &reason, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	e.Phone = nullStringPtr(phone)
	e.DepartmentID, e.DepartmentName = nullStringPtr(deptID), nullStringPtr(deptName)
	e.PositionID, e.PositionName = nullStringPtr(posID), nullStringPtr(posName)
	e.ManagerID, e.ManagerName = nullStringPtr(mgrID), nullStringPtr(mgrName)
	e.ScheduleID, e.ScheduleName = nullStringPtr(schedID), nullStringPtr(schedName)
	e.LocationID, e.LocationName = nullStringPtr(locID), nullStringPtr(locName)
	e.BaseSalary = nullStringPtr(salary)
	e.BankName, e.BankAccountNumber, e.BankAccountHolder = nullStringPtr(bankName), nullStringPtr(bankNo), nullStringPtr(bankHolder)
	e.DeactivationReason = nullStringPtr(reason)
	if joinedAt.Valid {
		e.JoinedAt = &joinedAt.Time
	}
	if inactiveSince.Valid {
		e.InactiveSince = &inactiveSince.Time
	}
	return &e, nil
}

// List mengambil daftar karyawan dengan filter dan pagination.
func (r *postgresEmployeeRepository) List(ctx context.Context, tenantID string, f EmployeeFilter) ([]*Employee, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 {
		f.PerPage = 20
	}

	// Parameter kosong berarti filter tidak dipakai. Wildcard LIKE pada kata kunci di-escape
	// agar input seperti "%" tidak berubah menjadi "cocok dengan semua".
	where := `
		WHERE u.tenant_id = $1
		  AND ($2 = '' OR u.status = $2)
		  AND ($3 = '' OR u.department_id::text = $3)
		  AND ($4 = '' OR u.role = $4)
		  AND ($5 = '' OR u.full_name ILIKE '%' || $5 || '%' ESCAPE '\' OR u.email ILIKE '%' || $5 || '%' ESCAPE '\')`
	args := []any{tenantID, f.Status, f.DepartmentID, f.Role, escapeLike(strings.TrimSpace(f.Search))}

	var (
		list  []*Employee
		total int
	)
	err := r.withTenantTx(ctx, tenantID, func(tx *sql.Tx) error {
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users u`+where, args...).Scan(&total); err != nil {
			return fmt.Errorf("failed to count employees: %w", err)
		}

		query := employeeSelect + where + ` ORDER BY u.full_name ASC, u.id ASC LIMIT $6 OFFSET $7`
		rows, err := tx.QueryContext(ctx, query, append(args, f.PerPage, (f.Page-1)*f.PerPage)...)
		if err != nil {
			return fmt.Errorf("failed to query employees: %w", err)
		}
		defer func() {
			_ = rows.Close()
		}()
		for rows.Next() {
			e, err := scanEmployee(rows)
			if err != nil {
				return fmt.Errorf("failed to scan employee: %w", err)
			}
			list = append(list, e)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// FindByID mengembalikan (nil, nil) bila karyawan tidak ada di tenant ini.
func (r *postgresEmployeeRepository) FindByID(ctx context.Context, tenantID, id string) (*Employee, error) {
	var found *Employee
	err := r.withTenantTx(ctx, tenantID, func(tx *sql.Tx) error {
		e, err := scanEmployee(tx.QueryRowContext(ctx, employeeSelect+` WHERE u.tenant_id = $1 AND u.id = $2`, tenantID, id))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return fmt.Errorf("failed to query employee: %w", err)
		}
		found = e
		return nil
	})
	return found, err
}

// Create menyimpan karyawan baru lalu membaca ulang barisnya (nama relasi ikut terisi).
func (r *postgresEmployeeRepository) Create(ctx context.Context, e *Employee, passwordHash string) error {
	return r.withTenantTx(ctx, e.TenantID, func(tx *sql.Tx) error {
		var id string
		err := tx.QueryRowContext(ctx, `
			INSERT INTO users (
				tenant_id, full_name, email, password_hash, phone, department_id, position_id,
				role, employment_type, status, is_active, joined_at, manager_id,
				work_schedule_id, office_location_id, allow_wfh_without_approval,
				base_salary, bank_name, bank_account_number, bank_account_holder
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17::numeric,$18,$19,$20)
			RETURNING id`,
			e.TenantID, e.FullName, e.Email, passwordHash, e.Phone, e.DepartmentID, e.PositionID,
			e.Role, e.EmploymentType, e.Status, e.Status == EmployeeStatusActive, dateParam(e.JoinedAt), e.ManagerID,
			e.ScheduleID, e.LocationID, e.AllowWFH,
			e.BaseSalary, e.BankName, e.BankAccountNumber, e.BankAccountHolder,
		).Scan(&id)
		if err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == pgUniqueViolation {
				return ErrEmailTaken
			}
			return fmt.Errorf("failed to insert employee: %w", err)
		}
		saved, err := scanEmployee(tx.QueryRowContext(ctx, employeeSelect+` WHERE u.tenant_id = $1 AND u.id = $2`, e.TenantID, id))
		if err != nil {
			return fmt.Errorf("failed to reload employee: %w", err)
		}
		*e = *saved
		return nil
	})
}

// Update menulis ulang seluruh kolom yang boleh diubah lewat API (PATCH, deactivate,
// reactivate). Email dan password tidak diubah di sini.
func (r *postgresEmployeeRepository) Update(ctx context.Context, e *Employee) error {
	return r.withTenantTx(ctx, e.TenantID, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE users SET
				full_name = $3, phone = $4, department_id = $5, position_id = $6,
				role = $7, employment_type = $8, status = $9, is_active = $10,
				joined_at = $11, manager_id = $12, work_schedule_id = $13, office_location_id = $14,
				allow_wfh_without_approval = $15, base_salary = $16::numeric,
				bank_name = $17, bank_account_number = $18, bank_account_holder = $19,
				inactive_since = $20, deactivation_reason = $21, updated_at = NOW()
			WHERE tenant_id = $1 AND id = $2`,
			e.TenantID, e.ID,
			e.FullName, e.Phone, e.DepartmentID, e.PositionID,
			e.Role, e.EmploymentType, e.Status, e.Status == EmployeeStatusActive,
			dateParam(e.JoinedAt), e.ManagerID, e.ScheduleID, e.LocationID,
			e.AllowWFH, e.BaseSalary,
			e.BankName, e.BankAccountNumber, e.BankAccountHolder,
			dateParam(e.InactiveSince), e.DeactivationReason,
		)
		if err != nil {
			return fmt.Errorf("failed to update employee: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrEmployeeNotFound
		}
		saved, err := scanEmployee(tx.QueryRowContext(ctx, employeeSelect+` WHERE u.tenant_id = $1 AND u.id = $2`, e.TenantID, e.ID))
		if err != nil {
			return fmt.Errorf("failed to reload employee: %w", err)
		}
		*e = *saved
		return nil
	})
}

// MissingRefs memeriksa setiap id relasi ada di tenant ini. Tanpa pemeriksaan ini,
// foreign key hanya memastikan id ada di tabel, bukan di tenant yang sama.
func (r *postgresEmployeeRepository) MissingRefs(ctx context.Context, tenantID string, refs EmployeeRefs) ([]string, error) {
	checks := []struct {
		field string
		table string
		id    *string
	}{
		{"department_id", "departments", refs.DepartmentID},
		{"position_id", "positions", refs.PositionID},
		{"manager_id", "users", refs.ManagerID},
		{"schedule_id", "work_schedules", refs.ScheduleID},
		{"location_id", "office_locations", refs.LocationID},
	}

	var missing []string
	err := r.withTenantTx(ctx, tenantID, func(tx *sql.Tx) error {
		for _, c := range checks {
			if c.id == nil {
				continue
			}
			// Nama tabel berasal dari daftar tetap di atas, bukan dari input pengguna.
			var exists bool
			q := `SELECT EXISTS (SELECT 1 FROM ` + c.table + ` WHERE tenant_id = $1 AND id::text = $2)`
			if err := tx.QueryRowContext(ctx, q, tenantID, *c.id).Scan(&exists); err != nil {
				return fmt.Errorf("failed to check %s: %w", c.field, err)
			}
			if !exists {
				missing = append(missing, c.field)
			}
		}
		return nil
	})
	return missing, err
}

// escapeLike meng-escape karakter khusus pola LIKE (\, %, _).
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// dateParam mengirim tanggal sebagai string YYYY-MM-DD agar tidak tergeser zona waktu.
func dateParam(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.DateOnly)
}
