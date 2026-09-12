package campaign

import (
	"context"
	"errors"
	"fmt"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type RolloutCampaignRepo interface {
	List(ctx context.Context, pagination domain.Pagination) ([]domain.RolloutCampaign, error)
	Get(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	Create(ctx context.Context, campaign domain.RolloutCampaign) (domain.RolloutCampaign, error)
	Start(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	Pause(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	Resume(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	AdvanceStage(ctx context.Context, campaignID uuid.UUID, prevStageID uuid.UUID) (domain.RolloutCampaign, error)
	Rollback(ctx context.Context, campaignID uuid.UUID, prevStageID uuid.UUID) (domain.RolloutCampaign, error)
	ListRunning(ctx context.Context) ([]domain.RolloutCampaign, error)
	FindActiveStages(ctx context.Context, campaignIDs []uuid.UUID) ([]domain.RolloutStage, error)
}

type FirmwareVersionRepo interface {
	Get(ctx context.Context, id uuid.UUID) (domain.FirmwareVersion, error)
}

type AppliedDecisionRepo interface {
	Get(ctx context.Context, decisionID uuid.UUID) (domain.AppliedDecision, error)
	Create(ctx context.Context, decision domain.AppliedDecision) (domain.AppliedDecision, error)
}

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type CampaignCacheRepo interface {
	SetCheckinData(ctx context.Context, id uuid.UUID, data domain.CampaignCheckinData) error
	DeleteCheckinData(ctx context.Context, id uuid.UUID) error
	AddRunningCampaigns(ctx context.Context, ids ...uuid.UUID) error
	RemoveRunningCampaigns(ctx context.Context, ids ...uuid.UUID) error
	DeleteAllRunningCampaigns(ctx context.Context) error
}

type StageCacheRepo interface {
	SetStageStats(ctx context.Context, id uuid.UUID, stats domain.StageStats) error
	DeleteStageStats(ctx context.Context, id uuid.UUID) error
}

type RolloutController interface {
	GetCampaignStats(ctx context.Context, id uuid.UUID) (domain.RolloutCampaignStats, error)
}

type RolloutCampaignService struct {
	campaignRepo  RolloutCampaignRepo
	firmwareRepo  FirmwareVersionRepo
	decisionRepo  AppliedDecisionRepo
	txManager     TxManager
	campaignCache CampaignCacheRepo
	stageCache    StageCacheRepo
	controller    RolloutController
}

func NewService(campaignRepo RolloutCampaignRepo, firmwareRepo FirmwareVersionRepo, decisionRepo AppliedDecisionRepo, txManager TxManager, campaignCache CampaignCacheRepo, stageCache StageCacheRepo, controller RolloutController) *RolloutCampaignService {
	return &RolloutCampaignService{
		campaignRepo:  campaignRepo,
		firmwareRepo:  firmwareRepo,
		decisionRepo:  decisionRepo,
		txManager:     txManager,
		campaignCache: campaignCache,
		stageCache:    stageCache,
		controller:    controller,
	}
}

func (s *RolloutCampaignService) List(ctx context.Context, pagination domain.Pagination) ([]domain.RolloutCampaign, error) {
	return s.campaignRepo.List(ctx, pagination)
}

func (s *RolloutCampaignService) Get(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	campaign, err := s.campaignRepo.Get(ctx, id)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	stats, err := s.controller.GetCampaignStats(ctx, id)
	if err != nil {
		logger := config.LoggerFromContext(ctx)
		logger.Warnw("failed to get campaign status", "error", err)
	} else {
		campaign.Stats = &stats
	}

	return campaign, nil
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

	// не должно быть возможным
	if len(startedCampaign.RolloutStages) == 0 {
		return startedCampaign, fmt.Errorf("campaign %s has no stages to start", id)
	}

	s.setCampaignStageCache(ctx, startedCampaign.RolloutStages[0])

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

	pausedCampaign, err := s.campaignRepo.Pause(ctx, id)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	// нет очистки остального кэша т.к. считаем что кампании не будут "забрасываться"
	// и нет проблем с ForceRollback, если после отправки запроса на него (когда кэш выставлен), но до обработки, делать Pause
	s.campaignCache.RemoveRunningCampaigns(ctx, id)

	return pausedCampaign, nil
}

func (s *RolloutCampaignService) Resume(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error) {
	campaign, err := s.campaignRepo.Get(ctx, id)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	if campaign.Status != domain.RolloutCampaignsStatusPaused {
		return domain.RolloutCampaign{}, fmt.Errorf("%w: can't resume %s campaign", domain.ErrRolloutCampaignWrongStatus, campaign.Status)
	}

	campaign, err = s.campaignRepo.Resume(ctx, id)
	if err != nil {
		return domain.RolloutCampaign{}, err
	}

	logger := config.LoggerFromContext(ctx)

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
		logger.Warnw("resume: failed to find active stage to put in campaignCache", "campaign_id", id)
		return campaign, nil
	}

	s.setCampaignStageCache(ctx, activeStage)

	return campaign, nil
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

	var joinedErr error

	err = s.campaignCache.DeleteAllRunningCampaigns(ctx)
	joinedErr = errors.Join(joinedErr, err)

	for _, stage := range stages {
		err = s.setCampaignStageCache(ctx, stage)
		joinedErr = errors.Join(joinedErr, err)
	}

	logger.Infow("finished cache warmup", "campaigns_processed", len(campaigns))
	return joinedErr
}
