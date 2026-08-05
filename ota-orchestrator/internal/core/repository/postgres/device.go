package postgres

import (
	"context"
	"fmt"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type DeviceRepo struct {
	*DB
}

func NewDeviceRepo(db *DB) *DeviceRepo {
	return &DeviceRepo{DB: db}
}

func (r *DeviceRepo) ListDevices(ctx context.Context) ([]domain.Device, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	query := `
	SELECT id, device_model, current_version, status, last_seen, created_at
	FROM devices
	ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(reqCtx, query)

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

func (r *DeviceRepo) GetDevice(ctx context.Context, id uuid.UUID) (domain.Device, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	query := `
	SELECT id, device_model, current_version, status, last_seen, created_at FROM devices
	WHERE id = $1
	`

	row := r.pool.QueryRow(reqCtx, query, id)

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

func (r *DeviceRepo) CreateDevice(ctx context.Context, device domain.Device) (domain.Device, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	query := `
	INSERT INTO devices (device_model, current_version)
	VALUES ($1, $2)
	RETURNING id, device_model, current_version, status, last_seen, created_at
	`

	row := r.pool.QueryRow(reqCtx, query, device.DeviceModel, device.CurrentVersion)

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

func (r *DeviceRepo) UpdateDeviceCheckinInfo(ctx context.Context, id uuid.UUID, version string) (domain.Device, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	query := `
	UPDATE devices SET current_version = $1, last_seen = now()
	WHERE id = $2
	RETURNING id, device_model, current_version, status, last_seen, created_at
	`

	row := r.pool.QueryRow(reqCtx, query, version, id)

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
		return domain.Device{}, fmt.Errorf("failed to create device: %w", err)
	}

	return device, nil
}

func (r *DeviceRepo) DecommissionDevice(ctx context.Context, id uuid.UUID) (domain.Device, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	query := `
	UPDATE devices SET status = 'decommissioned'
	WHERE id = $1
	RETURNING id, device_model, current_version, status, last_seen, created_at
	`

	row := r.pool.QueryRow(reqCtx, query, id)

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
