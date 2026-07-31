package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type FirmwareVersionRepo struct {
	*DB
}

func NewFirmwareVersionRepo(db *DB) *FirmwareVersionRepo {
	return &FirmwareVersionRepo{DB: db}
}

func (r *FirmwareVersionRepo) ListFirmwareVersions(ctx context.Context) ([]domain.FirmwareVersion, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	query := `
	SELECT id, device_model, fw_version, fw_checksum, binary_url, created_at
	FROM firmware_versions
	ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(reqCtx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list firmware versions: %w", err)
	}
	defer rows.Close()

	var firmwareVersions []domain.FirmwareVersion
	for rows.Next() {
		var firmwareVersion domain.FirmwareVersion
		err = rows.Scan(
			&firmwareVersion.ID,
			&firmwareVersion.DeviceModel,
			&firmwareVersion.FWVersion,
			&firmwareVersion.FWChecksum,
			&firmwareVersion.BinaryUrl,
			&firmwareVersion.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan firmware version: %w", err)
		}

		firmwareVersions = append(firmwareVersions, firmwareVersion)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read firmware versions: %w", err)
	}

	return firmwareVersions, nil
}

func (r *FirmwareVersionRepo) GetFirmwareVersion(ctx context.Context, id uuid.UUID) (domain.FirmwareVersion, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	query := `
	SELECT id, device_model, fw_version, fw_checksum, binary_url, created_at 
	FROM firmware_versions 
	WHERE id = $1
	`

	row := r.pool.QueryRow(reqCtx, query, id)

	var firmwareVersion domain.FirmwareVersion
	err := row.Scan(
		&firmwareVersion.ID,
		&firmwareVersion.DeviceModel,
		&firmwareVersion.FWVersion,
		&firmwareVersion.FWChecksum,
		&firmwareVersion.BinaryUrl,
		&firmwareVersion.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return domain.FirmwareVersion{}, domain.ErrFirmwareVersionNotFound
	} else if err != nil {
		return domain.FirmwareVersion{}, fmt.Errorf("failed to get firmware version: %w", err)
	}

	return firmwareVersion, nil
}

func (r *FirmwareVersionRepo) CreateFirmwareVersion(ctx context.Context, firmwareVersion domain.FirmwareVersion) (domain.FirmwareVersion, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	query := `
	INSERT INTO firmware_versions (device_model, fw_version, fw_checksum, binary_url)
	VALUES ($1, $2, $3, $4)
	RETURNING id, device_model, fw_version, fw_checksum, binary_url, created_at
	`

	row := r.pool.QueryRow(reqCtx, query, firmwareVersion.DeviceModel, firmwareVersion.FWVersion, firmwareVersion.FWChecksum, firmwareVersion.BinaryUrl)

	var createdFirmwareVersion domain.FirmwareVersion
	err := row.Scan(
		&createdFirmwareVersion.ID,
		&createdFirmwareVersion.DeviceModel,
		&createdFirmwareVersion.FWVersion,
		&createdFirmwareVersion.FWChecksum,
		&createdFirmwareVersion.BinaryUrl,
		&createdFirmwareVersion.CreatedAt,
	)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
		return domain.FirmwareVersion{}, domain.ErrFirmwareVersionAlreadyExists
	} else if err != nil {
		return domain.FirmwareVersion{}, fmt.Errorf("failed to create firmware version: %w", err)
	}

	return createdFirmwareVersion, nil
}
