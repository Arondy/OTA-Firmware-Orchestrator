package service

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

type RolloutCampaignService struct {
	campaignRepo RolloutCampaignRepo
	firmwareRepo FirmwareVersionRepo
}

func NewRolloutCampaignService(campaignRepo RolloutCampaignRepo, firmwareRepo FirmwareVersionRepo) *RolloutCampaignService {
	return &RolloutCampaignService{
		campaignRepo: campaignRepo,
		firmwareRepo: firmwareRepo,
	}
}

func (s *RolloutCampaignService) ListRolloutCampaigns(ctx context.Context) ([]domain.RolloutCampaign, error) {
	return s.campaignRepo.ListRolloutCampaigns(ctx)
}

func (s *RolloutCampaignService) GetRolloutCampaign(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	return s.campaignRepo.GetRolloutCampaign(ctx, id)
}

func (s *RolloutCampaignService) CreateRolloutCampaign(ctx context.Context, campaign domain.RolloutCampaign) (domain.RolloutCampaign, error) {
	fw, err := s.firmwareRepo.GetFirmwareVersion(ctx, campaign.FirmwareVersionID)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	campaign.DeviceModel = fw.DeviceModel
	return s.campaignRepo.CreateRolloutCampaign(ctx, campaign)
}

func (s *RolloutCampaignService) StartRolloutCampaign(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	campaign, err := s.campaignRepo.GetRolloutCampaign(ctx, id)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	if campaign.Status != domain.RolloutCampaignsStatusDraft {
		return domain.RolloutCampaign{}, fmt.Errorf("can't start %s campaign", campaign.Status)
	}

	return s.campaignRepo.StartRolloutCampaign(ctx, id)
}

func (s *RolloutCampaignService) PauseRolloutCampaign(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	campaign, err := s.campaignRepo.GetRolloutCampaign(ctx, id)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	if campaign.Status != domain.RolloutCampaignsStatusRunning {
		return domain.RolloutCampaign{}, fmt.Errorf("can't pause %s campaign", campaign.Status)
	}

	return s.campaignRepo.PauseRolloutCampaign(ctx, id)
}

func (s *RolloutCampaignService) ResumeRolloutCampaign(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	campaign, err := s.campaignRepo.GetRolloutCampaign(ctx, id)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	if campaign.Status != domain.RolloutCampaignsStatusPaused {
		return domain.RolloutCampaign{}, fmt.Errorf("can't resume %s campaign", campaign.Status)
	}

	return s.campaignRepo.ResumeRolloutCampaign(ctx, id)
}
