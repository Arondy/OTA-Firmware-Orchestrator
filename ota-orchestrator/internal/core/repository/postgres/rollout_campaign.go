package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
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

func (r *RolloutCampaignRepo) List(ctx context.Context) ([]domain.RolloutCampaign, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	exec := r.exec(reqCtx)

	query := `
	SELECT id, firmware_version_id, device_model, status, created_at, started_at, completed_at
	FROM rollout_campaigns
	ORDER BY created_at DESC
	`

	rows, err := exec.Query(reqCtx, query)
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

func (r *RolloutCampaignRepo) Get(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	exec := r.exec(reqCtx)

	query := `
	SELECT id, firmware_version_id, device_model, status, created_at, started_at, completed_at
	FROM rollout_campaigns
	WHERE id = $1
	`

	row := exec.QueryRow(reqCtx, query, id)

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

func (r *RolloutCampaignRepo) Create(ctx context.Context, campaign domain.RolloutCampaign) (domain.RolloutCampaign, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	var createdCampaign domain.RolloutCampaign
	err := r.tm.Do(reqCtx, func(txCtx context.Context) error {
		exec := r.exec(txCtx)

		campaignQuery := `
		INSERT INTO rollout_campaigns (firmware_version_id, device_model)
		VALUES ($1, $2)
		RETURNING id, firmware_version_id, device_model, status, created_at, started_at, completed_at
		`

		campaignRow := exec.QueryRow(txCtx, campaignQuery, campaign.FirmwareVersionID, campaign.DeviceModel)

		err := campaignRow.Scan(
			&createdCampaign.ID,
			&createdCampaign.FirmwareVersionID,
			&createdCampaign.DeviceModel,
			&createdCampaign.Status,
			&createdCampaign.CreatedAt,
			&createdCampaign.StartedAt,
			&createdCampaign.CompletedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to create rollout campaign: %w", err)
		}

		stagesQuery := `
		WITH inserted AS (
			INSERT INTO rollout_stages (campaign_id, order_index, target_percent, min_sample_size, success_threshold)
			SELECT *
			FROM unnest($1::uuid[], $2::int[], $3::int[], $4::int[], $5::real[])
			RETURNING id, campaign_id, order_index, target_percent, min_sample_size, success_threshold, status, entered_at
		)
		SELECT *
		FROM inserted
		ORDER BY order_index;
		`

		stagesLen := len(campaign.RolloutStages)
		campaignIDs := make([]uuid.UUID, stagesLen)
		orderIndexes := make([]int, stagesLen)
		targetPercents := make([]int, stagesLen)
		minSampleSizes := make([]int, stagesLen)
		successThresholds := make([]float32, stagesLen)

		for i, stage := range campaign.RolloutStages {
			campaignIDs[i] = createdCampaign.ID
			orderIndexes[i] = stage.OrderIndex
			targetPercents[i] = stage.TargetPercent
			minSampleSizes[i] = stage.MinSampleSize
			successThresholds[i] = stage.SuccessThreshold
		}

		rows, err := exec.Query(txCtx, stagesQuery, campaignIDs, orderIndexes, targetPercents, minSampleSizes, successThresholds)

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return domain.ErrRolloutStageAlreadyExists
		} else if err != nil {
			return fmt.Errorf("failed to create rollout stages during campaign creation: %w", err)
		}

		defer rows.Close()

		createdCampaign.RolloutStages = make([]domain.RolloutStage, 0, len(campaign.RolloutStages))

		for rows.Next() {
			var createdStage domain.RolloutStage
			err = rows.Scan(
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
				return fmt.Errorf("failed to scan rollout stage during campaign creation: %w", err)
			}

			createdCampaign.RolloutStages = append(createdCampaign.RolloutStages, createdStage)
		}

		if err = rows.Err(); err != nil {
			return fmt.Errorf("failed to read stages during campaign creation: %w", err)
		}

		return nil
	})

	return createdCampaign, err
}

func (r *RolloutCampaignRepo) Start(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	var campaign domain.RolloutCampaign
	err := r.tm.Do(reqCtx, func(txCtx context.Context) error {
		exec := r.exec(txCtx)

		campaignQuery := `
		UPDATE rollout_campaigns SET status = 'running', started_at = now()
		WHERE id = $1 AND status = 'draft'
		RETURNING id, firmware_version_id, device_model, status, created_at, started_at, completed_at
		`

		row := exec.QueryRow(txCtx, campaignQuery, id)

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
			return domain.ErrCampaignAlreadyRunning
		} else if err == pgx.ErrNoRows {
			return r.checkCampaignConflictError(txCtx, id)
		} else if err != nil {
			return fmt.Errorf("failed to start rollout campaign: %w", err)
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

		_, err = exec.Exec(txCtx, stageQuery, id)
		if err != nil {
			return fmt.Errorf("failed to update stage: %w", err)
		}

		return nil
	})
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	stages, err := r.listRolloutStagesByCampaignID(reqCtx, campaign.ID)
	if err != nil {
		return domain.RolloutCampaign{}, fmt.Errorf("failed to get stages for campaign: %w", err)
	}

	campaign.RolloutStages = stages
	return campaign, nil
}

func (r *RolloutCampaignRepo) Pause(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	exec := r.exec(reqCtx)

	query := `
	UPDATE rollout_campaigns SET status = 'paused'
	WHERE id = $1 AND status = 'running'
	RETURNING id, firmware_version_id, device_model, status, created_at, started_at, completed_at
	`

	row := exec.QueryRow(reqCtx, query, id)

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
		return domain.RolloutCampaign{}, r.checkCampaignConflictError(reqCtx, id)
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

func (r *RolloutCampaignRepo) Resume(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	exec := r.exec(reqCtx)

	query := `
	UPDATE rollout_campaigns SET status = 'running'
	WHERE id = $1 AND status = 'paused'
	RETURNING id, firmware_version_id, device_model, status, created_at, started_at, completed_at
	`

	row := exec.QueryRow(reqCtx, query, id)

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
		return domain.RolloutCampaign{}, r.checkCampaignConflictError(reqCtx, id)
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

func (r *RolloutCampaignRepo) FindRunning(ctx context.Context, deviceModel string) (domain.RolloutCampaign, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	exec := r.exec(reqCtx)

	query := `
	SELECT id, firmware_version_id, device_model, status, created_at, started_at, completed_at
	FROM rollout_campaigns
	WHERE device_model = $1 AND status = 'running'
	`

	row := exec.QueryRow(reqCtx, query, deviceModel)

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
		return domain.RolloutCampaign{}, r.checkCampaignStatusError(reqCtx, deviceModel)
	} else if err != nil {
		return domain.RolloutCampaign{}, fmt.Errorf("failed to get running rollout campaign: %w", err)
	}
	return campaign, nil
}

func (r *RolloutCampaignRepo) ListRunning(ctx context.Context) ([]domain.RolloutCampaign, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	exec := r.exec(reqCtx)

	query := `
	SELECT id, firmware_version_id, device_model, status, created_at, started_at, completed_at
	FROM rollout_campaigns
	WHERE status = 'running'
	ORDER BY created_at DESC
	`

	rows, err := exec.Query(reqCtx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list running rollout campaigns: %w", err)
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
		return nil, fmt.Errorf("failed to read running rollout campaigns: %w", err)
	}

	return rolloutCampaigns, nil
}

func (r *RolloutCampaignRepo) FindActiveStages(ctx context.Context, campaignIDs []uuid.UUID) ([]domain.RolloutStage, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	exec := r.exec(reqCtx)

	query := `
	SELECT id, campaign_id, order_index, target_percent, min_sample_size, success_threshold, status, entered_at
	FROM rollout_stages
	WHERE campaign_id = ANY($1) AND status = 'active'
	`

	rows, err := exec.Query(reqCtx, query, campaignIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to list rollout campaigns: %w", err)
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
			return nil, fmt.Errorf("failed to scan active rollout stage: %w", err)
		}

		stages = append(stages, stage)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read active rollout stages: %w", err)
	}

	return stages, nil
}

func (r *RolloutCampaignRepo) AdvanceStage(ctx context.Context, campaignID uuid.UUID, prevStageID uuid.UUID) (domain.RolloutCampaign, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	var result domain.RolloutCampaign
	err := r.tm.Do(reqCtx, func(txCtx context.Context) error {
		exec := r.exec(txCtx)

		stageQuery := `
		SELECT s.order_index, s.status, c.status
		FROM rollout_stages s
		JOIN rollout_campaigns c ON c.id = s.campaign_id
		WHERE s.id = $1 AND s.campaign_id = $2
		FOR UPDATE
		`

		stageRow := exec.QueryRow(txCtx, stageQuery, prevStageID, campaignID)

		var stage domain.RolloutStage
		var campaignStatus domain.RolloutCampaignsStatus
		err := stageRow.Scan(
			&stage.OrderIndex,
			&stage.Status,
			&campaignStatus,
		)

		if err == pgx.ErrNoRows {
			return domain.ErrRolloutStageNotFoundInCampaign
		} else if err != nil {
			return fmt.Errorf("failed to get stage: %w", err)
		}

		if campaignStatus != domain.RolloutCampaignsStatusRunning {
			return fmt.Errorf("%w: can't advance %s campaign", domain.ErrRolloutCampaignWrongStatus, campaignStatus)
		} else if stage.Status != domain.RolloutStagesStatusActive {
			return fmt.Errorf("%w: can't advance from %s stage", domain.ErrRolloutStageWrongStatus, stage.Status)
		}

		passedQuery := `
		UPDATE rollout_stages
		SET status = 'passed'
		WHERE id = $1
		`

		_, err = exec.Exec(txCtx, passedQuery, prevStageID)
		if err != nil {
			return fmt.Errorf("failed to change stage to passed: %w", err)
		}

		activeQuery := `
		UPDATE rollout_stages
		SET status = 'active', entered_at = now()
		WHERE status = 'pending' AND campaign_id = $1 AND order_index = $2
		`

		ct, err := exec.Exec(txCtx, activeQuery, campaignID, stage.OrderIndex+1)
		if err != nil {
			return fmt.Errorf("failed to change next stage to active: %w", err)
		}
		if ct.RowsAffected() == 0 {
			campaignQuery := `
			UPDATE rollout_campaigns
			SET status = 'completed', completed_at = now()
			WHERE id = $1
			`

			_, err = exec.Exec(txCtx, campaignQuery, campaignID)
			if err != nil {
				return fmt.Errorf("failed to complete rollout campaign: %w", err)
			}
		}

		campaign, err := r.Get(txCtx, campaignID)
		if err != nil {
			return err
		}
		result = campaign
		return nil
	})
	if err != nil {
		return domain.RolloutCampaign{}, err
	}
	return result, nil
}

func (r *RolloutCampaignRepo) Rollback(ctx context.Context, campaignID uuid.UUID, prevStageID uuid.UUID) (domain.RolloutCampaign, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	var result domain.RolloutCampaign
	err := r.tm.Do(reqCtx, func(txCtx context.Context) error {
		exec := r.exec(txCtx)

		stageQuery := `
		SELECT s.status, c.status
		FROM rollout_stages s
		JOIN rollout_campaigns c ON c.id = s.campaign_id
		WHERE s.id = $1 AND s.campaign_id = $2
		FOR UPDATE
		`

		stageRow := exec.QueryRow(txCtx, stageQuery, prevStageID, campaignID)

		var stageStatus domain.RolloutStagesStatus
		var campaignStatus domain.RolloutCampaignsStatus
		err := stageRow.Scan(
			&stageStatus,
			&campaignStatus,
		)

		if err == pgx.ErrNoRows {
			return domain.ErrRolloutStageNotFoundInCampaign
		} else if err != nil {
			return fmt.Errorf("failed to get stage: %w", err)
		}

		if !(campaignStatus == domain.RolloutCampaignsStatusRunning || campaignStatus == domain.RolloutCampaignsStatusPaused) {
			return fmt.Errorf("%w: can't rollback %s campaign", domain.ErrRolloutCampaignWrongStatus, campaignStatus)
		} else if stageStatus != domain.RolloutStagesStatusActive {
			return fmt.Errorf("%w: can't rollback from %s stage", domain.ErrRolloutStageWrongStatus, stageStatus)
		}

		failedQuery := `
		UPDATE rollout_stages
		SET status = 'failed'
		WHERE id = $1
		`

		_, err = exec.Exec(txCtx, failedQuery, prevStageID)
		if err != nil {
			return fmt.Errorf("failed to change stage to failed: %w", err)
		}

		rollbackQuery := `
		UPDATE rollout_campaigns
		SET status = 'rolled_back', completed_at = now()
		WHERE id = $1
		`

		_, err = exec.Exec(txCtx, rollbackQuery, campaignID)
		if err != nil {
			return fmt.Errorf("failed to change campaign to rolled_back: %w", err)
		}

		campaign, err := r.Get(txCtx, campaignID)
		if err != nil {
			return err
		}
		result = campaign
		return nil
	})
	if err != nil {
		return domain.RolloutCampaign{}, err
	}
	return result, nil
}

func (r *RolloutCampaignRepo) listRolloutStagesByCampaignID(ctx context.Context, campaignID uuid.UUID) ([]domain.RolloutStage, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	exec := r.exec(reqCtx)

	query := `
	SELECT id, campaign_id, order_index, target_percent, min_sample_size, success_threshold, status, entered_at
	FROM rollout_stages
	WHERE campaign_id = $1
	ORDER BY order_index
	`

	rows, err := exec.Query(reqCtx, query, campaignID)
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

// Без этого при параллельных запросах к одной кампании все кроме
// первого получат Not Found, из-за проверки статуса внутри query
func (r *RolloutCampaignRepo) checkCampaignConflictError(ctx context.Context, id uuid.UUID) error {
	exec := r.exec(ctx)

	var exists bool
	query := `
	SELECT EXISTS (
		SELECT * FROM rollout_campaigns
		WHERE id = $1
	)
	`
	row := exec.QueryRow(ctx, query, id)
	err := row.Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check campaign existence: %w", err)
	}
	if exists {
		return domain.ErrRolloutCampaignWrongStatus
	}

	return domain.ErrRolloutCampaignNotFound
}

func (r *RolloutCampaignRepo) checkCampaignStatusError(ctx context.Context, deviceModel string) error {
	exec := r.exec(ctx)

	var exists bool
	query := `
	SELECT EXISTS (
		SELECT * FROM rollout_campaigns
		WHERE device_model = $1
	)
	`
	row := exec.QueryRow(ctx, query, deviceModel)
	err := row.Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check campaign existence: %w", err)
	}
	if exists {
		return domain.ErrRolloutCampaignWrongStatus
	}

	return domain.ErrRolloutCampaignNotFound
}
