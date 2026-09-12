package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type DeviceRepo struct {
	*DB
}

func NewDeviceRepo(db *DB) *DeviceRepo {
	return &DeviceRepo{DB: db}
}

func (r *DeviceRepo) List(ctx context.Context, filters domain.DeviceFilters, pagination domain.Pagination) ([]domain.Device, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	exec := r.exec(reqCtx)

	var query string
	var args []any

	if (filters == domain.DeviceFilters{}) {
		query = `
		SELECT id, device_model, current_version, status, last_seen, created_at
		FROM devices
		ORDER BY created_at DESC, id DESC
		`
	} else {
		query, args = r.buildListQueryWithFilters(filters)
	}

	query, args = r.addPagination(query, args, pagination)
	rows, err := exec.Query(reqCtx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}
	defer rows.Close()

	var devices []domain.Device
	for rows.Next() {
		var device domain.Device
		err := rows.Scan(
			&device.ID,
			&device.DeviceModel,
			&device.CurrentVersion,
			&device.Status,
			&device.LastSeen,
			&device.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan device: %w", err)
		}
		devices = append(devices, device)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read devices: %w", err)
	}

	return devices, nil
}

func (r *DeviceRepo) buildListQueryWithFilters(filters domain.DeviceFilters) (query string, args []any) {
	var querySb strings.Builder
	querySb.WriteString(`
	SELECT id, device_model, current_version, status, last_seen, created_at
	FROM devices
	WHERE
	`)

	addAnd := func() {
		if len(args) > 1 {
			querySb.WriteString(" AND ")
		}
	}

	if filters.DeviceModel != "" {
		args = append(args, filters.DeviceModel)
		addAnd()
		fmt.Fprintf(&querySb, "device_model = $%d", len(args))
	}
	if filters.Status != "" {
		args = append(args, filters.Status)
		addAnd()
		fmt.Fprintf(&querySb, "status = $%d", len(args))
	}
	querySb.WriteString("\nORDER BY created_at DESC, id DESC\n")

	return querySb.String(), args
}

func (r *DeviceRepo) Get(ctx context.Context, id uuid.UUID) (domain.Device, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	exec := r.exec(reqCtx)

	query := `
	SELECT id, device_model, current_version, status, last_seen, created_at
	FROM devices
	WHERE id = $1
	`

	row := exec.QueryRow(reqCtx, query, id)

	var device domain.Device
	err := row.Scan(
		&device.ID,
		&device.DeviceModel,
		&device.CurrentVersion,
		&device.Status,
		&device.LastSeen,
		&device.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return domain.Device{}, domain.ErrDeviceNotFound
	} else if err != nil {
		return domain.Device{}, fmt.Errorf("failed to find device: %w", err)
	}

	return device, nil
}

func (r *DeviceRepo) Create(ctx context.Context, device domain.Device) (domain.Device, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	exec := r.exec(reqCtx)

	query := `
	INSERT INTO devices (device_model, current_version)
	VALUES ($1, $2)
	RETURNING id, device_model, current_version, status, last_seen, created_at
	`

	row := exec.QueryRow(reqCtx, query, device.DeviceModel, device.CurrentVersion)

	var createdDevice domain.Device
	err := row.Scan(
		&createdDevice.ID,
		&createdDevice.DeviceModel,
		&createdDevice.CurrentVersion,
		&createdDevice.Status,
		&createdDevice.LastSeen,
		&createdDevice.CreatedAt,
	)
	if err != nil {
		return domain.Device{}, fmt.Errorf("failed to create device: %w", err)
	}

	return createdDevice, nil
}

func (r *DeviceRepo) Decommission(ctx context.Context, id uuid.UUID) (domain.Device, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	exec := r.exec(reqCtx)

	query := `
	UPDATE devices SET status = 'decommissioned'
	WHERE id = $1
	RETURNING id, device_model, current_version, status, last_seen, created_at
	`

	row := exec.QueryRow(reqCtx, query, id)

	var device domain.Device
	err := row.Scan(
		&device.ID,
		&device.DeviceModel,
		&device.CurrentVersion,
		&device.Status,
		&device.LastSeen,
		&device.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return domain.Device{}, domain.ErrDeviceNotFound
	} else if err != nil {
		return domain.Device{}, fmt.Errorf("failed to decommission device: %w", err)
	}

	return device, nil
}
