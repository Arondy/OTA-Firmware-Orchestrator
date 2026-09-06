package campaign

import (
	"context"
	"errors"
	"fmt"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
)

func (s *RolloutCampaignService) ApplyDecision(ctx context.Context, decision domain.DecisionEvent) error {
	logger := config.LoggerFromContext(ctx)
	logger = logger.With("decision_id", decision.DecisionID, "previous_stage_id", decision.PreviousStageID)

	isAdvance := decision.DecisionType == domain.DecisionTypeAdvance
	var committedCampaign *domain.RolloutCampaign
	var shouldApplyCache bool

	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		_, err := s.decisionRepo.Get(txCtx, decision.DecisionID)
		if err == nil {
			logger.Debug("duplicate decision wasn't applied")
			return nil
		} else if !errors.Is(err, domain.ErrAppliedDecisionNotFound) {
			return err
		}

		_, err = s.decisionRepo.Create(txCtx, domain.AppliedDecisionFromEvent(decision))
		if err != nil {
			return err
		}

		if isAdvance {
			campaign, err := s.advanceStage(txCtx, decision.CampaignID, decision.PreviousStageID)
			if errors.Is(err, domain.ErrRolloutStageWrongStatus) || errors.Is(err, domain.ErrRolloutCampaignWrongStatus) {
				logger.Debugw("duplicate decision wasn't applied", "error", err)
				return nil
			} else if err != nil {
				return err
			}
			committedCampaign = &campaign
		} else {
			_, err := s.rollbackStage(txCtx, decision.CampaignID, decision.PreviousStageID)
			if errors.Is(err, domain.ErrRolloutStageWrongStatus) || errors.Is(err, domain.ErrRolloutCampaignWrongStatus) {
				logger.Debugw("duplicate decision wasn't applied", "error", err)
				return nil
			} else if err != nil {
				return err
			}
		}

		shouldApplyCache = true
		return nil
	})
	if err != nil {
		return err
	}
	if !shouldApplyCache {
		return nil
	}

	if isAdvance {
		if committedCampaign != nil {
			s.handleAdvanceCache(ctx, *committedCampaign, decision.PreviousStageID)
		}
	} else {
		s.campaignCache.RemoveRunningCampaigns(ctx, decision.CampaignID)
		if err := s.campaignCache.DeleteCheckinData(ctx, decision.CampaignID); err != nil {
			logger.Warnw("failed to delete checkin data from campaignCache", "error", err)
		}
		if err := s.stageCache.DeleteStageStats(ctx, decision.PreviousStageID); err != nil {
			logger.Warnw("failed to delete stage stats from stageCache", "error", err)
		}
	}

	return nil
}

func (s *RolloutCampaignService) advanceStage(ctx context.Context, campaignID uuid.UUID, prevStageID uuid.UUID) (domain.RolloutCampaign, error) {
	campaign, err := s.campaignRepo.Get(ctx, campaignID)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	if campaign.Status != domain.RolloutCampaignsStatusRunning {
		return domain.RolloutCampaign{}, fmt.Errorf("%w: can't advance %s campaign", domain.ErrRolloutCampaignWrongStatus, campaign.Status)
	}

	return s.campaignRepo.AdvanceStage(ctx, campaignID, prevStageID)
}

func (s *RolloutCampaignService) rollbackStage(ctx context.Context, campaignID uuid.UUID, prevStageID uuid.UUID) (domain.RolloutCampaign, error) {
	campaign, err := s.campaignRepo.Get(ctx, campaignID)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	if !(campaign.Status == domain.RolloutCampaignsStatusRunning || campaign.Status == domain.RolloutCampaignsStatusPaused) {
		return domain.RolloutCampaign{}, fmt.Errorf("%w: can't rollback %s campaign", domain.ErrRolloutCampaignWrongStatus, campaign.Status)
	}

	return s.campaignRepo.Rollback(ctx, campaignID, prevStageID)
}

func (s *RolloutCampaignService) handleAdvanceCache(ctx context.Context, campaign domain.RolloutCampaign, prevStageID uuid.UUID) {
	if campaign.Status == domain.RolloutCampaignsStatusCompleted {
		s.campaignCache.RemoveRunningCampaigns(ctx, campaign.ID)
		if err := s.stageCache.DeleteStageStats(ctx, prevStageID); err != nil {
			config.LoggerFromContext(ctx).Warnw("failed to delete stage stats from stageCache", "error", err)
		}
		if err := s.campaignCache.DeleteCheckinData(ctx, campaign.ID); err != nil {
			config.LoggerFromContext(ctx).Warnw("failed to delete checkin data from campaignCache", "error", err)
		}
		return
	}

	var activeStage domain.RolloutStage
	var found bool
	for _, stage := range campaign.RolloutStages {
		if stage.Status == domain.RolloutStagesStatusActive {
			activeStage = stage
			found = true
			break
		}
	}

	if !found {
		logger := config.LoggerFromContext(ctx)
		logger.Warnw("failed to find active stage to put in campaignCache", "campaign_id", campaign.ID)
		return
	}

	if err := s.stageCache.DeleteStageStats(ctx, prevStageID); err != nil {
		config.LoggerFromContext(ctx).Warnw("failed to delete stage stats from stageCache", "error", err)
	}
	s.setCampaignStageCache(ctx, activeStage)
}
