package campaign

import (
	"context"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
)

func (s *RolloutCampaignService) setCampaignStageCache(ctx context.Context, stage domain.RolloutStage) {
	campaignID := stage.CampaignID
	logger := config.LoggerFromContext(ctx)
	logger = logger.With("campaign_id", campaignID, "stage_id", stage.ID)

	err := s.campaignCache.SetCurrentStage(ctx, campaignID, stage.ID)
	if err != nil {
		logger.Warnw("failed to put current stage in campaignCache", "error", err)
	}

	err = s.campaignCache.SetCurrentTargetPercent(ctx, campaignID, stage.TargetPercent)
	if err != nil {
		logger.Warnw("failed to put current target percent in campaignCache", "error", err)
	}

	err = s.stageCache.SetMinSampleSize(ctx, stage.ID, stage.MinSampleSize)
	if err != nil {
		logger.Warnw("failed to put min sample size in stageCache", "error", err)
	}

	err = s.stageCache.SetSuccessThreshold(ctx, stage.ID, stage.SuccessThreshold)
	if err != nil {
		logger.Warnw("failed to put success threshold in stageCache", "error", err)
	}
}

func (s *RolloutCampaignService) deleteCampaignCache(ctx context.Context, campaignID uuid.UUID, stageID uuid.UUID) {
	logger := config.LoggerFromContext(ctx)
	logger = logger.With("campaign_id", campaignID, "stage_id", stageID)

	err := s.campaignCache.DeleteCurrentStage(ctx, campaignID)
	if err != nil {
		logger.Warnw("failed to delete current stage from campaignCache", "error", err, "campaign_id", campaignID)
	}

	err = s.campaignCache.DeleteCurrentTargetPercent(ctx, campaignID)
	if err != nil {
		logger.Warnw("failed to delete current target percent from campaignCache", "error", err, "campaign_id", campaignID)
	}
}

func (s *RolloutCampaignService) deleteStageCache(ctx context.Context, campaignID uuid.UUID, stageID uuid.UUID) {
	logger := config.LoggerFromContext(ctx)
	logger = logger.With("campaign_id", campaignID, "stage_id", stageID)

	err := s.stageCache.DeleteMinSampleSize(ctx, stageID)
	if err != nil {
		logger.Warnw("failed to delete min sample size from stageCache", "error", err, "stage_id", stageID)
	}

	err = s.stageCache.DeleteSuccessThreshold(ctx, stageID)
	if err != nil {
		logger.Warnw("failed to delete success threshold from stageCache", "error", err, "stage_id", stageID)
	}
}
