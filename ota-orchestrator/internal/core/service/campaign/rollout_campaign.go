package campaign

import (
	"context"
	"fmt"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/domain"
	"github.com/google/uuid"
)

type RolloutCampaignRepo interface {
	List(ctx context.Context) ([]domain.RolloutCampaign, error)
	Get(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	Create(ctx context.Context, campaign domain.RolloutCampaign) (domain.RolloutCampaign, error)
	Start(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	Pause(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	Resume(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
}

type FirmwareVersionRepo interface {
	Get(ctx context.Context, id uuid.UUID) (domain.FirmwareVersion, error)
}

type RolloutCampaignService struct {
	campaignRepo RolloutCampaignRepo
	firmwareRepo FirmwareVersionRepo
}

func NewService(campaignRepo RolloutCampaignRepo, firmwareRepo FirmwareVersionRepo) *RolloutCampaignService {
	return &RolloutCampaignService{
		campaignRepo: campaignRepo,
		firmwareRepo: firmwareRepo,
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

	return s.campaignRepo.Start(ctx, id)
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
