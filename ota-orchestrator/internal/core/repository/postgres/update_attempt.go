package postgres

import (
	"context"
	"fmt"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/domain"
)

type UpdateAttemptRepo struct {
	*DB
}

func NewUpdateAttemptRepo(db *DB) *UpdateAttemptRepo {
	return &UpdateAttemptRepo{DB: db}
}

func (r *UpdateAttemptRepo) CreateUpdateAttempt(ctx context.Context, updateAttempt domain.UpdateAttempt) (domain.UpdateAttempt, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	query := `
	INSERT INTO update_attempts (device_id, campaign_id, stage_id, result)
	VALUES ($1, $2, $3, $4)
	RETURNING id, device_id, campaign_id, stage_id, result, reported_at
	`

	row := r.pool.QueryRow(reqCtx, query, updateAttempt.DeviceID, updateAttempt.CampaignID, updateAttempt.StageID, updateAttempt.Result)

	var createdUpdateAttempt domain.UpdateAttempt
	err := row.Scan(
		&createdUpdateAttempt.ID,
		&createdUpdateAttempt.DeviceID,
		&createdUpdateAttempt.CampaignID,
		&createdUpdateAttempt.StageID,
		&createdUpdateAttempt.Result,
		&createdUpdateAttempt.ReportedAt,
	)

	if err != nil {
		return domain.UpdateAttempt{}, fmt.Errorf("failed to create update attempt: %w", err)
	}

	return createdUpdateAttempt, nil
}
