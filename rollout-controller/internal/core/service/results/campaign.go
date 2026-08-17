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
	campaingRepo CampaignRepo
}

func NewCampaignStatsService(campaingRepo CampaignRepo) *CampaignStatsService {
	return &CampaignStatsService{campaingRepo: campaingRepo}
}

func (s *CampaignStatsService) UpdateStageResults(ctx context.Context, event domain.UpdateResultsEvent) (int, error) {
	return s.campaingRepo.UpdateStageResults(ctx, event)
}

func (s *CampaignStatsService) GetCampaignStats(ctx context.Context, id uuid.UUID) (domain.CampaignStats, error) {
	return s.campaingRepo.GetCampaignStats(ctx, id)
}
