package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type WorkSchedule struct {
	ID                   string
	TenantID             string
	Name                 string
	StartTime            string // format "HH:MM", contoh "08:00"
	EndTime              string // format "HH:MM", contoh "17:00"
	ToleranceMinutes     int
	OvertimeAfterMinutes int
	MaxOvertimeMinutes   int
	WorkingDays          string
	AllowRemote          bool
	IsActive             bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type ScheduleRepository interface {
	Create(ctx context.Context, sched *WorkSchedule) error
	Update(ctx context.Context, sched *WorkSchedule) error
	FindByID(ctx context.Context, tenantID, id string) (*WorkSchedule, error)
	FindAll(ctx context.Context, tenantID string) ([]*WorkSchedule, error)
	Delete(ctx context.Context, tenantID, id string) error
}

type postgresScheduleRepository struct {
	db *sql.DB
}

func NewScheduleRepository(db *sql.DB) ScheduleRepository {
	return &postgresScheduleRepository{db: db}
}

func (r *postgresScheduleRepository) withTenantTx(ctx context.Context, tenantID string, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if tenantID != "" {
		if _, err := tx.ExecContext(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID); err != nil {
			return fmt.Errorf("failed to set app.current_tenant: %w", err)
		}
	}

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *postgresScheduleRepository) Create(ctx context.Context, sched *WorkSchedule) error {
	return r.withTenantTx(ctx, sched.TenantID, func(tx *sql.Tx) error {
		query := `
			INSERT INTO work_schedules (
				id, tenant_id, name, start_time, end_time, tolerance_minutes,
				overtime_after_minutes, max_overtime_minutes, working_days, allow_remote,
				is_active, created_at, updated_at
			) VALUES (
				COALESCE(NULLIF($1, '')::uuid, uuid_generate_v4()),
				$2::uuid, $3, $4::time, $5::time, $6, $7, $8, $9, $10, $11, NOW(), NOW()
			)
			RETURNING id, TO_CHAR(start_time, 'HH24:MI'), TO_CHAR(end_time, 'HH24:MI'), created_at, updated_at;
		`
		err := tx.QueryRowContext(ctx, query,
			sched.ID,
			sched.TenantID,
			sched.Name,
			sched.StartTime,
			sched.EndTime,
			sched.ToleranceMinutes,
			sched.OvertimeAfterMinutes,
			sched.MaxOvertimeMinutes,
			sched.WorkingDays,
			sched.AllowRemote,
			sched.IsActive,
		).Scan(&sched.ID, &sched.StartTime, &sched.EndTime, &sched.CreatedAt, &sched.UpdatedAt)

		if err != nil {
			return fmt.Errorf("failed to insert work schedule: %w", err)
		}
		return nil
	})
}

func (r *postgresScheduleRepository) Update(ctx context.Context, sched *WorkSchedule) error {
	return r.withTenantTx(ctx, sched.TenantID, func(tx *sql.Tx) error {
		query := `
			UPDATE work_schedules
			SET name = $1,
			    start_time = $2::time,
			    end_time = $3::time,
			    tolerance_minutes = $4,
			    overtime_after_minutes = $5,
			    max_overtime_minutes = $6,
			    working_days = $7,
			    allow_remote = $8,
			    is_active = $9,
			    updated_at = NOW()
			WHERE id = $10::uuid AND tenant_id = $11::uuid
			RETURNING TO_CHAR(start_time, 'HH24:MI'), TO_CHAR(end_time, 'HH24:MI'), updated_at;
		`
		err := tx.QueryRowContext(ctx, query,
			sched.Name,
			sched.StartTime,
			sched.EndTime,
			sched.ToleranceMinutes,
			sched.OvertimeAfterMinutes,
			sched.MaxOvertimeMinutes,
			sched.WorkingDays,
			sched.AllowRemote,
			sched.IsActive,
			sched.ID,
			sched.TenantID,
		).Scan(&sched.StartTime, &sched.EndTime, &sched.UpdatedAt)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return sql.ErrNoRows
			}
			return fmt.Errorf("failed to update work schedule: %w", err)
		}
		return nil
	})
}

func (r *postgresScheduleRepository) FindByID(ctx context.Context, tenantID, id string) (*WorkSchedule, error) {
	var sched WorkSchedule
	err := r.withTenantTx(ctx, tenantID, func(tx *sql.Tx) error {
		query := `
			SELECT id, tenant_id, name, TO_CHAR(start_time, 'HH24:MI'), TO_CHAR(end_time, 'HH24:MI'),
			       tolerance_minutes, overtime_after_minutes, max_overtime_minutes,
			       working_days, allow_remote, is_active, created_at, updated_at
			FROM work_schedules
			WHERE id = $1::uuid AND tenant_id = $2::uuid
			LIMIT 1;
		`
		row := tx.QueryRowContext(ctx, query, id, tenantID)
		err := row.Scan(
			&sched.ID,
			&sched.TenantID,
			&sched.Name,
			&sched.StartTime,
			&sched.EndTime,
			&sched.ToleranceMinutes,
			&sched.OvertimeAfterMinutes,
			&sched.MaxOvertimeMinutes,
			&sched.WorkingDays,
			&sched.AllowRemote,
			&sched.IsActive,
			&sched.CreatedAt,
			&sched.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return sql.ErrNoRows
			}
			return fmt.Errorf("failed to scan work schedule: %w", err)
		}
		return nil
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &sched, nil
}

func (r *postgresScheduleRepository) FindAll(ctx context.Context, tenantID string) ([]*WorkSchedule, error) {
	var schedules []*WorkSchedule
	err := r.withTenantTx(ctx, tenantID, func(tx *sql.Tx) error {
		query := `
			SELECT id, tenant_id, name, TO_CHAR(start_time, 'HH24:MI'), TO_CHAR(end_time, 'HH24:MI'),
			       tolerance_minutes, overtime_after_minutes, max_overtime_minutes,
			       working_days, allow_remote, is_active, created_at, updated_at
			FROM work_schedules
			WHERE tenant_id = $1::uuid
			ORDER BY created_at ASC;
		`
		rows, err := tx.QueryContext(ctx, query, tenantID)
		if err != nil {
			return fmt.Errorf("failed to query work schedules: %w", err)
		}
		defer func() {
			_ = rows.Close()
		}()

		for rows.Next() {
			var sched WorkSchedule
			if err := rows.Scan(
				&sched.ID,
				&sched.TenantID,
				&sched.Name,
				&sched.StartTime,
				&sched.EndTime,
				&sched.ToleranceMinutes,
				&sched.OvertimeAfterMinutes,
				&sched.MaxOvertimeMinutes,
				&sched.WorkingDays,
				&sched.AllowRemote,
				&sched.IsActive,
				&sched.CreatedAt,
				&sched.UpdatedAt,
			); err != nil {
				return fmt.Errorf("failed to scan work schedule row: %w", err)
			}
			schedules = append(schedules, &sched)
		}
		return rows.Err()
	})

	if err != nil {
		return nil, err
	}
	return schedules, nil
}

func (r *postgresScheduleRepository) Delete(ctx context.Context, tenantID, id string) error {
	return r.withTenantTx(ctx, tenantID, func(tx *sql.Tx) error {
		query := `
			DELETE FROM work_schedules
			WHERE id = $1::uuid AND tenant_id = $2::uuid;
		`
		res, err := tx.ExecContext(ctx, query, id, tenantID)
		if err != nil {
			return fmt.Errorf("failed to delete work schedule: %w", err)
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to check affected rows: %w", err)
		}
		if rows == 0 {
			return sql.ErrNoRows
		}
		return nil
	})
}

