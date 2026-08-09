package campaign

import (
	"context"
	"fmt"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type RolloutCampaignRepo interface {
	List(ctx context.Context) ([]domain.RolloutCampaign, error)
	Get(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	Create(ctx context.Context, campaign domain.RolloutCampaign) (domain.RolloutCampaign, error)
	Start(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	Pause(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	Resume(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	AdvanceStage(ctx context.Context, campaignID uuid.UUID) (domain.RolloutCampaign, error)
	ListRunning(ctx context.Context) ([]domain.RolloutCampaign, error)
	FindActiveStages(ctx context.Context, campaignIDs []uuid.UUID) ([]domain.RolloutStage, error)
}

type FirmwareVersionRepo interface {
	Get(ctx context.Context, id uuid.UUID) (domain.FirmwareVersion, error)
}

type CampaignCacheRepo interface {
	SetCurrentStage(ctx context.Context, id uuid.UUID, stageID uuid.UUID) error
	DeleteCurrentStage(ctx context.Context, id uuid.UUID) error
	SetCurrentTargetPercent(ctx context.Context, id uuid.UUID, percent int) error
	DeleteCurrentTargetPercent(ctx context.Context, id uuid.UUID) error
}

type RolloutCampaignService struct {
	campaignRepo RolloutCampaignRepo
	firmwareRepo FirmwareVersionRepo
	cache        CampaignCacheRepo
}

func NewService(campaignRepo RolloutCampaignRepo, firmwareRepo FirmwareVersionRepo, cache CampaignCacheRepo) *RolloutCampaignService {
	return &RolloutCampaignService{
		campaignRepo: campaignRepo,
		firmwareRepo: firmwareRepo,
		cache:        cache,
	}
}

func (s *RolloutCampaignService) List(ctx context.Context) ([]domain.RolloutCampaign, error) {
	return s.campaignRepo.List(ctx)
}

func (s *RolloutCampaignService) Get(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	return s.campaignRepo.Get(ctx, id)
}

func (s *RolloutCampaignService) Create(ctx context.Context, campaign domain.RolloutCampaign) (domain.RolloutCampaign, error) {
	fw, err := s.firmwareRepo.Get(ctx, campaign.FirmwareVersionID)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	campaign.DeviceModel = fw.DeviceModel
	return s.campaignRepo.Create(ctx, campaign)
}

func (s *RolloutCampaignService) Start(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	campaign, err := s.campaignRepo.Get(ctx, id)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	if campaign.Status != domain.RolloutCampaignsStatusDraft {
		return domain.RolloutCampaign{}, fmt.Errorf("%w: can't start %s campaign", domain.ErrRolloutCampaignWrongStatus, campaign.Status)
	}

	startedCampaign, err := s.campaignRepo.Start(ctx, id)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	logger := config.LoggerFromContext(ctx)

	// не должно быть возможным
	if len(startedCampaign.RolloutStages) == 0 {
		return startedCampaign, fmt.Errorf("campaign %s has no stages to start", id)
	}

	err = s.cache.SetCurrentStage(ctx, id, startedCampaign.RolloutStages[0].ID)
	if err != nil {
		logger.Warnw("failed to put current stage in cache", "error", err, "campaign_id", id)
	}

	err = s.cache.SetCurrentTargetPercent(ctx, id, startedCampaign.RolloutStages[0].TargetPercent)
	if err != nil {
		logger.Warnw("failed to put current target percent in cache", "error", err, "campaign_id", id)
	}

	return startedCampaign, nil
}

func (s *RolloutCampaignService) Pause(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	campaign, err := s.campaignRepo.Get(ctx, id)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	if campaign.Status != domain.RolloutCampaignsStatusRunning {
		return domain.RolloutCampaign{}, fmt.Errorf("%w: can't pause %s campaign", domain.ErrRolloutCampaignWrongStatus, campaign.Status)
	}

	return s.campaignRepo.Pause(ctx, id)
}

func (s *RolloutCampaignService) Resume(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	campaign, err := s.campaignRepo.Get(ctx, id)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	if campaign.Status != domain.RolloutCampaignsStatusPaused {
		return domain.RolloutCampaign{}, fmt.Errorf("%w: can't resume %s campaign", domain.ErrRolloutCampaignWrongStatus, campaign.Status)
	}

	return s.campaignRepo.Resume(ctx, id)
}

func (s *RolloutCampaignService) AdvanceStage(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	campaign, err := s.campaignRepo.Get(ctx, id)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	if campaign.Status != domain.RolloutCampaignsStatusRunning {
		return domain.RolloutCampaign{}, fmt.Errorf("%w: can't advance %s campaign", domain.ErrRolloutCampaignWrongStatus, campaign.Status)
	}

	advancedCampaign, err := s.campaignRepo.AdvanceStage(ctx, id)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	logger := config.LoggerFromContext(ctx)

	if advancedCampaign.Status == domain.RolloutCampaignsStatusCompleted {
		err = s.cache.DeleteCurrentStage(ctx, id)
		if err != nil {
			logger.Warnw("failed to delete current stage from cache", "error", err, "campaign_id", id)
		}

		err = s.cache.DeleteCurrentTargetPercent(ctx, id)
		if err != nil {
			logger.Warnw("failed to delete current target percent from cache", "error", err, "campaign_id", id)
		}
	} else {
		var activeStage domain.RolloutStage
		var found bool
		for _, stage := range advancedCampaign.RolloutStages {
			if stage.Status == domain.RolloutStagesStatusActive {
				activeStage = stage
				found = true
				break
			}
		}

		if !found {
			logger.Warnw("failed to find active stage to put in cache", "campaign_id", id)
			return advancedCampaign, nil
		}

		err = s.cache.SetCurrentStage(ctx, id, activeStage.ID)
		if err != nil {
			logger.Warnw("failed to put current stage in cache", "error", err, "campaign_id", id)
		}

		err = s.cache.SetCurrentTargetPercent(ctx, id, activeStage.TargetPercent)
		if err != nil {
			logger.Warnw("failed to put current target percent in cache", "error", err, "campaign_id", id)
		}
	}

	return advancedCampaign, nil
}

func (s *RolloutCampaignService) WarmUpCache(ctx context.Context, logger *zap.SugaredLogger) error {
	campaigns, err := s.campaignRepo.ListRunning(ctx)
	if err != nil {
		return err
	}

	campaignIDs := make([]uuid.UUID, len(campaigns))
	for i, campaign := range campaigns {
		campaignIDs[i] = campaign.ID
	}

	stages, err := s.campaignRepo.FindActiveStages(ctx, campaignIDs)
	if err != nil {
		return err
	}

	for _, stage := range stages {
		err = s.cache.SetCurrentStage(ctx, stage.CampaignID, stage.ID)
		if err != nil {
			logger.Warnw("warmup: failed to set campaign current stage", "error", err, "campaign_id", stage.CampaignID, "stage_id", stage.ID)
		}
		err = s.cache.SetCurrentTargetPercent(ctx, stage.CampaignID, stage.TargetPercent)
		if err != nil {
			logger.Warnw("warmup: failed to set campaign current target percent", "error", err, "campaign_id", stage.CampaignID, "stage_id", stage.ID)
		}
	}

	logger.Infow("finished cache warmup", "campaigns_processed", len(campaigns))
	return err
}
