package results

import (
	"context"

	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/domain"
	"github.com/google/uuid"
)

type CampaignRepo interface {
	UpdateStageResults(ctx context.Context, event domain.UpdateResultsEvent) (int, error)
	GetCampaignStats(ctx context.Context, id uuid.UUID) (domain.CampaignStats, error)
}

type CampaignStatsService struct {
	campaignRepo CampaignRepo
}

func NewCampaignStatsService(campaignRepo CampaignRepo) *CampaignStatsService {
	return &CampaignStatsService{campaignRepo: campaignRepo}
}

func (s *CampaignStatsService) UpdateStageResults(ctx context.Context, event domain.UpdateResultsEvent) (int, error) {
	return s.campaignRepo.UpdateStageResults(ctx, event)
}

func (s *CampaignStatsService) GetCampaignStats(ctx context.Context, id uuid.UUID) (domain.CampaignStats, error) {
	return s.campaignRepo.GetCampaignStats(ctx, id)
}
