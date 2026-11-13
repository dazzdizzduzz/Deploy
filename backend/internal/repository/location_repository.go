package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type OfficeLocation struct {
	ID             string
	TenantID       string
	Name           string
	Address        string
	Latitude       float64
	Longitude      float64
	RadiusMeters   int
	SelfieRequired bool
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type LocationRepository interface {
	Create(ctx context.Context, loc *OfficeLocation) error
	Update(ctx context.Context, loc *OfficeLocation) error
	FindByID(ctx context.Context, tenantID, id string) (*OfficeLocation, error)
	FindAll(ctx context.Context, tenantID string) ([]*OfficeLocation, error)
	Delete(ctx context.Context, tenantID, id string) error
}

type postgresLocationRepository struct {
	db *sql.DB
}

func NewLocationRepository(db *sql.DB) LocationRepository {
	return &postgresLocationRepository{db: db}
}

func (r *postgresLocationRepository) withTenantTx(ctx context.Context, tenantID string, fn func(tx *sql.Tx) error) error {
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

func (r *postgresLocationRepository) Create(ctx context.Context, loc *OfficeLocation) error {
	return r.withTenantTx(ctx, loc.TenantID, func(tx *sql.Tx) error {
		query := `
			INSERT INTO office_locations (
				id, tenant_id, name, address, latitude, longitude, radius_meters, selfie_required, is_active, created_at, updated_at
			) VALUES (
				COALESCE(NULLIF($1, '')::uuid, uuid_generate_v4()),
				$2::uuid, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW()
			)
			RETURNING id, created_at, updated_at;
		`
		err := tx.QueryRowContext(ctx, query,
			loc.ID,
			loc.TenantID,
			loc.Name,
			loc.Address,
			loc.Latitude,
			loc.Longitude,
			loc.RadiusMeters,
			loc.SelfieRequired,
			loc.IsActive,
		).Scan(&loc.ID, &loc.CreatedAt, &loc.UpdatedAt)

		if err != nil {
			return fmt.Errorf("failed to insert office location: %w", err)
		}
		return nil
	})
}

func (r *postgresLocationRepository) Update(ctx context.Context, loc *OfficeLocation) error {
	return r.withTenantTx(ctx, loc.TenantID, func(tx *sql.Tx) error {
		query := `
			UPDATE office_locations
			SET name = $1,
			    address = $2,
			    latitude = $3,
			    longitude = $4,
			    radius_meters = $5,
			    selfie_required = $6,
			    is_active = $7,
			    updated_at = NOW()
			WHERE id = $8::uuid AND tenant_id = $9::uuid
			RETURNING updated_at;
		`
		err := tx.QueryRowContext(ctx, query,
			loc.Name,
			loc.Address,
			loc.Latitude,
			loc.Longitude,
			loc.RadiusMeters,
			loc.SelfieRequired,
			loc.IsActive,
			loc.ID,
			loc.TenantID,
		).Scan(&loc.UpdatedAt)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return sql.ErrNoRows
			}
			return fmt.Errorf("failed to update office location: %w", err)
		}
		return nil
	})
}

func (r *postgresLocationRepository) FindByID(ctx context.Context, tenantID, id string) (*OfficeLocation, error) {
	var loc OfficeLocation
	err := r.withTenantTx(ctx, tenantID, func(tx *sql.Tx) error {
		query := `
			SELECT id, tenant_id, name, address, latitude, longitude, radius_meters, selfie_required, is_active, created_at, updated_at
			FROM office_locations
			WHERE id = $1::uuid AND tenant_id = $2::uuid
			LIMIT 1;
		`
		row := tx.QueryRowContext(ctx, query, id, tenantID)
		err := row.Scan(
			&loc.ID,
			&loc.TenantID,
			&loc.Name,
			&loc.Address,
			&loc.Latitude,
			&loc.Longitude,
			&loc.RadiusMeters,
			&loc.SelfieRequired,
			&loc.IsActive,
			&loc.CreatedAt,
			&loc.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return sql.ErrNoRows
			}
			return fmt.Errorf("failed to scan office location: %w", err)
		}
		return nil
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &loc, nil
}

func (r *postgresLocationRepository) FindAll(ctx context.Context, tenantID string) ([]*OfficeLocation, error) {
	var locations []*OfficeLocation
	err := r.withTenantTx(ctx, tenantID, func(tx *sql.Tx) error {
		query := `
			SELECT id, tenant_id, name, address, latitude, longitude, radius_meters, selfie_required, is_active, created_at, updated_at
			FROM office_locations
			WHERE tenant_id = $1::uuid
			ORDER BY created_at ASC;
		`
		rows, err := tx.QueryContext(ctx, query, tenantID)
		if err != nil {
			return fmt.Errorf("failed to query office locations: %w", err)
		}
		defer func() {
			_ = rows.Close()
		}()

		for rows.Next() {
			var loc OfficeLocation
			if err := rows.Scan(
				&loc.ID,
				&loc.TenantID,
				&loc.Name,
				&loc.Address,
				&loc.Latitude,
				&loc.Longitude,
				&loc.RadiusMeters,
				&loc.SelfieRequired,
				&loc.IsActive,
				&loc.CreatedAt,
				&loc.UpdatedAt,
			); err != nil {
				return fmt.Errorf("failed to scan location row: %w", err)
			}
			locations = append(locations, &loc)
		}
		return rows.Err()
	})

	if err != nil {
		return nil, err
	}
	return locations, nil
}

func (r *postgresLocationRepository) Delete(ctx context.Context, tenantID, id string) error {
	return r.withTenantTx(ctx, tenantID, func(tx *sql.Tx) error {
		query := `
			DELETE FROM office_locations
			WHERE id = $1::uuid AND tenant_id = $2::uuid;
		`
		res, err := tx.ExecContext(ctx, query, id, tenantID)
		if err != nil {
			return fmt.Errorf("failed to delete office location: %w", err)
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

