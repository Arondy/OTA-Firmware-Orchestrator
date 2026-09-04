package campaign

import (
	"context"
	"errors"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
)

func (s *RolloutCampaignService) setCampaignStageCache(ctx context.Context, stage domain.RolloutStage) error {
	campaignID := stage.CampaignID
	logger := config.LoggerFromContext(ctx)
	logger = logger.With("campaign_id", campaignID, "stage_id", stage.ID)

	var joinedErr error

	err := s.campaignCache.SetCheckinData(ctx, campaignID, domain.CampaignCheckinData{
		StageID:       stage.ID,
		TargetPercent: stage.TargetPercent,
	})
	if err != nil {
		logger.Warnw("failed to put checkin data in campaignCache", "error", err)
		joinedErr = errors.Join(joinedErr, err)
	}

	err = s.campaignCache.AddRunningCampaigns(ctx, campaignID)
	if err != nil {
		logger.Warnw("failed to add running campaign in campaignCache", "error", err)
		joinedErr = errors.Join(joinedErr, err)
	}

	err = s.stageCache.SetStageStats(ctx, stage.ID, domain.StageStats{
		MinSampleSize:    stage.MinSampleSize,
		SuccessThreshold: stage.SuccessThreshold,
	})
	if err != nil {
		logger.Warnw("failed to put stage stats in stageCache", "error", err)
		joinedErr = errors.Join(joinedErr, err)
	}

	return joinedErr
}
