package admin

import (
	"context"
	"fmt"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
)

type RolloutCampaignRepo interface {
	Get(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
}

type CampaignCacheRepo interface {
	SetCheckinData(ctx context.Context, id uuid.UUID, data domain.CampaignCheckinData) error
}

type RolloutController interface {
	ForceRollback(ctx context.Context, campaignID uuid.UUID) error
}

type AdminService struct {
	campaignRepo  RolloutCampaignRepo
	campaignCache CampaignCacheRepo
	controller    RolloutController
}

func NewAdminService(campaignRepo RolloutCampaignRepo, campaignCache CampaignCacheRepo, controller RolloutController) *AdminService {
	return &AdminService{
		campaignRepo:  campaignRepo,
		campaignCache: campaignCache,
		controller:    controller,
	}
}

func (s *AdminService) ForceRollback(ctx context.Context, campaignID uuid.UUID) error {
	campaign, err := s.campaignRepo.Get(ctx, campaignID)
	if err != nil {
		return err
	}

	if !(campaign.Status == domain.RolloutCampaignsStatusRunning || campaign.Status == domain.RolloutCampaignsStatusPaused) {
		return fmt.Errorf("%w: can't rollback %s campaign", domain.ErrRolloutCampaignWrongStatus, campaign.Status)
	}

	if campaign.Status == domain.RolloutCampaignsStatusPaused {
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
			return fmt.Errorf("no active stage in %s campaign", campaign.Status)
		}

		data := domain.CampaignCheckinData{
			StageID:       activeStage.ID,
			TargetPercent: activeStage.TargetPercent,
		}

		err = s.campaignCache.SetCheckinData(ctx, campaignID, data)
		if err != nil {
			return err
		}
	}

	return s.controller.ForceRollback(ctx, campaignID)
}
