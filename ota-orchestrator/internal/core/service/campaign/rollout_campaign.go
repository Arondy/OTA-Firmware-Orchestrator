package campaign

import (
	"context"
	"fmt"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/domain"
	"github.com/google/uuid"
)

type RolloutCampaignRepo interface {
	ListRolloutCampaigns(ctx context.Context) ([]domain.RolloutCampaign, error)
	GetRolloutCampaign(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	CreateRolloutCampaign(ctx context.Context, campaign domain.RolloutCampaign) (domain.RolloutCampaign, error)
	StartRolloutCampaign(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	PauseRolloutCampaign(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	ResumeRolloutCampaign(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
}

type FirmwareVersionRepo interface {
	GetFirmwareVersion(ctx context.Context, id uuid.UUID) (domain.FirmwareVersion, error)
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
	return s.campaignRepo.ListRolloutCampaigns(ctx)
}

func (s *RolloutCampaignService) Get(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	return s.campaignRepo.GetRolloutCampaign(ctx, id)
}

func (s *RolloutCampaignService) Create(ctx context.Context, campaign domain.RolloutCampaign) (domain.RolloutCampaign, error) {
	fw, err := s.firmwareRepo.GetFirmwareVersion(ctx, campaign.FirmwareVersionID)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	campaign.DeviceModel = fw.DeviceModel
	return s.campaignRepo.CreateRolloutCampaign(ctx, campaign)
}

func (s *RolloutCampaignService) Start(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	campaign, err := s.campaignRepo.GetRolloutCampaign(ctx, id)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	if campaign.Status != domain.RolloutCampaignsStatusDraft {
		return domain.RolloutCampaign{}, fmt.Errorf("%w: can't start %s campaign", domain.ErrRolloutCampaignWrongStatus, campaign.Status)
	}

	return s.campaignRepo.StartRolloutCampaign(ctx, id)
}

func (s *RolloutCampaignService) Pause(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	campaign, err := s.campaignRepo.GetRolloutCampaign(ctx, id)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	if campaign.Status != domain.RolloutCampaignsStatusRunning {
		return domain.RolloutCampaign{}, fmt.Errorf("%w: can't pause %s campaign", domain.ErrRolloutCampaignWrongStatus, campaign.Status)
	}

	return s.campaignRepo.PauseRolloutCampaign(ctx, id)
}

func (s *RolloutCampaignService) Resume(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	campaign, err := s.campaignRepo.GetRolloutCampaign(ctx, id)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	if campaign.Status != domain.RolloutCampaignsStatusPaused {
		return domain.RolloutCampaign{}, fmt.Errorf("%w: can't resume %s campaign", domain.ErrRolloutCampaignWrongStatus, campaign.Status)
	}

	return s.campaignRepo.ResumeRolloutCampaign(ctx, id)
}
