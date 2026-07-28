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

type RolloutCampaignRepo struct {
	*DB
}

func NewRolloutCampaignRepo(db *DB) *RolloutCampaignRepo {
	return &RolloutCampaignRepo{DB: db}
}

func (r *RolloutCampaignRepo) ListRolloutCampaigns(ctx context.Context) ([]domain.RolloutCampaign, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	query := `
	SELECT id, firmware_version_id, device_model, status, created_at, started_at, completed_at
	FROM rollout_campaigns
	ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(reqCtx, query)

	if err != nil {
		return nil, fmt.Errorf("failed to list rollout campaigns: %w", err)
	}
	defer rows.Close()

	var rolloutCampaigns []domain.RolloutCampaign
	for rows.Next() {
		var rolloutCampaign domain.RolloutCampaign
		err = rows.Scan(
			&rolloutCampaign.ID,
			&rolloutCampaign.FirmwareVersionID,
			&rolloutCampaign.DeviceModel,
			&rolloutCampaign.Status,
			&rolloutCampaign.CreatedAt,
			&rolloutCampaign.StartedAt,
			&rolloutCampaign.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rollout campaign: %w", err)
		}

		rolloutCampaigns = append(rolloutCampaigns, rolloutCampaign)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read rollout campaigns: %w", err)
	}

	return rolloutCampaigns, nil
}

func (r *RolloutCampaignRepo) GetRolloutCampaign(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	query := `
	SELECT id, firmware_version_id, device_model, status, created_at, started_at, completed_at
	FROM rollout_campaigns
	WHERE id = $1
	`

	row := r.pool.QueryRow(reqCtx, query, id)

	var campaign domain.RolloutCampaign
	err := row.Scan(
		&campaign.ID,
		&campaign.FirmwareVersionID,
		&campaign.DeviceModel,
		&campaign.Status,
		&campaign.CreatedAt,
		&campaign.StartedAt,
		&campaign.CompletedAt,
	)

	if err == pgx.ErrNoRows {
		return domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound
	} else if err != nil {
		return domain.RolloutCampaign{}, fmt.Errorf("failed to get rollout campaign: %w", err)
	}

	stages, err := r.listRolloutStagesByCampaignID(reqCtx, campaign.ID)
	if err != nil {
		return domain.RolloutCampaign{}, fmt.Errorf("failed to get stages for campaign: %w", err)
	}

	campaign.RolloutStages = stages
	return campaign, nil
}

func (r *RolloutCampaignRepo) CreateRolloutCampaign(ctx context.Context, campaign domain.RolloutCampaign) (domain.RolloutCampaign, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	tx, err := r.pool.BeginTx(reqCtx, pgx.TxOptions{})
	if err != nil {
		return domain.RolloutCampaign{}, fmt.Errorf("failed to start transaction for campaign creation: %w", err)
	}
	defer tx.Rollback(reqCtx)

	campaignQuery := `
	INSERT INTO rollout_campaigns (firmware_version_id, device_model)
	VALUES ($1, $2)
	RETURNING id, firmware_version_id, device_model, status, created_at, started_at, completed_at
	`

	campaignRow := tx.QueryRow(reqCtx, campaignQuery, campaign.FirmwareVersionID, campaign.DeviceModel)

	var createdCampaign domain.RolloutCampaign
	err = campaignRow.Scan(
		&createdCampaign.ID,
		&createdCampaign.FirmwareVersionID,
		&createdCampaign.DeviceModel,
		&createdCampaign.Status,
		&createdCampaign.CreatedAt,
		&createdCampaign.StartedAt,
		&createdCampaign.CompletedAt,
	)
	if err != nil {
		return domain.RolloutCampaign{}, fmt.Errorf("failed to create rollout campaign: %w", err)
	}

	stagesQuery := `
	INSERT INTO rollout_stages (campaign_id, order_index, target_percent, min_sample_size, success_threshold)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id, campaign_id, order_index, target_percent, min_sample_size, success_threshold, status, entered_at
	`

	createdCampaign.RolloutStages = make([]domain.RolloutStage, 0, len(campaign.RolloutStages))

	for _, stage := range campaign.RolloutStages {
		row := tx.QueryRow(reqCtx, stagesQuery, createdCampaign.ID, stage.OrderIndex, stage.TargetPercent, stage.MinSampleSize, stage.SuccessThreshold)

		var createdStage domain.RolloutStage
		err = row.Scan(
			&createdStage.ID,
			&createdStage.CampaignID,
			&createdStage.OrderIndex,
			&createdStage.TargetPercent,
			&createdStage.MinSampleSize,
			&createdStage.SuccessThreshold,
			&createdStage.Status,
			&createdStage.EnteredAt,
		)
		if err != nil {
			return domain.RolloutCampaign{}, fmt.Errorf("failed to create rollout stage during campaign creation: %w", err)
		}

		createdCampaign.RolloutStages = append(createdCampaign.RolloutStages, createdStage)
	}

	err = tx.Commit(reqCtx)
	if err != nil {
		return domain.RolloutCampaign{}, fmt.Errorf("failed to commit campaign and stages creation: %w", err)
	}

	return createdCampaign, nil
}

func (r *RolloutCampaignRepo) StartRolloutCampaign(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	tx, err := r.pool.BeginTx(reqCtx, pgx.TxOptions{})
	if err != nil {
		return domain.RolloutCampaign{}, fmt.Errorf("failed to start transaction for campaign creation: %w", err)
	}
	defer tx.Rollback(reqCtx)

	campaignQuery := `
	UPDATE rollout_campaigns SET status = 'running', started_at = now()
	WHERE id = $1 AND status = 'draft'
	RETURNING id, firmware_version_id, device_model, status, created_at, started_at, completed_at
	`

	row := tx.QueryRow(reqCtx, campaignQuery, id)

	var campaign domain.RolloutCampaign
	err = row.Scan(
		&campaign.ID,
		&campaign.FirmwareVersionID,
		&campaign.DeviceModel,
		&campaign.Status,
		&campaign.CreatedAt,
		&campaign.StartedAt,
		&campaign.CompletedAt,
	)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
		return domain.RolloutCampaign{}, domain.ErrCampaignAlreadyRunning
	} else if err == pgx.ErrNoRows {
		return domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound
	} else if err != nil {
		return domain.RolloutCampaign{}, fmt.Errorf("failed to start rollout campaign: %w", err)
	}

	stageQuery := `
	WITH target AS (
		SELECT id
		FROM rollout_stages
		WHERE campaign_id = $1
		ORDER BY order_index
		LIMIT 1
		FOR UPDATE
	)
		
	UPDATE rollout_stages
	SET status = 'active', entered_at = now()
	FROM target
	WHERE rollout_stages.id = target.id
	`

	_, err = tx.Exec(reqCtx, stageQuery, id)
	if err != nil {
		return domain.RolloutCampaign{}, fmt.Errorf("failed to update stage: %w", err)
	}

	err = tx.Commit(reqCtx)
	if err != nil {
		return domain.RolloutCampaign{}, fmt.Errorf("failed to commit campaign start: %w", err)
	}

	stages, err := r.listRolloutStagesByCampaignID(reqCtx, campaign.ID)
	if err != nil {
		return domain.RolloutCampaign{}, fmt.Errorf("failed to get stages for campaign: %w", err)
	}

	campaign.RolloutStages = stages
	return campaign, nil
}

func (r *RolloutCampaignRepo) PauseRolloutCampaign(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	query := `
	UPDATE rollout_campaigns SET status = 'paused'
	WHERE id = $1 AND status = 'running'
	RETURNING id, firmware_version_id, device_model, status, created_at, started_at, completed_at
	`

	row := r.pool.QueryRow(reqCtx, query, id)

	var campaign domain.RolloutCampaign
	err := row.Scan(
		&campaign.ID,
		&campaign.FirmwareVersionID,
		&campaign.DeviceModel,
		&campaign.Status,
		&campaign.CreatedAt,
		&campaign.StartedAt,
		&campaign.CompletedAt,
	)

	if err == pgx.ErrNoRows {
		return domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound
	} else if err != nil {
		return domain.RolloutCampaign{}, fmt.Errorf("failed to pause rollout campaign: %w", err)
	}

	stages, err := r.listRolloutStagesByCampaignID(reqCtx, campaign.ID)
	if err != nil {
		return domain.RolloutCampaign{}, fmt.Errorf("failed to get stages for campaign: %w", err)
	}

	campaign.RolloutStages = stages
	return campaign, nil
}

func (r *RolloutCampaignRepo) ResumeRolloutCampaign(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	query := `
	UPDATE rollout_campaigns SET status = 'running'
	WHERE id = $1 AND status = 'paused'
	RETURNING id, firmware_version_id, device_model, status, created_at, started_at, completed_at
	`

	row := r.pool.QueryRow(reqCtx, query, id)

	var campaign domain.RolloutCampaign
	err := row.Scan(
		&campaign.ID,
		&campaign.FirmwareVersionID,
		&campaign.DeviceModel,
		&campaign.Status,
		&campaign.CreatedAt,
		&campaign.StartedAt,
		&campaign.CompletedAt,
	)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
		return domain.RolloutCampaign{}, domain.ErrCampaignAlreadyRunning
	} else if err == pgx.ErrNoRows {
		return domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound
	} else if err != nil {
		return domain.RolloutCampaign{}, fmt.Errorf("failed to resume rollout campaign: %w", err)
	}

	stages, err := r.listRolloutStagesByCampaignID(reqCtx, campaign.ID)
	if err != nil {
		return domain.RolloutCampaign{}, fmt.Errorf("failed to get stages for campaign: %w", err)
	}

	campaign.RolloutStages = stages
	return campaign, nil
}

func (r *RolloutCampaignRepo) listRolloutStagesByCampaignID(ctx context.Context, campaignID uuid.UUID) ([]domain.RolloutStage, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	query := `
	SELECT id, campaign_id, order_index, target_percent, min_sample_size, success_threshold, status, entered_at
	FROM rollout_stages
	WHERE campaign_id = $1
	ORDER BY order_index
	`

	rows, err := r.pool.Query(reqCtx, query, campaignID)
	if err != nil {
		return nil, fmt.Errorf("failed to list rollout stages: %w", err)
	}
	defer rows.Close()

	var stages []domain.RolloutStage
	for rows.Next() {
		var stage domain.RolloutStage
		err := rows.Scan(
			&stage.ID,
			&stage.CampaignID,
			&stage.OrderIndex,
			&stage.TargetPercent,
			&stage.MinSampleSize,
			&stage.SuccessThreshold,
			&stage.Status,
			&stage.EnteredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rollout stage: %w", err)
		}

		stages = append(stages, stage)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read rollout stages: %w", err)
	}

	return stages, nil
}
